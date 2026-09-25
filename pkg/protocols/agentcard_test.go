package protocols

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/tools"
)

func TestAgentCard_Validate(t *testing.T) {
	tests := []struct {
		name    string
		card    *AgentCard
		wantErr bool
	}{
		{
			name: "valid card",
			card: &AgentCard{
				ID:          "test-agent",
				Name:        "Test Agent",
				Description: "A test agent",
				Endpoint:    "http://localhost:8080",
				Version:     "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			card: &AgentCard{
				Name:     "Test Agent",
				Endpoint: "http://localhost:8080",
			},
			wantErr: true,
		},
		{
			name: "missing Name",
			card: &AgentCard{
				ID:       "test-agent",
				Endpoint: "http://localhost:8080",
			},
			wantErr: true,
		},
		{
			name: "missing endpoint",
			card: &AgentCard{
				ID:      "test-agent",
				Name:    "Test Agent",
				Version: "1.0.0",
			},
			wantErr: true,
		},
		{
			name:    "missing all required fields",
			card:    &AgentCard{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgentCard(tt.card)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAgentCard() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgentCard_ToJSON(t *testing.T) {
	card := &AgentCard{
		ID:           "test-agent",
		Name:         "Test Agent",
		Description:  "A test agent for serialization",
		Endpoint:     "http://localhost:8080",
		Role:         "agent",
		Version:      "1.0.0",
		Capabilities: []string{"agent_framework", "tool_use"},
		Metadata: map[string]string{
			"source": "test",
		},
	}

	data, err := card.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse serialized JSON: %v", err)
	}

	if result["id"] != "test-agent" {
		t.Errorf("serialized id = %v, want %v", result["id"], "test-agent")
	}
	if result["name"] != "Test Agent" {
		t.Errorf("serialized name = %v, want %v", result["name"], "Test Agent")
	}
	if result["description"] != "A test agent for serialization" {
		t.Errorf("serialized description = %v, want %v", result["description"], "A test agent for serialization")
	}
	if result["version"] != "1.0.0" {
		t.Errorf("serialized version = %v, want %v", result["version"], "1.0.0")
	}
}

func TestAgentCard_ToJSONCompact(t *testing.T) {
	card := &AgentCard{
		ID:       "compact-agent",
		Name:     "Compact Agent",
		Endpoint: "http://localhost:9090",
	}

	data, err := card.ToJSONCompact()
	if err != nil {
		t.Fatalf("ToJSONCompact() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse compact JSON: %v", err)
	}

	if result["id"] != "compact-agent" {
		t.Errorf("compact id = %v, want %v", result["id"], "compact-agent")
	}
	if result["name"] != "Compact Agent" {
		t.Errorf("compact name = %v, want %v", result["name"], "Compact Agent")
	}
}

func TestAgentCard_Deserialize(t *testing.T) {
	jsonData := `{
		"id": "test-agent",
		"name": "Test Agent",
		"description": "A test agent for deserialization",
		"version": "2.0.0",
		"endpoint": "http://example.com/agent",
		"capabilities": ["test_cap"],
		"metadata": {"source": "test"}
	}`

	var card AgentCard
	if err := json.Unmarshal([]byte(jsonData), &card); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if card.ID != "test-agent" {
		t.Errorf("ID = %v, want %v", card.ID, "test-agent")
	}
	if card.Name != "Test Agent" {
		t.Errorf("Name = %v, want %v", card.Name, "Test Agent")
	}
	if card.Version != "2.0.0" {
		t.Errorf("Version = %v, want %v", card.Version, "2.0.0")
	}
	if len(card.Capabilities) != 1 {
		t.Errorf("Capabilities = %v, want %v", card.Capabilities, []string{"test_cap"})
	}
	if _, ok := card.Metadata["source"]; !ok {
		t.Error("Metadata should contain 'source' key")
	}
}

func TestAgentCard_EmptyCard(t *testing.T) {
	card := &AgentCard{}
	if err := ValidateAgentCard(card); err == nil {
		t.Error("ValidateAgentCard() should return error for empty card")
	}
}

func TestAgentCard_Builder(t *testing.T) {
	builder := NewAgentCardBuilder()
	card, err := builder.
		WithID("builder-agent").
		WithName("Builder Agent").
		WithEndpoint("http://localhost:7070").
		WithDescription("A builder agent").
		WithVersion("1.0.0").
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if card.ID != "builder-agent" {
		t.Errorf("ID = %v, want %v", card.ID, "builder-agent")
	}
	if card.Name != "Builder Agent" {
		t.Errorf("Name = %v, want %v", card.Name, "Builder Agent")
	}
	if card.Endpoint != "http://localhost:7070" {
		t.Errorf("Endpoint = %v, want %v", card.Endpoint, "http://localhost:7070")
	}
}

func TestAgentCard_Builder_Invalid(t *testing.T) {
	builder := NewAgentCardBuilder()
	_, err := builder.
		WithID(""). // missing ID
		WithName("Agent").
		WithEndpoint("http://localhost:7070").
		Build()

	if err == nil {
		t.Error("Build() should fail with empty ID")
	}
}

func TestToolCard_GenerateFromRegistry(t *testing.T) {
	registry := tools.GlobalRegistry
	registry.Register(&mockTestTool{
		name:        "tool1",
		description: "First tool",
		schema: []tools.ArgSchema{
			{Name: "query", Type: "string", Description: "Search query", Required: true},
		},
	})
	registry.Register(&mockTestTool{
		name:        "tool2",
		description: "Second tool",
	})

	cards := GenerateToolCardsFromRegistry(registry)

	if len(cards) != 2 {
		t.Fatalf("GenerateToolCardsFromRegistry() length = %d, want 2", len(cards))
	}

	if cards[0].Name != "tool1" {
		t.Errorf("GenerateToolCardsFromRegistry()[0].Name = %v, want %v", cards[0].Name, "tool1")
	}
	if cards[0].Description != "First tool" {
		t.Errorf("GenerateToolCardsFromRegistry()[0].Description = %v, want %v", cards[0].Description, "First tool")
	}
	if cards[1].Name != "tool2" {
		t.Errorf("GenerateToolCardsFromRegistry()[1].Name = %v, want %v", cards[1].Name, "tool2")
	}
}

func TestToolCard_EmptyRegistry(t *testing.T) {
	emptyReg := tools.NewToolRegistry()
	cards := GenerateToolCardsFromRegistry(emptyReg)
	if len(cards) != 0 {
		t.Errorf("GenerateToolCardsFromRegistry() length = %d, want 0", len(cards))
	}
}

func TestToolCard_Struct(t *testing.T) {
	tc := &ToolCard{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "The name to greet",
				},
			},
			"required": []string{"name"},
		},
	}

	jsonData, err := json.Marshal(tc)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if result["name"] != "test_tool" {
		t.Errorf("name = %v, want %v", result["name"], "test_tool")
	}
	if result["description"] != "A test tool" {
		t.Errorf("description = %v, want %v", result["description"], "A test tool")
	}

	props, ok := result["input_schema"].(map[string]interface{})
	if !ok {
		t.Fatal("input_schema should be an object")
	}
	if props["type"] != "object" {
		t.Errorf("input_schema.type = %v, want %v", props["type"], "object")
	}
}

// mockTestTool implements tools.Tool for testing.
type mockTestTool struct {
	name        string
	description string
	schema      []tools.ArgSchema
}

func (m *mockTestTool) Name() string {
	return m.name
}

func (m *mockTestTool) Description() string {
	return m.description
}

func (m *mockTestTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	return "executed: " + m.name, nil
}

func (m *mockTestTool) RequiresReview() bool {
	return false
}

func (m *mockTestTool) ArgsSchema() []tools.ArgSchema {
	return m.schema
}

func (m *mockTestTool) CacheFunction(input map[string]interface{}) string {
	return ""
}
