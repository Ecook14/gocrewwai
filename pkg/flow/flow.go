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
}

func NewFlow(initialState State) *Flow {
	if initialState == nil {
		initialState = make(State)
	}
	return &Flow{
		nodes: make([]Node, 0),
		state: initialState,
	}
}

// AddNode pushes a generic work block onto the state machine chain.
func (f *Flow) AddNode(n Node) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nodes = append(f.nodes, n)
}

// Kickoff securely runs the state machine top-to-bottom.
// In Go, because Channels and WaitGroups are first-class citizens, this base sequencer
// natively supports Context timeouts far more securely than Python loops.
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
