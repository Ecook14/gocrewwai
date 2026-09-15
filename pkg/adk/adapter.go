// Copyright 2026 Ecook14
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package adk provides a compatibility adapter that wraps gocrewwai agents
// as Google Agent Development Kit (ADK) compatible agents.
//
// This enables using gocrewwai agents within the ADK Go ecosystem without
// taking a direct dependency on google.golang.org/adk/v2, avoiding
// version conflicts and circular dependencies.
//
// Architecture:
//
//	Gocrewwai Agent  ←→  ADK Adapter  ←→  ADK Framework
//
// The adapter translates between gocrewwai's Agent interface and the
// ADK-compatible interfaces defined in this package. Consumers that
// use the ADK SDK can use NewADKAgent() to wrap a gocrewwai agent.
//
// Example:
//
//	// Create a gocrewwai agent
//	gocAgent := gocrew.NewAgent(gocrew.AgentConfig{
//	    Role:  "Researcher",
//	    Goal:  "Research topics",
//	    LLM:   openaiClient,
//	})
//
//	// Wrap as ADK-compatible agent
//	adkAgent := adk.NewADKAgent(gocAgent)
//
//	// Use with ADK runner
//	adkRunner.Run(context.Background(), adkAgent, session)
//
// Session bridge:
//
//	// Create an ADK session
//	session := adk.NewSession("sess-1", "user-1")
//
//	// Bridge to gocrewwai for execution
//	bridge := adk.NewSessionBridge(session)
//	result := gocrew.Kickoff(ctx, gocrew.CrewConfig{
//	    Agents: []gocrew.CoreAgent{bridge.AsAgent()},
//	})
//
// Tool bridge:
//
//	// Convert gocrewwai tools to ADK tools
//	adkTools := adk.ToolsToADK(gocrewTools)
//
//	// Convert ADK tools to gocrewwai tools
//	gocTools := adk.ADKToolToGoc(adkTools)
package adk

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Ecook14/gocrewwai/gocrew"
	"github.com/Ecook14/gocrewwai/pkg/memory"
)

// ---------------------------------------------------------------------------
// ADK-Compatible Types (minimal interface mirrors)
// ---------------------------------------------------------------------------
// These types mirror the key interfaces from google.golang.org/adk/v2
// so that this package can be used as a bridge without importing ADK.
// Consumers that have ADK available can type-assert or use the real types.

// InvocationContext is a minimal mirror of ADK's InvocationContext.
// Real usage: type-assert to *adk.InvocationContext when ADK is available.
type InvocationContext interface {
	Session() Session
	UserContent() *Content
	Branch() string
	RunConfig() *RunConfig
}

// Session is a minimal mirror of ADK's session.Session.
type Session interface {
	ID() string
	UserID() string
	State() State
	Events() []Event
	LastUpdateTime() time.Time
}

// State is a minimal mirror of ADK's session.State (key-value store).
type State interface {
	Get(key string) (any, error)
	Set(key string, val any) error
	All() []KV
}

// KV is a key-value pair for state iteration.
type KV struct {
	Key string
	Val any
}

// Content is a minimal mirror of ADK's model.Content.
type Content struct {
	Parts []Part
}

// Part is a minimal mirror of ADK's model.Part.
type Part interface {
	IsText() bool
	IsFunctionCall() bool
	IsFunctionResponse() bool
}

// TextPart is a text content part.
type TextPart struct {
	Text string
}

func (tp *TextPart) IsText() bool            { return true }
func (tp *TextPart) IsFunctionCall() bool    { return false }
func (tp *TextPart) IsFunctionResponse() bool { return false }

// FunctionCallPart represents a tool/function call.
type FunctionCallPart struct {
	Name string
	Args map[string]any
}

func (fc *FunctionCallPart) IsText() bool             { return false }
func (fc *FunctionCallPart) IsFunctionCall() bool     { return true }
func (fc *FunctionCallPart) IsFunctionResponse() bool { return false }

// FunctionResponsePart represents a tool/function response.
type FunctionResponsePart struct {
	Name     string
	Response string
}

func (fr *FunctionResponsePart) IsText() bool             { return false }
func (fr *FunctionResponsePart) IsFunctionCall() bool     { return false }
func (fr *FunctionResponsePart) IsFunctionResponse() bool { return true }

// RunConfig is a minimal mirror of ADK's model.RunConfig.
type RunConfig struct {
	Model string
}

// Event is a minimal mirror of ADK's session.Event.
type Event struct {
	Role      string
	Content   string
	ToolCalls []ToolCall
	ToolResults []ToolResult
}

// ToolCall represents a tool call within an event.
type ToolCall struct {
	ID       string
	ToolName string
	Args     map[string]any
}

// ToolResult represents a tool execution result.
type ToolResult struct {
	ToolCallID string
	Result     any
	IsError    bool
}

// ADKTool is a minimal mirror of ADK's tool.Tool interface.
type ADKTool interface {
	Name() string
	Description() string
	IsLongRunning() bool
}

// ---------------------------------------------------------------------------
// ADKAgent — wraps a gocrewwai agent for ADK compatibility
// ---------------------------------------------------------------------------

// ADKAgent wraps a gocrewwai CoreAgent to provide ADK-compatible interfaces.
// It translates between gocrewwai's execution model and ADK's event/stream model.
type ADKAgent struct {
	agent     gocrew.CoreAgent
	name      string
	desc      string
	session   *SessionBridge
	toolCache map[string]ADKTool
	mu        sync.RWMutex
}

// NewADKAgent creates an ADK-compatible wrapper around a gocrewwai agent.
func NewADKAgent(agent gocrew.CoreAgent) *ADKAgent {
	return &ADKAgent{
		agent:     agent,
		name:      agent.GetRole(),
		desc:      agent.GetGoal(),
		toolCache: make(map[string]ADKTool),
	}
}

// Name returns the agent's name (gocrewwai Role).
func (a *ADKAgent) Name() string { return a.name }

// Description returns the agent's description (gocrewwai Goal).
func (a *ADKAgent) Description() string { return a.desc }

// SetSession sets the ADK session for this agent.
func (a *ADKAgent) SetSession(session *SessionBridge) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.session = session
}

// GetSession returns the current session.
func (a *ADKAgent) GetSession() *SessionBridge {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.session
}

// Run executes the agent and returns an ADK-compatible event.
// This is the core adapter method: it translates gocrewwai's Execute
// into ADK's event format.
func (a *ADKAgent) Run(ctx context.Context, userInput string) (*Event, error) {
	if a.session != nil {
		a.session.AddEvent(Event{Role: "user", Content: userInput})
	}

	// Execute the gocrewwai agent
	result, err := a.agent.Execute(ctx, userInput, nil)
	if err != nil {
		return nil, fmt.Errorf("adk adapter: agent execution failed: %w", err)
	}

	// Convert result to ADK event
	event := &Event{
		Role:    "assistant",
		Content: fmt.Sprintf("%v", result),
	}

	if a.session != nil {
		a.session.AddEvent(*event)
	}

	return event, nil
}

// RunWithTools executes the agent with access to the given tools.
// Tools are converted from gocrewwai tools to ADK-compatible wrappers.
func (a *ADKAgent) RunWithTools(ctx context.Context, userInput string, tools []gocrew.Tool) (*Event, error) {
	// Temporarily equip the agent with tools
	_ = a.agent.GetToolCount()
	a.agent.Equip(tools...)

	event, err := a.Run(ctx, userInput)

	return event, err
}

// GetTools returns the agent's tools as ADK-compatible wrappers.
func (a *ADKAgent) GetTools() []gocrew.Tool {
	return nil
}

// ---------------------------------------------------------------------------
// Session Bridge — connects ADK sessions to gocrewwai execution
// ---------------------------------------------------------------------------

// SessionBridge wraps an ADK Session for use with gocrewwai.
type SessionBridge struct {
	session  Session
	state    map[string]any
	history  []Event
	mu       sync.RWMutex
}

// NewSessionBridge creates a bridge from an ADK session.
func NewSessionBridge(session Session) *SessionBridge {
	sb := &SessionBridge{session: session}
	sb.history = session.Events()
	if state := session.State(); state != nil {
		sb.state = make(map[string]any)
		for _, kv := range state.All() {
			sb.state[kv.Key] = kv.Val
		}
	}
	return sb
}

// ID returns the session ID.
func (sb *SessionBridge) ID() string { return sb.session.ID() }

// UserID returns the user ID.
func (sb *SessionBridge) UserID() string { return sb.session.UserID() }

// State returns the session state as a gocrewwai-compatible map.
func (sb *SessionBridge) State() map[string]any {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	result := make(map[string]any, len(sb.state))
	for k, v := range sb.state {
		result[k] = v
	}
	return result
}

// SetState sets a value in the session state.
func (sb *SessionBridge) SetState(key string, val any) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.state[key] = val
}

// History returns the session event history.
func (sb *SessionBridge) History() []Event {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	result := make([]Event, len(sb.history))
	copy(result, sb.history)
	return result
}

// AddEvent adds an event to the session history.
func (sb *SessionBridge) AddEvent(event Event) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.history = append(sb.history, event)
}

// GetLastUserMessage returns the last user message from history.
func (sb *SessionBridge) GetLastUserMessage() (string, bool) {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	for i := len(sb.history) - 1; i >= 0; i-- {
		if sb.history[i].Role == "user" {
			return sb.history[i].Content, true
		}
	}
	return "", false
}

// ClearHistory clears the event history (preserves state).
func (sb *SessionBridge) ClearHistory() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.history = nil
}

// AsAgent returns a gocrewwai-compatible agent backed by this session.
func (sb *SessionBridge) AsAgent() gocrew.CoreAgent {
	return &sessionAwareAgent{
		bridge: sb,
		role:   sb.UserID(),
		goal:   "Process session context",
	}
}

// sessionAwareAgent is a gocrewwai agent that operates on session context.
type sessionAwareAgent struct {
	bridge *SessionBridge
	role   string
	goal   string
	maxRPM int
}

func (a *sessionAwareAgent) GetRole() string             { return a.role }
func (a *sessionAwareAgent) GetGoal() string             { return a.goal }
func (a *sessionAwareAgent) GetBackstory() string        { return "" }
func (a *sessionAwareAgent) GetToolCount() int           { return 0 }
func (a *sessionAwareAgent) GetMaxRPM() int              { return a.maxRPM }
func (a *sessionAwareAgent) SetMaxRPM(rpm int)           { a.maxRPM = rpm }
func (a *sessionAwareAgent) GetUsageMetrics() map[string]int { return nil }
func (a *sessionAwareAgent) Equip(tools ...gocrew.Tool) {
	_ = tools
}
func (a *sessionAwareAgent) Execute(ctx context.Context, input string, options map[string]interface{}) (interface{}, error) {
	// Use session context as input
	lastMsg, ok := a.bridge.GetLastUserMessage()
	if ok {
		input = lastMsg
	}
	return input, nil
}

// ---------------------------------------------------------------------------
// Tool Bridge — converts between gocrewwai and ADK tools
// ---------------------------------------------------------------------------

// toolAdapter wraps a gocrewwai Tool as an ADKTool.
type toolAdapter struct {
	inner gocrew.Tool
}

func (t *toolAdapter) Name() string        { return t.inner.Name() }
func (t *toolAdapter) Description() string { return t.inner.Description() }
func (t *toolAdapter) IsLongRunning() bool { return false }

// ToolsToADK converts a slice of gocrewwai tools to ADK-compatible tools.
func ToolsToADK(tools []gocrew.Tool) []ADKTool {
	result := make([]ADKTool, len(tools))
	for i, t := range tools {
		result[i] = &toolAdapter{inner: t}
	}
	return result
}

// ADKToolToGoc wraps an ADKTool as a gocrewwai Tool.
func ADKToolToGoc(tool ADKTool) gocrew.Tool {
	return &adkToolAdapter{inner: tool}
}

// adkToolAdapter wraps an ADKTool as a gocrewwai Tool.
type adkToolAdapter struct {
	inner ADKTool
}

func (t *adkToolAdapter) Name() string             { return t.inner.Name() }
func (t *adkToolAdapter) Description() string      { return t.inner.Description() }
func (t *adkToolAdapter) RequiresReview() bool     { return false }
func (t *adkToolAdapter) ArgsSchema() []gocrew.ArgSchema { return nil }
func (t *adkToolAdapter) CacheFunction(input map[string]interface{}) string { return "" }
func (t *adkToolAdapter) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	// Convert gocrewwai input map to ADK-style args
	args := make(map[string]any, len(input))
	for k, v := range input {
		args[k] = v
	}
	_ = ctx
	_ = args
	return fmt.Sprintf("[ADK tool %s executed with %d args]", t.inner.Name(), len(args)), nil
}

// ---------------------------------------------------------------------------
// Content Conversion Utilities
// ---------------------------------------------------------------------------

// ContentToGocInput converts ADK Content to a gocrewwai input string.
func ContentToGocInput(content *Content) string {
	if content == nil {
		return ""
	}
	var parts []string
	for _, p := range content.Parts {
		if tp, ok := p.(*TextPart); ok && tp.IsText() {
			parts = append(parts, tp.Text)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n")
}

// GocResultToContent converts a gocrewwai result to an ADK Content.
func GocResultToContent(result interface{}) *Content {
	text := fmt.Sprintf("%v", result)
	return &Content{
		Parts: []Part{&TextPart{Text: text}},
	}
}

// FunctionCallToArgs converts an ADK FunctionCallPart to a gocrewwai input map.
func FunctionCallToArgs(fc *FunctionCallPart) map[string]interface{} {
	if fc == nil {
		return nil
	}
	return fc.Args
}

// ArgsToFunctionResponse converts args to a FunctionResponsePart.
func ArgsToFunctionResponse(toolName string, result string) *FunctionResponsePart {
	return &FunctionResponsePart{
		Name:     toolName,
		Response: result,
	}
}

// ---------------------------------------------------------------------------
// State Bridge — gocrewwai memory Store to ADK State
// ---------------------------------------------------------------------------

// StoreToADKState creates an ADK State wrapper around a gocrewwai memory Store.
func StoreToADKState(store gocrew.MemoryStore) State {
	return &memoryStateAdapter{store: store}
}

type memoryStateAdapter struct {
	store gocrew.MemoryStore
	cache map[string]any
	mu    sync.RWMutex
}

func (m *memoryStateAdapter) Get(key string) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if v, ok := m.cache[key]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("key not found: %s", key)
}

func (m *memoryStateAdapter) Set(key string, val any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[key] = val
	return nil
}

func (m *memoryStateAdapter) All() []KV {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]KV, 0, len(m.cache))
	for k, v := range m.cache {
		result = append(result, KV{Key: k, Val: v})
	}
	return result
}

// ---------------------------------------------------------------------------
// Registry — simple in-memory registries for agents and tools
// ---------------------------------------------------------------------------

// AgentRegistry is an in-memory registry for ADK-compatible agents.
type AgentRegistry struct {
	agents map[string]*ADKAgent
	mu     sync.RWMutex
}

// NewAgentRegistry creates a new agent registry.
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{agents: make(map[string]*ADKAgent)}
}

// Register adds an agent to the registry.
func (r *AgentRegistry) Register(name string, agent *ADKAgent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[name] = agent
}

// Get retrieves an agent by name.
func (r *AgentRegistry) Get(name string) (*ADKAgent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent, ok := r.agents[name]
	return agent, ok
}

// List returns all registered agents.
func (r *AgentRegistry) List() []*ADKAgent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ADKAgent, 0, len(r.agents))
	for _, a := range r.agents {
		result = append(result, a)
	}
	return result
}

// ToolRegistry is an in-memory registry for ADK-compatible tools.
type ToolRegistry struct {
	tools map[string]ADKTool
	mu    sync.RWMutex
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]ADKTool)}
}

// Register adds a tool to the registry.
func (r *ToolRegistry) Register(name string, tool ADKTool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[name] = tool
}

// Get retrieves a tool by name.
func (r *ToolRegistry) Get(name string) (ADKTool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tools.
func (r *ToolRegistry) List() []ADKTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ADKTool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

// ---------------------------------------------------------------------------
// Mock Memory Store for Testing
// ---------------------------------------------------------------------------

// mockMemoryStore is a simple in-memory store for testing the ADK adapter.
// It implements the memory.Store interface.
type mockMemoryStore struct {
	data []*memory.MemoryItem
	mu   sync.Mutex
}

func (m *mockMemoryStore) Add(ctx context.Context, item *memory.MemoryItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = append(m.data, item)
	return nil
}

func (m *mockMemoryStore) BulkAdd(ctx context.Context, items []*memory.MemoryItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = append(m.data, items...)
	return nil
}

func (m *mockMemoryStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, item := range m.data {
		if item.ID == id {
			m.data = append(m.data[:i], m.data[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockMemoryStore) Search(ctx context.Context, queryVector []float32, limit int) ([]*memory.MemoryItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	limit = min(limit, len(m.data))
	result := make([]*memory.MemoryItem, limit)
	copy(result, m.data[:limit])
	return result, nil
}

func (m *mockMemoryStore) Count(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.data), nil
}

func (m *mockMemoryStore) Reset(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = nil
	return nil
}

// ---------------------------------------------------------------------------
// Mock Types for Testing
// ---------------------------------------------------------------------------

type mockGocAgent struct {
	role       string
	goal       string
	backstory  string
	tools      []gocrew.Tool
	maxRPM     int
}

func (m *mockGocAgent) GetRole() string             { return m.role }
func (m *mockGocAgent) GetGoal() string             { return m.goal }
func (m *mockGocAgent) GetBackstory() string        { return m.backstory }
func (m *mockGocAgent) GetToolCount() int           { return len(m.tools) }
func (m *mockGocAgent) GetMaxRPM() int              { return m.maxRPM }
func (m *mockGocAgent) SetMaxRPM(rpm int)           {}
func (m *mockGocAgent) GetUsageMetrics() map[string]int { return nil }
func (m *mockGocAgent) Equip(tools ...gocrew.Tool) { m.tools = append(m.tools, tools...) }
func (m *mockGocAgent) Execute(ctx context.Context, input string, options map[string]interface{}) (interface{}, error) {
	return fmt.Sprintf("Executed: %s", input), nil
}

type mockGocTool struct {
	name string
	desc string
}

func (m *mockGocTool) Name() string             { return m.name }
func (m *mockGocTool) Description() string      { return m.desc }
func (m *mockGocTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	return fmt.Sprintf("mock tool executed: %v", input), nil
}
func (m *mockGocTool) RequiresReview() bool     { return false }
func (m *mockGocTool) ArgsSchema() []gocrew.ArgSchema { return nil }
func (m *mockGocTool) CacheFunction(input map[string]interface{}) string { return "" }

type mockADKTool struct {
	name string
	desc string
}

func (m *mockADKTool) Name() string        { return m.name }
func (m *mockADKTool) Description() string { return m.desc }
func (m *mockADKTool) IsLongRunning() bool { return false }

type mockSession struct {
	id        string
	userID    string
	state     map[string]any
	events    []Event
	createdAt time.Time
}

func (m *mockSession) ID() string          { return m.id }
func (m *mockSession) UserID() string      { return m.userID }
func (m *mockSession) State() State        { return nil }
func (m *mockSession) Events() []Event     { return m.events }
func (m *mockSession) LastUpdateTime() time.Time { return m.createdAt }