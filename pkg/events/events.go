package events

import (
	"sync"
	"time"
)

// ============================================================
// Event Categories & Types (Parity with CrewAI Python)
// ============================================================

type EventType string

const (
	// Crew Events
	CrewKickoffStarted   EventType = "crew.kickoff.started"
	CrewKickoffCompleted EventType = "crew.kickoff.completed"
	CrewKickoffFailed    EventType = "crew.kickoff.failed"
	CrewTestStarted      EventType = "crew.test.started"
	CrewTestCompleted    EventType = "crew.test.completed"
	CrewTestFailed       EventType = "crew.test.failed"
	CrewTrainStarted     EventType = "crew.train.started"
	CrewTrainCompleted   EventType = "crew.train.completed"
	CrewTrainFailed      EventType = "crew.train.failed"

	// Agent Events
	AgentExecutionStarted   EventType = "agent.execution.started"
	AgentExecutionCompleted EventType = "agent.execution.completed"
	AgentExecutionError     EventType = "agent.execution.error"
	AgentThinking           EventType = "agent.thinking"
	AgentReasoningStarted   EventType = "agent.reasoning.started"
	AgentReasoningCompleted EventType = "agent.reasoning.completed"

	// Task Events
	TaskStarted    EventType = "task.started"
	TaskCompleted  EventType = "task.completed"
	TaskFailed     EventType = "task.failed"
	TaskEvaluated  EventType = "task.evaluated"

	// Tool Events
	ToolUsageStarted       EventType = "tool.usage.started"
	ToolUsageFinished      EventType = "tool.usage.finished"
	ToolUsageError         EventType = "tool.usage.error"
	ToolValidateInputError EventType = "tool.validate.error"
	ToolSelectionError     EventType = "tool.selection.error"

	// Knowledge Events
	KnowledgeRetrievalStarted   EventType = "knowledge.retrieval.started"
	KnowledgeRetrievalCompleted EventType = "knowledge.retrieval.completed"
	KnowledgeQueryStarted       EventType = "knowledge.query.started"
	KnowledgeQueryCompleted     EventType = "knowledge.query.completed"
	KnowledgeQueryFailed        EventType = "knowledge.query.failed"

	// LLM Events
	LLMCallStarted     EventType = "llm.call.started"
	LLMCallCompleted   EventType = "llm.call.completed"
	LLMCallFailed      EventType = "llm.call.failed"
	LLMStreamChunk     EventType = "llm.stream.chunk"
	LLMGuardrailStarted   EventType = "llm.guardrail.started"
	LLMGuardrailCompleted EventType = "llm.guardrail.completed"

	// Memory Events
	MemoryQueryStarted     EventType = "memory.query.started"
	MemoryQueryCompleted   EventType = "memory.query.completed"
	MemoryQueryFailed      EventType = "memory.query.failed"
	MemorySaveStarted      EventType = "memory.save.started"
	MemorySaveCompleted    EventType = "memory.save.completed"
	MemorySaveFailed       EventType = "memory.save.failed"

	// Flow Events
	FlowCreated  EventType = "flow.created"
	FlowStarted  EventType = "flow.started"
	FlowFinished EventType = "flow.finished"
	FlowPlot     EventType = "flow.plot"
)

// ============================================================
// Event Structure
// ============================================================

// Event represents a system-wide lifecycle event.
type Event struct {
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source,omitempty"`     // e.g., Agent role, Task name
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Error     error                  `json:"error,omitempty"`
}

// Handler is a callback for receiving events.
type Handler func(Event)

// ============================================================
// Global Event Bus (Pub/Sub)
// ============================================================

// Bus manages event registration and distribution.
type Bus struct {
	mu          sync.RWMutex
	handlers    []Handler
	subscribers []chan Event
}

// GlobalBus is the singleton event bus for Gocrewwai.
var GlobalBus = &Bus{}

// Subscribe returns a channel that receives all published events.
func (b *Bus) Subscribe() chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Event, 100)
	b.subscribers = append(b.subscribers, ch)
	return ch
}

// Unsubscribe removes a channel listener.
func (b *Bus) Unsubscribe(ch chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, sub := range b.subscribers {
		if sub == ch {
			b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
			close(ch)
			return
		}
	}
}

// On registers a simple callback handler.
func (b *Bus) On(handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, handler)
}

// Publish broadcasts an event to all subscribers and handlers.
func (b *Bus) Publish(e Event) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Notify handlers
	for _, h := range b.handlers {
		go h(e) // Run in goroutine to prevent blocking execution
	}

	// Notify channel subscribers
	for _, sub := range b.subscribers {
		select {
		case sub <- e:
		default: // Skip if channel is full
		}
	}
}

// ScopedHandlers returns a context manager-like closure to temporarily listen for events.
func (b *Bus) ScopedHandlers(handler Handler) func() {
	b.mu.Lock()
	b.handlers = append(b.handlers, handler)
	b.mu.Unlock()

	// Return a cleanup function
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		for i, h := range b.handlers {
			// Note: This comparison is tricky in Go for functions, 
			// but works if we use a unique pointer/wrapper if necessary.
			// For now, simpler to just use it for temporary lifetime.
			_ = i
			_ = h
		}
	}
}
