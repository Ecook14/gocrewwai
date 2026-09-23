// Package protocols implements interoperability protocols for Gocrew agents.
//
// This file provides A2A-compliant AgentCard JSON generation and validation
// per the A2A project spec: https://a2aproject.org
//
// The AgentCard type is defined in a2a.go. This file adds A2A-compliant
// serialization, validation, and generation helpers.
package protocols

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/tools"
)

// ---------------------------------------------------------------------------
// A2A-Compliant AgentCard Helpers
// ---------------------------------------------------------------------------
// The AgentCard struct is defined in a2a.go. This file provides:
// - Validation against A2A spec requirements
// - JSON serialization (pretty and compact)
// - Generation from core.Agent with functional options
// - Tool advertisement from tool registries and MCP servers

// ValidateAgentCard checks an AgentCard for A2A spec compliance.
// Required fields: id, name, endpoint.
func ValidateAgentCard(card *AgentCard) error {
	if card == nil {
		return fmt.Errorf("agentcard: card is nil")
	}
	if card.ID == "" {
		return fmt.Errorf("agentcard: id is required")
	}
	if card.Name == "" {
		return fmt.Errorf("agentcard: name is required")
	}
	if card.Endpoint == "" {
		return fmt.Errorf("agentcard: endpoint is required")
	}
	if _, err := url.ParseRequestURI(card.Endpoint); err != nil {
		return fmt.Errorf("agentcard: invalid endpoint URL: %w", err)
	}
	return nil
}

// ToJSON serializes the AgentCard to pretty-printed JSON bytes.
func (c *AgentCard) ToJSON() ([]byte, error) {
	if err := ValidateAgentCard(c); err != nil {
		return nil, err
	}
	return json.MarshalIndent(c, "", "  ")
}

// ToJSONCompact serializes the AgentCard to compact JSON bytes.
func (c *AgentCard) ToJSONCompact() ([]byte, error) {
	if err := ValidateAgentCard(c); err != nil {
		return nil, err
	}
	return json.Marshal(c)
}

// AgentCardBuilder helps construct A2A-compliant AgentCards.
type AgentCardBuilder struct {
	card *AgentCard
}

// NewAgentCardBuilder creates a builder for an A2A AgentCard.
func NewAgentCardBuilder() *AgentCardBuilder {
	card := &AgentCard{
		Metadata:  make(map[string]string),
		CreatedAt: time.Now(),
	}
	return &AgentCardBuilder{card: card}
}

// Build returns the constructed AgentCard.
func (b *AgentCardBuilder) Build() (*AgentCard, error) {
	if err := ValidateAgentCard(b.card); err != nil {
		return nil, err
	}
	return b.card, nil
}

// WithID sets the agent's unique identifier.
func (b *AgentCardBuilder) WithID(id string) *AgentCardBuilder {
	b.card.ID = id
	return b
}

// WithName sets the human-readable agent name.
func (b *AgentCardBuilder) WithName(name string) *AgentCardBuilder {
	b.card.Name = name
	return b
}

// WithEndpoint sets the agent's A2A message endpoint URL.
func (b *AgentCardBuilder) WithEndpoint(endpoint string) *AgentCardBuilder {
	b.card.Endpoint = endpoint
	return b
}

// WithVersion sets the agent version.
func (b *AgentCardBuilder) WithVersion(version string) *AgentCardBuilder {
	b.card.Version = version
	return b
}

// WithCapability adds a capability to the card.
func (b *AgentCardBuilder) WithCapability(cap string) *AgentCardBuilder {
	b.card.Capabilities = append(b.card.Capabilities, cap)
	return b
}

// WithCapabilities sets all capabilities at once.
func (b *AgentCardBuilder) WithCapabilities(caps []string) *AgentCardBuilder {
	b.card.Capabilities = caps
	return b
}

// WithMetadata sets a metadata key-value pair.
func (b *AgentCardBuilder) WithMetadata(key, value string) *AgentCardBuilder {
	b.card.Metadata[key] = value
	return b
}

// WithDescription sets the agent description.
func (b *AgentCardBuilder) WithDescription(desc string) *AgentCardBuilder {
	b.card.Description = desc
	return b
}

// SetCard sets the internal card directly.
func (b *AgentCardBuilder) SetCard(card *AgentCard) *AgentCardBuilder {
	b.card = card
	return b
}

// GenerateAgentCardFromCore generates an A2A AgentCard from a gocrewwai CoreAgent.
func GenerateAgentCardFromCore(agent core.Agent, endpoint string, opts ...AgentCardOption) (*AgentCard, error) {
	builder := NewAgentCardBuilder()
	builder.WithID(agent.GetRole())
	builder.WithName(agent.GetRole())
	builder.WithDescription(agent.GetGoal())
	builder.WithEndpoint(endpoint)

	for _, opt := range opts {
		opt(builder.card)
	}

	return builder.Build()
}

// AgentCardOption is a functional option for customizing generated AgentCards.
type AgentCardOption func(*AgentCard)

// WithAgentCardID overrides the agent card ID.
func WithAgentCardID(id string) AgentCardOption {
	return func(c *AgentCard) {
		c.ID = id
	}
}

// WithAgentCardName overrides the agent card name.
func WithAgentCardName(name string) AgentCardOption {
	return func(c *AgentCard) {
		c.Name = name
	}
}

// WithAgentCardVersion sets the version.
func WithAgentCardVersion(version string) AgentCardOption {
	return func(c *AgentCard) {
		c.Version = version
	}
}

// WithAgentCardMetadata adds metadata.
func WithAgentCardMetadata(metadata map[string]string) AgentCardOption {
	return func(c *AgentCard) {
		for k, v := range metadata {
			c.Metadata[k] = v
		}
	}
}

// WithAgentCardCapabilities sets capabilities.
func WithAgentCardCapabilities(caps []string) AgentCardOption {
	return func(c *AgentCard) {
		c.Capabilities = caps
	}
}

// ToolCard represents a tool advertised in an agent's capability card.
type ToolCard struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema,omitempty"`
}

// GenerateToolCardsFromRegistry creates ToolCard entries from a tool registry.
func GenerateToolCardsFromRegistry(registry *tools.ToolRegistry) []*ToolCard {
	var cards []*ToolCard
	for _, tool := range registry.List() {
		cards = append(cards, &ToolCard{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: toolSchemaToMap(tool),
		})
	}
	// Sort by name for deterministic output.
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].Name < cards[j].Name
	})
	return cards
}

// toolSchemaToMap converts a gocrewwai Tool's args schema to a JSON-serializable map.
func toolSchemaToMap(tool tools.Tool) map[string]interface{} {
	schema := tool.ArgsSchema()
	if len(schema) == 0 {
		return nil
	}

	properties := make(map[string]interface{})
	required := make([]string, 0)
	for _, arg := range schema {
		properties[arg.Name] = map[string]interface{}{
			"type":        arg.Type,
			"description": arg.Description,
		}
		if arg.Required {
			required = append(required, arg.Name)
		}
	}

	if len(properties) == 0 {
		return nil
	}

	result := map[string]interface{}{
		"type": "object",
		"properties": properties,
	}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}

// AgentCardFromToolRegistry generates an A2A AgentCard with tool advertisements.
func AgentCardFromToolRegistry(agent core.Agent, endpoint string, registry *tools.ToolRegistry, opts ...AgentCardOption) (*AgentCard, error) {
	card, err := GenerateAgentCardFromCore(agent, endpoint, opts...)
	if err != nil {
		return nil, err
	}

	toolCards := GenerateToolCardsFromRegistry(registry)
	if len(toolCards) > 0 {
		card.Capabilities = append(card.Capabilities, "tool_use")
		toolIDs := make([]string, len(toolCards))
		for i, tc := range toolCards {
			toolIDs[i] = tc.Name
		}
		card.Metadata["tools"] = toolIDsString(toolIDs)
	}

	return card, nil
}

// toolIDsString converts tool ID slice to a comma-separated string.
func toolIDsString(ids []string) string {
	result := ""
	for i, id := range ids {
		if i > 0 {
			result += ","
		}
		result += id
	}
	return result
}

// AgentCardFromMCPServer generates an A2A AgentCard from an MCP server's tools.
func AgentCardFromMCPServer(agent core.Agent, endpoint string, mcpServer *MCPServer, opts ...AgentCardOption) (*AgentCard, error) {
	card, err := GenerateAgentCardFromCore(agent, endpoint, opts...)
	if err != nil {
		return nil, err
	}

	mcpServer.mu.RLock()
	toolNames := make([]string, 0, len(mcpServer.tools))
	for name := range mcpServer.tools {
		toolNames = append(toolNames, name)
	}
	mcpServer.mu.RUnlock()

	if len(toolNames) > 0 {
		card.Capabilities = append(card.Capabilities, "tool_use")
		card.Metadata["tools"] = toolIDsString(toolNames)
	}

	return card, nil
}

// AgentCardFromAgentAdapter creates an AgentCard from a RemoteAgentAdapter.
func AgentCardFromAgentAdapter(adapter *RemoteAgentAdapter) *AgentCard {
	return &AgentCard{
		ID:           adapter.Card.ID,
		Name:         adapter.Card.Name,
		Role:         adapter.Card.Role,
		Description:  adapter.Card.Description,
		Capabilities: adapter.Card.Capabilities,
		Endpoint:     adapter.Card.Endpoint,
		Version:      adapter.Card.Version,
		Metadata:     adapter.Card.Metadata,
		CreatedAt:    adapter.Card.CreatedAt,
	}
}
