package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
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
	EventSystemMetrics  EventType = "system_metrics"
	EventSandboxStatus  EventType = "sandbox_status"
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
// ---------------------------------------------------------------------------
// Dynamic Entity Registry
// ---------------------------------------------------------------------------

type DynamicRegistry struct {
	Agents         []interface{}
	Tasks          []interface{}
	MCPClients     []interface{}
	A2ABridges     []interface{}
	
	// Internal pending queues for the engine to consume
	pendingAgents  []interface{}
	pendingTasks   []interface{}
	pendingMCP     []interface{}
	pendingA2A     []interface{}
	processType    string

	mu sync.RWMutex
}

var GlobalDynamicRegistry = &DynamicRegistry{
	Agents:     make([]interface{}, 0),
	Tasks:      make([]interface{}, 0),
	MCPClients: make([]interface{}, 0),
	A2ABridges: make([]interface{}, 0),
}

func (r *DynamicRegistry) AddAgent(agent interface{}, pending bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Agents = append(r.Agents, agent)
	if pending {
		r.pendingAgents = append(r.pendingAgents, agent)
	}
}

func (r *DynamicRegistry) AddTask(task interface{}, pending bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Tasks = append(r.Tasks, task)
	if pending {
		r.pendingTasks = append(r.pendingTasks, task)
	}
}

func (r *DynamicRegistry) AddMCPClient(client interface{}, pending bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.MCPClients = append(r.MCPClients, client)
	if pending {
		r.pendingMCP = append(r.pendingMCP, client)
	}
}

func (r *DynamicRegistry) AddA2ABridge(bridge interface{}, pending bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.A2ABridges = append(r.A2ABridges, bridge)
	if pending {
		r.pendingA2A = append(r.pendingA2A, bridge)
	}
}

func (r *DynamicRegistry) DeleteAgent(index int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if index >= 0 && index < len(r.Agents) {
		agent := r.Agents[index]
		r.Agents = append(r.Agents[:index], r.Agents[index+1:]...)
		// Also remove from pending if present
		for i, p := range r.pendingAgents {
			if p == agent {
				r.pendingAgents = append(r.pendingAgents[:i], r.pendingAgents[i+1:]...)
				break
			}
		}
	}
}

func (r *DynamicRegistry) DeleteTask(index int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if index >= 0 && index < len(r.Tasks) {
		task := r.Tasks[index]
		r.Tasks = append(r.Tasks[:index], r.Tasks[index+1:]...)
		for i, p := range r.pendingTasks {
			if p == task {
				r.pendingTasks = append(r.pendingTasks[:i], r.pendingTasks[i+1:]...)
				break
			}
		}
	}
}

func (r *DynamicRegistry) DeleteMCP(index int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if index >= 0 && index < len(r.MCPClients) {
		client := r.MCPClients[index]
		r.MCPClients = append(r.MCPClients[:index], r.MCPClients[index+1:]...)
		for i, p := range r.pendingMCP {
			if p == client {
				r.pendingMCP = append(r.pendingMCP[:i], r.pendingMCP[i+1:]...)
				break
			}
		}
	}
}

func (r *DynamicRegistry) DeleteA2A(index int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if index >= 0 && index < len(r.A2ABridges) {
		bridge := r.A2ABridges[index]
		r.A2ABridges = append(r.A2ABridges[:index], r.A2ABridges[index+1:]...)
		for i, p := range r.pendingA2A {
			if p == bridge {
				r.pendingA2A = append(r.pendingA2A[:i], r.pendingA2A[i+1:]...)
				break
			}
		}
	}
}

func (r *DynamicRegistry) UpdateAgent(index int, agent interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if index >= 0 && index < len(r.Agents) {
		r.Agents[index] = agent
		r.pendingAgents = append(r.pendingAgents, agent)
	}
}

func (r *DynamicRegistry) UpdateTask(index int, task interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if index >= 0 && index < len(r.Tasks) {
		r.Tasks[index] = task
		r.pendingTasks = append(r.pendingTasks, task)
	}
}

// SyncTaskResult updates a task's status and output in the registry without re-triggering it.
func (r *DynamicRegistry) SyncTaskResult(description string, agentRole string, output interface{}, err error, processed bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, t := range r.Tasks {
		// 1. Try Rich Object Interface (pointers to Task)
		if taskPtr, ok := t.(interface {
			GetDescription() string
			GetAgentRole() string
			SetOutput(interface{})
			SetError(error)
			SetProcessed(bool)
		}); ok {
			if taskPtr.GetDescription() == description && 
			   (agentRole == "" || strings.EqualFold(taskPtr.GetAgentRole(), agentRole)) {
				taskPtr.SetOutput(output)
				taskPtr.SetError(err)
				taskPtr.SetProcessed(processed)
				return
			}
		}

		// 2. Fallback: Map Check (UI staged tasks)
		if taskMap, ok := t.(map[string]interface{}); ok {
			if taskMap["description"] == description && 
			   (agentRole == "" || strings.EqualFold(fmt.Sprintf("%v", taskMap["agent_role"]), agentRole)) {
				taskMap["output"] = output
				taskMap["processed"] = processed
				if err != nil {
					taskMap["failed"] = true
					taskMap["error"] = err.Error()
				}
				r.Tasks[i] = taskMap
				return
			}
		}
	}
	
	slog.Debug("SyncTaskResult: No match found for task", slog.String("desc", description), slog.String("role", agentRole))
}

func (r *DynamicRegistry) UpdateMCP(index int, client interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if index >= 0 && index < len(r.MCPClients) {
		r.MCPClients[index] = client
		r.pendingMCP = append(r.pendingMCP, client)
	}
}

func (r *DynamicRegistry) UpdateA2A(index int, bridge interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if index >= 0 && index < len(r.A2ABridges) {
		r.A2ABridges[index] = bridge
		r.pendingA2A = append(r.pendingA2A, bridge)
	}
}

func (r *DynamicRegistry) SetProcessType(p string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.processType = p
}

func (r *DynamicRegistry) GetProcessType() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.processType
}

func (r *DynamicRegistry) ListAll() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create shallow copies of the slices to prevent race conditions during JSON marshaling
	agents := make([]interface{}, len(r.Agents))
	copy(agents, r.Agents)
	tasks := make([]interface{}, len(r.Tasks))
	copy(tasks, r.Tasks)
	mcp := make([]interface{}, len(r.MCPClients))
	copy(mcp, r.MCPClients)
	a2a := make([]interface{}, len(r.A2ABridges))
	copy(a2a, r.A2ABridges)

	return map[string]interface{}{
		"agents": agents,
		"tasks":  tasks,
		"mcp":    mcp,
		"a2a":    a2a,
		"config": map[string]interface{}{
			"process_type": r.processType,
		},
	}
}

func (r *DynamicRegistry) PullTasks() []interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	t := r.pendingTasks
	r.pendingTasks = make([]interface{}, 0)
	return t
}

func (r *DynamicRegistry) PullAgents() []interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	a := r.pendingAgents
	r.pendingAgents = make([]interface{}, 0)
	return a
}

func (r *DynamicRegistry) PullMCPClients() []interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := r.pendingMCP
	r.pendingMCP = make([]interface{}, 0)
	return c
}
