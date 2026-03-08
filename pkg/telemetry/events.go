package telemetry

import (
	"context"
	//"fmt"
	"sync"
	"time"
)

// EventType defines the category of a telemetry event.
type EventType string

const (
	EventAgentStarted   EventType = "agent_started"
	EventAgentThinking  EventType = "agent_thinking"
	EventToolStarted    EventType = "tool_started"
	EventToolFinished   EventType = "tool_finished"
	EventAgentFinished  EventType = "agent_finished"
	EventTaskStarted    EventType = "task_started"
	EventTaskFinished   EventType = "task_finished"
	EventSystemLog      EventType = "system_log"
)

// Event represents a single unit of telemetry data pushed to the dashboard.
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	AgentRole string                 `json:"agent_role,omitempty"`
	Payload   map[string]interface{} `json:"payload"`
}

// EventBus handles the subscription and broadcasting of execution events.
type EventBus struct {
	subscribers []chan Event
	mu          sync.RWMutex
}

var GlobalBus = &EventBus{
	subscribers: make([]chan Event, 0),
}

// Subscribe adds a new listener for events.
func (b *EventBus) Subscribe() chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Event, 100)
	b.subscribers = append(b.subscribers, ch)
	return ch
}

// Unsubscribe removes a listener.
func (b *EventBus) Unsubscribe(ch chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, sub := range b.subscribers {
		if sub == ch {
			b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

// Publish broadcasts an event to all active subscribers.
func (b *EventBus) Publish(e Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	for _, sub := range b.subscribers {
		// Non-blocking send to avoid hanging the engine if a subscriber is slow
		select {
		case sub <- e:
		default:
		}
	}
}

// ---------------------------------------------------------------------------
// Human-In-The-Loop (HITL) Global Review Manager
// ---------------------------------------------------------------------------

type ReviewManager struct {
	Pending map[string]chan bool
	mu      sync.Mutex
}

var GlobalReviewManager = &ReviewManager{
	Pending: make(map[string]chan bool),
}

// RequestReview blocks until the UI or manual API approves or rejects the execution.
func (r *ReviewManager) RequestReview(id string, agentRole, toolName string, input interface{}) bool {
	ch := make(chan bool)
	r.mu.Lock()
	r.Pending[id] = ch
	r.mu.Unlock()

	// Broadcast an event so the UI knows a review is pending
	GlobalBus.Publish(Event{
		Type:      "review_requested",
		AgentRole: agentRole,
		Payload: map[string]interface{}{
			"review_id": id,
			"tool_name": toolName,
			"input":     input,
		},
	})

	// Block until UI responds via API
	decision := <-ch

	r.mu.Lock()
	delete(r.Pending, id)
	r.mu.Unlock()

	return decision
}

// SubmitReview resolves a pending review.
func (r *ReviewManager) SubmitReview(id string, approved bool) {
	r.mu.Lock()
	ch, ok := r.Pending[id]
	r.mu.Unlock()
	if ok {
		ch <- approved
	}
}

// ---------------------------------------------------------------------------
// Dynamic Execution Control
// ---------------------------------------------------------------------------

type ExecutionController struct {
	mu     sync.Mutex
	cond   *sync.Cond
	paused bool
}

// GlobalExecutionController allows the Web UI dashboard to pause the entire ReAct loop.
var GlobalExecutionController = &ExecutionController{}

func init() {
	GlobalExecutionController.cond = sync.NewCond(&GlobalExecutionController.mu)
}

// Pause flags the execution engine to halt before the next Tool invocation.
func (c *ExecutionController) Pause() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = true
}

// Resume unblocks all waiting Agent routines.
func (c *ExecutionController) Resume() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = false
	c.cond.Broadcast()
}

// WaitIfPaused blocks the calling goroutine until the dashboard resumes execution.
func (c *ExecutionController) WaitIfPaused(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	for c.paused {
		// Periodically wake up to check context cancellation while paused
		done := make(chan struct{})
		go func() {
			c.cond.Wait()
			close(done)
		}()

		select {
		case <-ctx.Done():
			c.cond.Broadcast() // free the waiter goroutine
			return ctx.Err()
		case <-done:
			// Woken up by Broadcast, re-check c.paused
		}
	}
	return nil
}
