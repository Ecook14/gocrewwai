package flow

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Predicate is a function that evaluates state and returns true/false.
type Predicate func(state State) bool

// ConditionalNode wraps a Node with a predicate — the node only executes
// if the predicate returns true for the current state.
type ConditionalNode struct {
	Pred Predicate
	Node Node
	Name string // Optional name for logging
}

// RouterNode maps multiple predicates to different downstream nodes.
// The first matching predicate's node is executed.
// If no predicates match, the DefaultNode is used (if provided).
type RouterNode struct {
	Routes      []Route
	DefaultNode Node
}

// Route pairs a predicate with a node for use in RouterNode.
type Route struct {
	Name string
	Pred Predicate
	Node Node
}

// ListenerConfig triggers a node when a specific state key changes value.
type ListenerConfig struct {
	Key      string
	Expected interface{}
	Node     Node
}

// AddConditionalNode adds a node that only executes when the predicate is satisfied.
func (f *Flow) AddConditionalNode(name string, pred Predicate, node Node) {
	wrapped := func(ctx context.Context, state State) (State, error) {
		if !pred(state) {
			if name != "" {
				slog.Info("Conditional node skipped", slog.String("name", name))
			}
			return state, nil // Pass through unchanged
		}
		if name != "" {
			slog.Info("Conditional node executing", slog.String("name", name))
		}
		return node(ctx, state)
	}
	f.AddNode(wrapped)
}

// AddRouter adds a routing node that dispatches to different nodes based on state conditions.
// Routes are evaluated in order; the first matching route's node is executed.
func (f *Flow) AddRouter(router *RouterNode) {
	wrapped := func(ctx context.Context, state State) (State, error) {
		for _, route := range router.Routes {
			if route.Pred(state) {
				slog.Info("Router matched", slog.String("route", route.Name))
				return route.Node(ctx, state)
			}
		}
		if router.DefaultNode != nil {
			slog.Info("Router using default route")
			return router.DefaultNode(ctx, state)
		}
		slog.Info("Router: no routes matched, passing through")
		return state, nil
	}
	f.AddNode(wrapped)
}

// AddListener adds a node that triggers only when a specific state key has the expected value.
func (f *Flow) AddListener(config *ListenerConfig) {
	pred := func(state State) bool {
		val, exists := state[config.Key]
		return exists && val == config.Expected
	}
	f.AddConditionalNode("listener:"+config.Key, pred, config.Node)
}

// AddParallelNodes adds multiple nodes that execute concurrently.
// All nodes receive a copy of the current state; their outputs are merged back.
// In case of key conflicts, later nodes (by index) overwrite earlier ones.
func (f *Flow) AddParallelNodes(nodes []Node) {
	wrapped := func(ctx context.Context, state State) (State, error) {
		type result struct {
			state State
			err   error
			index int
		}

		results := make([]result, len(nodes))
		var wg sync.WaitGroup

		for i, node := range nodes {
			wg.Add(1)
			go func(idx int, n Node) {
				defer wg.Done()
				// Create a shallow copy of state for each parallel node
				stateCopy := make(State)
				for k, v := range state {
					stateCopy[k] = v
				}
				newState, err := n(ctx, stateCopy)
				results[idx] = result{state: newState, err: err, index: idx}
			}(i, node)
		}

		wg.Wait()

		// Check for errors
		for _, r := range results {
			if r.err != nil {
				return nil, fmt.Errorf("parallel node %d failed: %w", r.index, r.err)
			}
		}

		// Merge all results back into a single state
		merged := make(State)
		for k, v := range state {
			merged[k] = v
		}
		for _, r := range results {
			for k, v := range r.state {
				merged[k] = v
			}
		}

		slog.Info("Parallel nodes complete", slog.Int("count", len(nodes)))
		return merged, nil
	}
	f.AddNode(wrapped)
}
// OnEvent is a DX-friendly wrapper for registering event listeners.
// It allows for cleaner flow definitions in a builder-like style.
func (f *Flow) OnEvent(event string, handler Node) *Flow {
	f.On(event, handler)
	return f
}

// When registers a listener that only triggers if the event matches AND a secondary predicate is true.
func (f *Flow) When(event string, pred Predicate, handler Node) *Flow {
	wrapped := func(ctx context.Context, state State) (State, error) {
		if !pred(state) {
			slog.Info("Event predicate not met, skipping handler", slog.String("event", event))
			return state, nil
		}
		return handler(ctx, state)
	}
	f.On(event, wrapped)
	return f
}
