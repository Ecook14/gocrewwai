package flows

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	//"time"
)

// Engine orchestrates the execution of a Flow.
type Engine struct {
	Checkpointer *CheckpointManager
	OnStepStream func(token string) // NEW: Global stream listener for all agent nodes in the flow
}

// Run executes a flow from start to finish.
func (e *Engine) Run(ctx context.Context, f *Flow) error {
	finalState, err := e.executeNode(ctx, f, f.Initial, f.State)
	if err != nil {
		return err
	}
	f.State = finalState
	slog.Info("🏁 Flow execution completed", slog.String("flow_id", f.ID))
	return nil
}

func (e *Engine) executeNode(ctx context.Context, f *Flow, nodeID string, currentState State) (State, error) {
	if nodeID == "" {
		return currentState, nil
	}

	node, ok := f.Nodes[nodeID]
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
		nextNodeID = node.Router(currentState)

	case NodeParallel:
		var wg sync.WaitGroup
		mu := sync.Mutex{}
		branchStates := make([]State, 0)
		
		for _, branch := range node.ParallelBranches {
			wg.Add(1)
			go func(bID string) {
				defer wg.Done()
				// Pass a clone to each branch
				resState, bErr := e.executeNode(ctx, f, bID, currentState.Clone())
				if bErr == nil {
					mu.Lock()
					branchStates = append(branchStates, resState)
					mu.Unlock()
				}
			}(branch)
		}
		wg.Wait()
		
		// If the next node is a Reduce node, we pass the branchStates
		if len(node.Next) > 0 {
			reduceNode := f.Nodes[node.Next[0]]
			if reduceNode != nil && reduceNode.Type == NodeReduce && reduceNode.Merge != nil {
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
				results := make([]interface{}, len(source))
				for i, item := range source {
					wg.Add(1)
					go func(idx int, val interface{}) {
						defer wg.Done()
						// Create a temporary state for this item
						itemState := currentState.Clone().(*BaseState)
						itemState.Data["item"] = val
						resState, _ := node.Action(ctx, itemState)
						if rb, ok := resState.(*BaseState); ok {
							results[idx] = rb.Data["result"]
						}
					}(i, item)
				}
				wg.Wait()
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

	// 2. Persist checkpoint
	if e.Checkpointer != nil {
		_ = e.Checkpointer.Save(f.ID, nodeID, nextState)
	}

	return e.executeNode(ctx, f, nextNodeID, nextState)
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
	if err := f.State.FromJSON(checkpoint.Data); err != nil {
		return fmt.Errorf("failed to restore state from checkpoint: %w", err)
	}

	// Find the next node after the checkpointed one
	node, ok := f.Nodes[checkpoint.NodeID]
	if !ok {
		return fmt.Errorf("checkpoint node %s not found in flow", checkpoint.NodeID)
	}

	if len(node.Next) == 0 {
		slog.Info("🏁 Flow was already completed at checkpoint", slog.String("flow_id", f.ID))
		return nil
	}

	// Move flow initial pointer and run
	finalState, err := e.executeNode(ctx, f, node.Next[0], f.State)
	if err != nil {
		return err
	}
	f.State = finalState
	return nil
}
