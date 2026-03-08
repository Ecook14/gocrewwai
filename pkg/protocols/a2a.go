// Package protocols implements interoperability protocols for Gocrew agents.
//
// This package provides two major protocol implementations:
//
//   - A2A (Agent-to-Agent): Enables agents to discover, communicate with,
//     and delegate tasks to other agents across process and network boundaries.
//
//   - MCP (Model Context Protocol): Enables agents to discover and invoke
//     tools hosted on MCP-compatible servers, and expose Gocrew tools as
//     MCP resources.
package protocols

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// A2A Protocol — Agent-to-Agent Communication
// ---------------------------------------------------------------------------

// Register is a global registry for Agent-to-Agent discovery.
var GlobalA2ARegistry = NewAgentRegistry()

// AgentCard declares an agent's identity, capabilities, and endpoint.
// This is the core discovery object in the A2A protocol.
type AgentCard struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Role         string            `json:"role"`
	Description  string            `json:"description,omitempty"`
	Capabilities []string          `json:"capabilities"` // e.g., ["research", "code_review", "writing"]
	Endpoint     string            `json:"endpoint"`     // HTTP endpoint for receiving messages
	Version      string            `json:"version,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
}

// A2AMessage is the standard envelope for inter-agent communication.
type A2AMessage struct {
	ID          string                 `json:"id"`
	From        string                 `json:"from"`         // Sender agent ID
	To          string                 `json:"to"`           // Recipient agent ID
	Type        A2AMessageType         `json:"type"`         // request, response, event, error
	Action      string                 `json:"action"`       // What the sender wants (e.g., "delegate_task", "ask_question")
	Payload     map[string]interface{} `json:"payload"`
	CorrelationID string              `json:"correlation_id,omitempty"` // Links request ↔ response
	Timestamp   time.Time              `json:"timestamp"`
}

// A2AMessageType categorizes inter-agent messages.
type A2AMessageType string

const (
	A2ARequest  A2AMessageType = "request"
	A2AResponse A2AMessageType = "response"
	A2AEvent    A2AMessageType = "event"
	A2AError    A2AMessageType = "error"
)

// A2ATaskRequest is the payload for delegating a task to another agent.
type A2ATaskRequest struct {
	Description    string                 `json:"description"`
	ExpectedOutput string                 `json:"expected_output,omitempty"`
	Context        string                 `json:"context,omitempty"`
	Priority       int                    `json:"priority,omitempty"` // 1=highest, 5=lowest
	Deadline       time.Time              `json:"deadline,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// A2ATaskResponse is the payload for returning a task result.
type A2ATaskResponse struct {
	Result  string `json:"result"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ---------------------------------------------------------------------------
// Agent Registry — Discovery Service
// ---------------------------------------------------------------------------

// AgentRegistry provides agent discovery for the A2A protocol.
type AgentRegistry struct {
	mu     sync.RWMutex
	agents map[string]*AgentCard
}

// NewAgentRegistry creates an empty registry.
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]*AgentCard),
	}
}

// Register adds an agent card to the registry.
func (r *AgentRegistry) Register(card *AgentCard) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if card.CreatedAt.IsZero() {
		card.CreatedAt = time.Now()
	}
	r.agents[card.ID] = card
}

// Unregister removes an agent from the registry.
func (r *AgentRegistry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, id)
}

// Lookup finds an agent by ID.
func (r *AgentRegistry) Lookup(id string) (*AgentCard, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	card, ok := r.agents[id]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", id)
	}
	return card, nil
}

// FindByCapability returns all agents that declare the given capability.
func (r *AgentRegistry) FindByCapability(capability string) []*AgentCard {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []*AgentCard
	for _, card := range r.agents {
		for _, cap := range card.Capabilities {
			if cap == capability {
				matches = append(matches, card)
				break
			}
		}
	}
	return matches
}

// ListAll returns all registered agent cards.
func (r *AgentRegistry) ListAll() []*AgentCard {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*AgentCard, 0, len(r.agents))
	for _, card := range r.agents {
		result = append(result, card)
	}
	return result
}

// ---------------------------------------------------------------------------
// A2A Message Handler
// ---------------------------------------------------------------------------

// A2AHandler processes incoming A2A messages.
type A2AHandler func(ctx context.Context, msg A2AMessage) (*A2AMessage, error)

// A2ARouter routes incoming messages to registered handlers by action.
type A2ARouter struct {
	mu       sync.RWMutex
	handlers map[string]A2AHandler
	fallback A2AHandler
}

// NewA2ARouter creates a message router.
func NewA2ARouter() *A2ARouter {
	return &A2ARouter{
		handlers: make(map[string]A2AHandler),
	}
}

// Handle registers a handler for a specific action.
func (r *A2ARouter) Handle(action string, handler A2AHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[action] = handler
}

// SetFallback sets a handler for unrecognized actions.
func (r *A2ARouter) SetFallback(handler A2AHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallback = handler
}

// Route dispatches a message to the appropriate handler.
func (r *A2ARouter) Route(ctx context.Context, msg A2AMessage) (*A2AMessage, error) {
	r.mu.RLock()
	handler, ok := r.handlers[msg.Action]
	fallback := r.fallback
	r.mu.RUnlock()

	if ok {
		return handler(ctx, msg)
	}
	if fallback != nil {
		return fallback(ctx, msg)
	}
	return nil, fmt.Errorf("no handler for action: %s", msg.Action)
}

// ---------------------------------------------------------------------------
// Serialization Helpers
// ---------------------------------------------------------------------------

// MarshalTaskRequest converts a task request into a generic payload map.
func MarshalTaskRequest(req A2ATaskRequest) map[string]interface{} {
	data, _ := json.Marshal(req)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result
}

// UnmarshalTaskRequest extracts a task request from a message payload.
func UnmarshalTaskRequest(payload map[string]interface{}) (*A2ATaskRequest, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var req A2ATaskRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	return &req, nil
}
