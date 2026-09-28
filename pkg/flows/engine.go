package flows

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	//"time"
)

// MaxFlowDepth bounds recursive node traversal (cycle guard).
const MaxFlowDepth = 256

// Engine orchestrates the execution of a Flow.
type Engine struct {
	Checkpointer *CheckpointManager
	// OnStepStream is a global stream listener for all agent nodes in the flow.
	// Tokens may contain sensitive model output — callers must redact before logging.
	OnStepStream func(token string) // NEW: Global stream listener for all agent nodes in the flow
}

// Run executes a flow from start to finish.
func (e *Engine) Run(ctx context.Context, f *Flow) error {
	finalState, err := e.executeNode(ctx, f, f.Initial, f.getState(), 0)
	if err != nil {
		return err
	}
	f.setState(finalState)
	slog.Info("🏁 Flow execution completed", slog.String("flow_id", f.ID))
	return nil
}

func (e *Engine) executeNode(ctx context.Context, f *Flow, nodeID string, currentState State, depth int) (State, error) {
	select {
	case <-ctx.Done():
		return currentState, ctx.Err()
	default:
	}
	if depth > MaxFlowDepth {
		return currentState, fmt.Errorf("flow %s: max depth %d exceeded at node %s (cycle guard)", f.ID, MaxFlowDepth, nodeID)
	}
	if nodeID == "" {
		return currentState, nil
	}

	node, ok := f.getNode(nodeID)
	if !ok {
		return currentState, fmt.Errorf("node %s not found in flow %s", nodeID, f.ID)
	}

	slog.Info("📍 Flow Node", slog.String("node_id", node.ID), slog.String("type", string(node.Type)))

	var nextState State = currentState
	var nextNodeID string
	var err error

	switch node.Type {
	case NodeStep:
		nextState, err = node.Action(ctx, currentState)
		if err == nil && len(node.Next) > 0 {
			nextNodeID = node.Next[0]
		}

	case NodeRouter:
		select {
		case <-ctx.Done():
			return currentState, ctx.Err()
		default:
		}
		nextNodeID = node.Router(currentState)

	case NodeParallel:
		var wg sync.WaitGroup
		mu := sync.Mutex{}
		branchStates := make([]State, 0)
		var branchErrs []error

		for _, branch := range node.ParallelBranches {
			wg.Add(1)
			go func(bID string) {
				defer wg.Done()
				// Pass a clone to each branch
				resState, bErr := e.executeNode(ctx, f, bID, currentState.Clone(), depth+1)
				mu.Lock()
				defer mu.Unlock()
				if bErr != nil {
					branchErrs = append(branchErrs, fmt.Errorf("branch %s: %w", bID, bErr))
					return
				}
				branchStates = append(branchStates, resState)
			}(branch)
		}
		wg.Wait()

		// Previously branch errors were silently dropped. If every branch
		// failed, surface the first error instead of running on stale state.
		if len(branchStates) == 0 && len(node.ParallelBranches) > 0 {
			if len(branchErrs) > 0 {
				return currentState, branchErrs[0]
			}
			return currentState, fmt.Errorf("node %s: all parallel branches failed", nodeID)
		}
		if len(branchErrs) > 0 {
			slog.Warn("flow parallel branches partially failed", slog.String("node_id", nodeID), slog.Int("failed", len(branchErrs)))
		}

		// If the next node is a Reduce node, we pass the branchStates
		if len(node.Next) > 0 {
			reduceNode, ok := f.getNode(node.Next[0])
			if ok && reduceNode != nil && reduceNode.Type == NodeReduce && reduceNode.Merge != nil {
				nextState = reduceNode.Merge(branchStates)
				// Skip to the node AFTER reduce if needed,
				// but here we just let the recursion handle it.
				nextNodeID = node.Next[0]
			} else {
				// Default: take the first branch result or keep original
				if len(branchStates) > 0 {
					nextState = branchStates[0]
				}
				nextNodeID = node.Next[0]
			}
		}

	case NodeMap:
		// Map logic: Execute action for each item in source key
		if base, ok := currentState.(*BaseState); ok {
			source, ok := base.Data[node.MapSourceKey].([]interface{})
			if ok {
				var wg sync.WaitGroup
				var mapMu sync.Mutex
				var firstErr error
				failed := 0
				results := make([]interface{}, len(source))
				for i, item := range source {
					wg.Add(1)
					go func(idx int, val interface{}) {
						defer wg.Done()
						// Create a temporary state for this item
						itemState := currentState.Clone().(*BaseState)
						itemState.Data["item"] = val
						resState, err := node.Action(ctx, itemState)
						if err != nil {
							mapMu.Lock()
							failed++
							if firstErr == nil {
								firstErr = fmt.Errorf("map item %d: %w", idx, err)
							}
							mapMu.Unlock()
							return
						}
						if rb, ok := resState.(*BaseState); ok {
							results[idx] = rb.Data["result"]
						}
					}(i, item)
				}
				wg.Wait()
				if failed == len(source) && len(source) > 0 {
					return currentState, fmt.Errorf("node %s: all map items failed: %w", nodeID, firstErr)
				}
				if firstErr != nil {
					slog.Warn("flow map items partially failed", slog.String("node_id", nodeID), slog.Int("failed", failed))
				}
				base.Data[node.MapResultKey] = results
				nextState = base
			}
		}
		if len(node.Next) > 0 {
			nextNodeID = node.Next[0]
		}

	case NodeReduce:
		// Logic is usually handled by Parallel/Map parent
		// but if reached normally, just move forward
		if len(node.Next) > 0 {
			nextNodeID = node.Next[0]
		}
	}

	if err != nil {
		return nextState, fmt.Errorf("node %s failed: %w", nodeID, err)
	}

	// 2. Persist checkpoint (log failures; a failed checkpoint must not
	// silently pass — the next Resume would start from stale state).
	if e.Checkpointer != nil {
		if err := e.Checkpointer.Save(f.ID, nodeID, nextState); err != nil {
			slog.Warn("flow checkpoint save failed", slog.String("flow_id", f.ID), slog.String("node_id", nodeID), slog.String("error", err.Error()))
		}
	}

	return e.executeNode(ctx, f, nextNodeID, nextState, depth+1)
}

// Resume attempts to restart a flow from its last known checkpoint.
func (e *Engine) Resume(ctx context.Context, f *Flow) error {
	if e.Checkpointer == nil {
		return fmt.Errorf("no checkpointer configured for flow %s", f.ID)
	}

	checkpoint, err := e.Checkpointer.Load(f.ID)
	if err != nil {
		return err
	}

	slog.Info("📍 Resuming flow from checkpoint", slog.String("flow_id", f.ID), slog.String("node_id", checkpoint.NodeID))

	// Restore starting state
	state := f.getState()
	if err := state.FromJSON(checkpoint.Data); err != nil {
		return fmt.Errorf("failed to restore state from checkpoint: %w", err)
	}

	// Find the next node after the checkpointed one
	node, ok := f.getNode(checkpoint.NodeID)
	if !ok {
		return fmt.Errorf("checkpoint node %s not found in flow", checkpoint.NodeID)
	}

	if len(node.Next) == 0 {
		slog.Info("🏁 Flow was already completed at checkpoint", slog.String("flow_id", f.ID))
		return nil
	}

	// Move flow initial pointer and run
	finalState, err := e.executeNode(ctx, f, node.Next[0], state, 0)
	if err != nil {
		return err
	}
	f.setState(finalState)
	return nil
}
