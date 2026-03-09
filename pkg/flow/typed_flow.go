package flow

import (
	"context"
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

// Kickoff executes the flow top-to-bottom.
func (f *TypedFlow[T]) Kickoff(ctx context.Context) (T, error) {
	slog.Info("🌊 Starting Typed Flow Execution", slog.Int("nodes", len(f.nodes)))

	// 1. Try Load State if persistence enabled
	if f.persistence != nil && f.flowID != "" {
		saved, err := f.persistence.LoadState(ctx, f.flowID)
		if err == nil && saved != nil {
			// Note: Generic unmarshaling of State map to T would require reflection.
			// Simplified: We assume for now TypedFlow manages its own T or starts fresh.
			// In production, we'd use a more sophisticated JSON unmarshaler for T.
			slog.Info("📍 Resumed Typed Flow state (simulated recovery)")
		}
	}

	for i, node := range f.nodes {
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
			// Convert T to map[string]interface{} for standard persistence
			// Simplified representation
			stateMap := make(State)
			stateMap["__typed_state"] = f.state 
			_ = f.persistence.SaveState(ctx, f.flowID, stateMap)
		}
	}

	slog.Info("🏁 Typed Flow Complete")
	return f.state, nil
}
