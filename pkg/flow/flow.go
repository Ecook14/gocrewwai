package flow

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// State represents the generic memory payload passed continuously down the flow pipeline.
type State map[string]interface{}

// Node represents a single executable block within the larger state machine.
// In CrewAI, this is usually an entire `Crew.Kickoff()` wrapped in a function.
type Node func(ctx context.Context, state State) (State, error)

// Flow orchestrates multiple Crews sequentially or concurrently based on State triggers.
// Modeled after `crewai.Flow` to allow multi-agent massive ecosystems.
type Flow struct {
	nodes []Node
	state State
	mu    sync.RWMutex

	// Event-Driven Architecture
	listeners map[string][]Node
	lMu       sync.RWMutex
}

func NewFlow(initialState State) *Flow {
	if initialState == nil {
		initialState = make(State)
	}
	return &Flow{
		nodes:     make([]Node, 0),
		state:     initialState,
		listeners: make(map[string][]Node),
	}
}

// AddNode pushes a generic work block onto the state machine chain manually.
func (f *Flow) AddNode(n Node) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nodes = append(f.nodes, n)
}

// On registers a listener for a specific event. When the event is emitted, this node executes.
func (f *Flow) On(event string, n Node) {
	f.lMu.Lock()
	defer f.lMu.Unlock()
	f.listeners[event] = append(f.listeners[event], n)
}

// Emit broadcasts an event to all registered listeners.
func (f *Flow) Emit(ctx context.Context, event string, state State) error {
	f.lMu.RLock()
	handlers := f.listeners[event]
	f.lMu.RUnlock()

	if len(handlers) == 0 {
		return nil
	}

	slog.Info("📡 Flow Event Emitted", slog.String("event", event), slog.Int("listeners", len(handlers)))

	for _, handler := range handlers {
		newState, err := handler(ctx, state)
		if err != nil {
			return err
		}
		// Merge outputs back into the master state
		f.mu.Lock()
		for k, v := range newState {
			f.state[k] = v
		}
		f.mu.Unlock()
	}
	return nil
}

// Start kicks off the flow from a specific starting event.
func (f *Flow) Start(ctx context.Context, event string) (State, error) {
	slog.Info("🌊 Starting Event Flow...", slog.String("entry_event", event))
	if err := f.Emit(ctx, event, f.state); err != nil {
		return nil, err
	}
	return f.state, nil
}

// Kickoff securely runs the state machine top-to-bottom sequentially.
func (f *Flow) Kickoff(ctx context.Context) (State, error) {
	slog.Info("🌊 Starting Event Flow...", slog.Int("nodes", len(f.nodes)))

	for i, node := range f.nodes {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		f.mu.RLock()
		currentState := f.state
		f.mu.RUnlock()

		slog.Info("Executing Flow Node", slog.Int("index", i))
		newState, err := node(ctx, currentState)
		if err != nil {
			return nil, fmt.Errorf("flow node %d failed deterministically: %w", i, err)
		}

		// Merge outputs back into the master state
		f.mu.Lock()
		for k, v := range newState {
			f.state[k] = v
		}
		f.mu.Unlock()
	}

	slog.Info("🏁 Flow Complete")
	
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state, nil
}
