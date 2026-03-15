package flows

import (
	"context"
	"encoding/json"
	//"fmt"
)

// State defines the strictly-typed object that tracks the flow's progress.
// All custom Flow states must implement this to be serializable.
type State interface {
	ToJSON() ([]byte, error)
	FromJSON(data []byte) error
	Clone() State
}

// BaseState is a simple helper for generic state needs.
type BaseState struct {
	Data map[string]interface{} `json:"data"`
}

func (s *BaseState) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

func (s *BaseState) FromJSON(data []byte) error {
	return json.Unmarshal(data, s)
}

func (s *BaseState) Clone() State {
	newData := make(map[string]interface{})
	for k, v := range s.Data {
		newData[k] = v
	}
	return &BaseState{Data: newData}
}

// NodeType defines the execution behavior of a flow node.
type NodeType string

const (
	NodeStep     NodeType = "step"     // Standard sequential step
	NodeRouter   NodeType = "router"   // Dynamic next node based on state
	NodeParallel NodeType = "parallel" // Execute multiple branches concurrently
	NodeMap      NodeType = "map"      // Execute action for each item in a slice
	NodeReduce   NodeType = "reduce"   // Wait and merge results from branches
)

// FlowNode represents a single step in a Flow graph.
type FlowNode struct {
	ID          string
	Description string
	Type        NodeType
	
	// Action is the primary logic for NodeStep and NodeMap items.
	Action func(ctx context.Context, state State) (State, error)
	
	// Router used for NodeRouter to determine next hop.
	Router func(state State) string
	
	// Parallel branches used for NodeParallel.
	ParallelBranches []string
	
	// MapConfig used for NodeMap items.
	MapSourceKey string // Key in state.Data that contains []interface{}
	MapResultKey string // Key in state.Data to store results
	
	// Merge is used for NodeReduce to combine states from multiple branches.
	Merge func(states []State) State

	// Next is the default sequel for NodeStep and NodeReduce.
	Next []string 
}

// Flow is the top-level orchestration unit for complex multi-crew workflows.
type Flow struct {
	ID      string
	Nodes   map[string]*FlowNode
	Initial string
	State   State
}

func NewFlow(id string, initial string, state State) *Flow {
	return &Flow{
		ID:      id,
		Nodes:   make(map[string]*FlowNode),
		Initial: initial,
		State:   state,
	}
}

func (f *Flow) AddNode(node *FlowNode) {
	f.Nodes[node.ID] = node
}
