package adk

import (
	"context"
	"fmt"
	"testing"

	"github.com/Ecook14/gocrewwai/gocrew"
	"github.com/Ecook14/gocrewwai/pkg/memory"
)

func TestADKAgent_Name(t *testing.T) {
	agent := &mockGocAgent{role: "Test Researcher", goal: "Research things"}
	adkAgent := NewADKAgent(agent)
	if got := adkAgent.Name(); got != "Test Researcher" {
		t.Errorf("Name() = %q, want %q", got, "Test Researcher")
	}
}

func TestADKAgent_Description(t *testing.T) {
	agent := &mockGocAgent{role: "Test Researcher", goal: "Research things"}
	adkAgent := NewADKAgent(agent)
	if got := adkAgent.Description(); got != "Research things" {
		t.Errorf("Description() = %q, want %q", got, "Research things")
	}
}

func TestADKAgent_SetSession(t *testing.T) {
	agent := &mockGocAgent{role: "Test", goal: "Test goal"}
	adkAgent := NewADKAgent(agent)
	mockSess := &mockSession{id: "sess-1", userID: "user-1"}
	bridge := NewSessionBridge(mockSess)
	adkAgent.SetSession(bridge)
	if adkAgent.GetSession() == nil {
		t.Error("GetSession should not return nil after SetSession")
	}
	if adkAgent.GetSession().ID() != "sess-1" {
		t.Errorf("session ID = %q, want %q", adkAgent.GetSession().ID(), "sess-1")
	}
}

func TestADKAgent_Run(t *testing.T) {
	agent := &mockGocAgent{role: "Test", goal: "Test goal"}
	adkAgent := NewADKAgent(agent)
	mockSess := &mockSession{id: "sess-1", userID: "user-1"}
	bridge := NewSessionBridge(mockSess)
	adkAgent.SetSession(bridge)

	event, err := adkAgent.Run(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if event.Role != "assistant" {
		t.Errorf("Run() event.Role = %q, want %q", event.Role, "assistant")
	}
	if event.Content == "" {
		t.Error("Run() event.Content should not be empty")
	}
}

func TestADKAgent_RunWithTools(t *testing.T) {
	agent := &mockGocAgent{role: "Test", goal: "Test goal"}
	adkAgent := NewADKAgent(agent)
	_, err := adkAgent.RunWithTools(context.Background(), "Hello", nil)
	if err != nil {
		t.Fatalf("RunWithTools() error = %v", err)
	}
}

func TestSessionBridge_ID(t *testing.T) {
	session := &mockSession{id: "sess-1", userID: "user-1"}
	bridge := NewSessionBridge(session)
	if got := bridge.ID(); got != "sess-1" {
		t.Errorf("ID() = %q, want %q", got, "sess-1")
	}
}

func TestSessionBridge_UserID(t *testing.T) {
	session := &mockSession{id: "sess-1", userID: "user-1"}
	bridge := NewSessionBridge(session)
	if got := bridge.UserID(); got != "user-1" {
		t.Errorf("UserID() = %q, want %q", got, "user-1")
	}
}

func TestSessionBridge_AddEvent(t *testing.T) {
	session := &mockSession{id: "sess-1", userID: "user-1"}
	bridge := NewSessionBridge(session)
	bridge.AddEvent(Event{Role: "assistant", Content: "Hi"})
	bridge.AddEvent(Event{Role: "user", Content: "Hello again"})
	if len(bridge.History()) != 2 {
		t.Fatalf("History length = %d, want 2", len(bridge.History()))
	}
}

func TestSessionBridge_GetLastUserMessage(t *testing.T) {
	session := &mockSession{id: "sess-1", userID: "user-1"}
	bridge := NewSessionBridge(session)
	bridge.AddEvent(Event{Role: "user", Content: "First"})
	bridge.AddEvent(Event{Role: "assistant", Content: "Response"})
	bridge.AddEvent(Event{Role: "user", Content: "Second"})

	msg, ok := bridge.GetLastUserMessage()
	if !ok {
		t.Fatal("GetLastUserMessage() returned false")
	}
	if msg != "Second" {
		t.Errorf("GetLastUserMessage() = %q, want %q", msg, "Second")
	}
}

func TestSessionBridge_ClearHistory(t *testing.T) {
	session := &mockSession{id: "sess-1", userID: "user-1"}
	bridge := NewSessionBridge(session)
	bridge.AddEvent(Event{Role: "user", Content: "Hello"})
	bridge.ClearHistory()
	if len(bridge.History()) != 0 {
		t.Errorf("After ClearHistory, History length = %d, want 0", len(bridge.History()))
	}
}

func TestSessionBridge_AsAgent(t *testing.T) {
	session := &mockSession{id: "sess-1", userID: "user-1"}
	bridge := NewSessionBridge(session)
	agent := bridge.AsAgent()
	if agent.GetRole() != "user-1" {
		t.Errorf("AsAgent().GetRole() = %q, want %q", agent.GetRole(), "user-1")
	}
}

func TestToolsToADK(t *testing.T) {
	gocTools := []gocrew.Tool{
		&mockGocTool{name: "tool1", desc: "First tool"},
		&mockGocTool{name: "tool2", desc: "Second tool"},
	}
	adkTools := ToolsToADK(gocTools)
	if len(adkTools) != 2 {
		t.Fatalf("ToolsToADK() length = %d, want 2", len(adkTools))
	}
	if got := adkTools[0].Name(); got != "tool1" {
		t.Errorf("ToolsToADK()[0].Name() = %q, want %q", got, "tool1")
	}
	if got := adkTools[1].Name(); got != "tool2" {
		t.Errorf("ToolsToADK()[1].Name() = %q, want %q", got, "tool2")
	}
}

func TestADKToolToGoc(t *testing.T) {
	adkTool := &mockADKTool{name: "adk-tool", desc: "ADK tool"}
	gocTool := ADKToolToGoc(adkTool)
	if got := gocTool.Name(); got != "adk-tool" {
		t.Errorf("ADKToolToGoc().Name() = %q, want %q", got, "adk-tool")
	}
	if got := gocTool.Description(); got != "ADK tool" {
		t.Errorf("ADKToolToGoc().Description() = %q, want %q", got, "ADK tool")
	}
	if _, err := gocTool.Execute(context.Background(), map[string]interface{}{"input": "test"}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestContentToGocInput(t *testing.T) {
	content := &Content{
		Parts: []Part{
			&TextPart{Text: "Hello world"},
		},
	}
	input := ContentToGocInput(content)
	if got, want := input, "Hello world"; got != want {
		t.Errorf("ContentToGocInput() = %q, want %q", got, want)
	}
}

func TestContentToGocInput_Nil(t *testing.T) {
	input := ContentToGocInput(nil)
	if input != "" {
		t.Errorf("ContentToGocInput(nil) = %q, want empty", input)
	}
}

func TestGocResultToContent(t *testing.T) {
	result := "Hello from gocrewwai"
	content := GocResultToContent(result)
	if content == nil {
		t.Fatal("GocResultToContent() returned nil")
	}
	if len(content.Parts) != 1 {
		t.Fatalf("Content.Parts length = %d, want 1", len(content.Parts))
	}
}

func TestFunctionCallToArgs(t *testing.T) {
	fc := &FunctionCallPart{
		Name: "search",
		Args: map[string]any{"query": "hello", "limit": 10},
	}
	args := FunctionCallToArgs(fc)
	if len(args) != 2 {
		t.Fatalf("FunctionCallToArgs() returned %d args, want 2", len(args))
	}
	if got, ok := args["query"].(string); !ok || got != "hello" {
		t.Errorf("args['query'] = %v, want 'hello'", args["query"])
	}
}

func TestArgsToFunctionResponse(t *testing.T) {
	fr := ArgsToFunctionResponse("search", `{"result": "found"}`)
	if fr.Name != "search" {
		t.Errorf("FunctionResponsePart.Name = %q, want %q", fr.Name, "search")
	}
	if fr.Response != `{"result": "found"}` {
		t.Errorf("FunctionResponsePart.Response = %q, want %q", fr.Response, `{"result": "found"}`)
	}
}

// simpleStore is a minimal in-memory store for state adapter tests.
type simpleStore struct {
	data map[string]any
}

func (s *simpleStore) Add(ctx context.Context, item *memory.MemoryItem) error {
	if s.data == nil {
		s.data = make(map[string]any)
	}
	s.data[item.ID] = item.Text
	return nil
}

func (s *simpleStore) Get(ctx context.Context, key string) (*memory.MemoryItem, error) {
	val, ok := s.data[key]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return &memory.MemoryItem{ID: key, Text: val.(string)}, nil
}

func (s *simpleStore) Search(ctx context.Context, query string, opts ...interface{}) ([]*memory.MemoryItem, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestAgentRegistry_RegisterAndGet(t *testing.T) {
	r := NewAgentRegistry()
	agent := &mockGocAgent{role: "test", goal: "test goal"}
	adkAgent := NewADKAgent(agent)
	r.Register("test-agent", adkAgent)

	got, ok := r.Get("test-agent")
	if !ok {
		t.Fatal("Get() returned false")
	}
	if got.Name() != "test" {
		t.Errorf("Get().Name() = %q, want %q", got.Name(), "test")
	}
}

func TestAgentRegistry_List(t *testing.T) {
	r := NewAgentRegistry()
	r.Register("a1", NewADKAgent(&mockGocAgent{role: "a1"}))
	r.Register("a2", NewADKAgent(&mockGocAgent{role: "a2"}))

	agents := r.List()
	if len(agents) != 2 {
		t.Fatalf("List() length = %d, want 2", len(agents))
	}
}

func TestToolRegistry_RegisterAndGet(t *testing.T) {
	r := NewToolRegistry()
	tool := &mockADKTool{name: "my-tool", desc: "desc"}
	r.Register("my-tool", tool)

	got, ok := r.Get("my-tool")
	if !ok {
		t.Fatal("Get() returned false")
	}
	if got.Name() != "my-tool" {
		t.Errorf("Get().Name() = %q, want %q", got.Name(), "my-tool")
	}
}

func TestToolRegistry_List(t *testing.T) {
	r := NewToolRegistry()
	r.Register("tool1", &mockADKTool{name: "tool1"})
	r.Register("tool2", &mockADKTool{name: "tool2"})
	r.Register("tool3", &mockADKTool{name: "tool3"})

	tools := r.List()
	if len(tools) != 3 {
		t.Fatalf("List() length = %d, want 3", len(tools))
	}
}
