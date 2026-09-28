package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
)

// TypedNode represents a work block that operates on a strongly typed state.
type TypedNode[T any] func(ctx context.Context, state T) (T, error)

// TypedFlow provides a type-safe wrapper around the state machine orchestration.
// Exceeds Python's approach by providing native Go compile-time safety for state.
type TypedFlow[T any] struct {
	nodes []TypedNode[T]
	state T
	mu    sync.RWMutex

	persistence FlowPersistence
	flowID      string
}

// NewTypedFlow initializes a flow with a specific type-save state.
func NewTypedFlow[T any](initial T) *TypedFlow[T] {
	return &TypedFlow[T]{
		nodes: make([]TypedNode[T], 0),
		state: initial,
	}
}

// WithPersistence enables auto-saving for this typed flow.
func (f *TypedFlow[T]) WithPersistence(id string, p FlowPersistence) *TypedFlow[T] {
	f.flowID = id
	f.persistence = p
	return f
}

// AddNode registers a new step in the flow.
func (f *TypedFlow[T]) AddNode(n TypedNode[T]) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nodes = append(f.nodes, n)
}

// typedStateKey is the envelope key under which a TypedFlow persists T.
const typedStateKey = "__typed_state"

// Kickoff executes the flow top-to-bottom, resuming persisted state when available.
func (f *TypedFlow[T]) Kickoff(ctx context.Context) (T, error) {
	slog.Info("🌊 Starting Typed Flow Execution", slog.Int("nodes", len(f.nodes)))

	// 1. Restore persisted state if available
	if f.persistence != nil && f.flowID != "" {
		saved, err := f.persistence.LoadState(ctx, f.flowID)
		if err != nil {
			return f.state, fmt.Errorf("typed flow: failed to load state: %w", err)
		}
		if saved != nil {
			if err := restoreTypedState(saved, &f.state); err != nil {
				return f.state, fmt.Errorf("typed flow: failed to restore state: %w", err)
			}
			slog.Info("📍 Resumed Typed Flow state from persistence")
		}
	}

	for i, node := range f.nodes {
		select {
		case <-ctx.Done():
			return f.state, ctx.Err()
		default:
		}

		f.mu.RLock()
		current := f.state
		f.mu.RUnlock()

		slog.Info("Executing Typed Node", slog.Int("index", i))
		next, err := node(ctx, current)
		if err != nil {
			return f.state, fmt.Errorf("node %d failed: %w", i, err)
		}

		f.mu.Lock()
		f.state = next
		f.mu.Unlock()

		// 2. Auto-Persist if enabled
		if f.persistence != nil && f.flowID != "" {
			if err := saveTypedState(ctx, f.persistence, f.flowID, f.state); err != nil {
				return f.state, fmt.Errorf("typed flow: state persistence failed at node %d: %w", i, err)
			}
		}
	}

	slog.Info("🏁 Typed Flow Complete")
	return f.state, nil
}

// saveTypedState round-trips T through JSON into the untyped State envelope.
func saveTypedState[T any](ctx context.Context, p FlowPersistence, flowID string, state T) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal typed state: %w", err)
	}
	stateMap := make(State)
	stateMap[typedStateKey] = json.RawMessage(data)
	return p.SaveState(ctx, flowID, stateMap)
}

// restoreTypedState extracts T from the untyped State envelope.
func restoreTypedState[T any](saved State, target *T) error {
	raw, ok := saved[typedStateKey]
	if !ok {
		return fmt.Errorf("no %q in persisted state", typedStateKey)
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("failed to re-encode persisted state: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to decode persisted state: %w", err)
	}
	return nil
}
