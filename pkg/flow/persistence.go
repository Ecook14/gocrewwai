package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// ============================================================
// Flow Persistence
// ============================================================

// FlowPersistence defines the interface for persisting flow state.
type FlowPersistence interface {
	// SaveState persists the current flow state.
	SaveState(ctx context.Context, flowID string, state State) error
	// LoadState restores a previously persisted flow state.
	LoadState(ctx context.Context, flowID string) (State, error)
	// DeleteState removes persisted state for a flow.
	DeleteState(ctx context.Context, flowID string) error
}

// JSONFilePersistence stores flow state as JSON files on disk.
// This is the default persistence backend.
type JSONFilePersistence struct {
	Dir string // Directory to store state files
	mu  sync.Mutex
}

// NewJSONFilePersistence creates a file-based persistence backend.
func NewJSONFilePersistence(dir string) *JSONFilePersistence {
	os.MkdirAll(dir, 0755)
	return &JSONFilePersistence{Dir: dir}
}

func (p *JSONFilePersistence) SaveState(ctx context.Context, flowID string, state State) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal flow state: %w", err)
	}

	path := fmt.Sprintf("%s/%s.json", p.Dir, flowID)
	return os.WriteFile(path, data, 0644)
}

func (p *JSONFilePersistence) LoadState(ctx context.Context, flowID string) (State, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	path := fmt.Sprintf("%s/%s.json", p.Dir, flowID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No prior state
		}
		return nil, fmt.Errorf("failed to read flow state: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal flow state: %w", err)
	}
	return state, nil
}

func (p *JSONFilePersistence) DeleteState(ctx context.Context, flowID string) error {
	path := fmt.Sprintf("%s/%s.json", p.Dir, flowID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ============================================================
// Flow Router — Conditional Branching
// ============================================================

// Router is a special node that returns a route label instead of modifying state.
// The label determines which listener fires next.
type Router func(ctx context.Context, state State) (string, error)

// AddEventRouter registers a router that conditionally branches execution.
// Based on the returned label, the corresponding event listener is triggered.
func (f *Flow) AddEventRouter(router Router) {
	f.AddNode(func(ctx context.Context, state State) (State, error) {
		label, err := router(ctx, state)
		if err != nil {
			return state, err
		}

		// Emit the route label as an event
		if err := f.Emit(ctx, label, state); err != nil {
			return state, err
		}
		return state, nil
	})
}

// ============================================================
// Listener Combinators — or_() and and_()
// ============================================================

// Or_ triggers the node when ANY of the specified events fire.
func (f *Flow) Or_(events []string, node Node) {
	for _, event := range events {
		f.On(event, node)
	}
}

// And_ triggers the node only when ALL specified events have fired.
func (f *Flow) And_(events []string, node Node) {
	receivedMu := sync.Mutex{}
	received := make(map[string]bool)
	var states []State

	for _, event := range events {
		evt := event // capture
		f.On(evt, func(ctx context.Context, state State) (State, error) {
			receivedMu.Lock()
			received[evt] = true
			states = append(states, state)

			// Check if all events have fired
			allFired := true
			for _, e := range events {
				if !received[e] {
					allFired = false
					break
				}
			}
			receivedMu.Unlock()

			if allFired {
				// Merge all states
				merged := make(State)
				for _, s := range states {
					for k, v := range s {
						merged[k] = v
					}
				}
				return node(ctx, merged)
			}
			return state, nil
		})
	}
}

// ============================================================
// Human Feedback Node
// ============================================================

// HumanFeedbackConfig configures a human feedback pause point.
type HumanFeedbackConfig struct {
	Message        string   // Prompt displayed to the human
	PossibleRoutes []string // e.g., ["approved", "rejected"]
	DefaultOutcome string   // Fallback if input can't be classified
}

// AddHumanFeedback inserts a pause point where the flow waits for human input.
func (f *Flow) AddHumanFeedback(cfg HumanFeedbackConfig) {
	f.AddNode(func(ctx context.Context, state State) (State, error) {
		fmt.Printf("\n🤖 HUMAN FEEDBACK REQUIRED: %s\n", cfg.Message)
		if len(cfg.PossibleRoutes) > 0 {
			fmt.Printf("   Options: %v\n", cfg.PossibleRoutes)
		}
		fmt.Print("   Your input: ")

		var input string
		fmt.Scanln(&input)

		state["human_feedback"] = input

		// Route based on input
		outcome := cfg.DefaultOutcome
		for _, route := range cfg.PossibleRoutes {
			if input == route {
				outcome = route
				break
			}
		}

		if outcome != "" {
			state["human_outcome"] = outcome
			if err := f.Emit(ctx, outcome, state); err != nil {
				return state, err
			}
		}

		return state, nil
	})
}

// ============================================================
// Persistent Flow — Auto-saves state after each node
// ============================================================

// PersistentFlow wraps a Flow with automatic state persistence.
type PersistentFlow struct {
	*Flow
	persistence FlowPersistence
	flowID      string
}

// NewPersistentFlow creates a flow that auto-persists state after each node.
func NewPersistentFlow(flowID string, persistence FlowPersistence, initial State) *PersistentFlow {
	return &PersistentFlow{
		Flow:        NewFlow(initial),
		persistence: persistence,
		flowID:      flowID,
	}
}

// Kickoff runs the persistent flow, restoring state if available.
func (pf *PersistentFlow) Kickoff(ctx context.Context) (State, error) {
	// Try to restore previous state
	if saved, err := pf.persistence.LoadState(ctx, pf.flowID); err == nil && saved != nil {
		pf.Flow.mu.Lock()
		for k, v := range saved {
			pf.Flow.state[k] = v
		}
		pf.Flow.mu.Unlock()
	}

	// Run each node with auto-persist
	for i, node := range pf.Flow.nodes {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		pf.Flow.mu.RLock()
		currentState := pf.Flow.state
		pf.Flow.mu.RUnlock()

		newState, err := node(ctx, currentState)
		if err != nil {
			// Persist on failure for recovery
			pf.persistence.SaveState(ctx, pf.flowID, pf.Flow.state)
			return nil, fmt.Errorf("persistent flow node %d failed: %w", i, err)
		}

		pf.Flow.mu.Lock()
		for k, v := range newState {
			pf.Flow.state[k] = v
		}
		pf.Flow.mu.Unlock()

		// Auto-persist after each successful node
		if err := pf.persistence.SaveState(ctx, pf.flowID, pf.Flow.state); err != nil {
			return nil, fmt.Errorf("state persistence failed at node %d: %w", i, err)
		}
	}

	// Clean up on successful completion
	pf.persistence.DeleteState(ctx, pf.flowID)

	pf.Flow.mu.RLock()
	defer pf.Flow.mu.RUnlock()
	return pf.Flow.state, nil
}
