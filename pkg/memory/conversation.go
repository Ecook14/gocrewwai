package memory

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Conversation Thread Model
// ---------------------------------------------------------------------------

// Role represents a conversation participant.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// ConversationMessage represents a single message in a conversation thread.
type ConversationMessage struct {
	ID        string                 `json:"id"`
	ThreadID  string                 `json:"thread_id"`
	Role      Role                   `json:"role"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// ConversationThread holds an ordered sequence of messages.
type ConversationThread struct {
	ID        string                 `json:"id"`
	Title     string                 `json:"title,omitempty"`
	AgentRole string                 `json:"agent_role,omitempty"`
	Messages  []ConversationMessage  `json:"messages"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// ---------------------------------------------------------------------------
// ConversationStore Interface
// ---------------------------------------------------------------------------

// ConversationStore manages multi-turn conversation threads.
type ConversationStore interface {
	// CreateThread starts a new conversation thread.
	CreateThread(ctx context.Context, thread *ConversationThread) error

	// AddMessage appends a message to an existing thread.
	AddMessage(ctx context.Context, threadID string, msg ConversationMessage) error

	// GetThread retrieves a full conversation thread with all messages.
	GetThread(ctx context.Context, threadID string) (*ConversationThread, error)

	// GetMessages returns the last N messages from a thread (most recent first).
	GetMessages(ctx context.Context, threadID string, limit int) ([]ConversationMessage, error)

	// ListThreads returns all thread IDs and titles.
	ListThreads(ctx context.Context) ([]*ConversationThread, error)

	// DeleteThread removes a thread and all its messages.
	DeleteThread(ctx context.Context, threadID string) error
}

// ---------------------------------------------------------------------------
// InMemConversationStore — Thread-Safe In-Memory Implementation
// ---------------------------------------------------------------------------

// InMemConversationStore implements ConversationStore with an in-memory map.
// Suitable for development, single-crew runs, and testing.
type InMemConversationStore struct {
	mu      sync.RWMutex
	threads map[string]*ConversationThread
	msgSeq  int
}

// NewInMemConversationStore creates an empty conversation store.
func NewInMemConversationStore() *InMemConversationStore {
	return &InMemConversationStore{
		threads: make(map[string]*ConversationThread),
	}
}

func (s *InMemConversationStore) CreateThread(ctx context.Context, thread *ConversationThread) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if thread.ID == "" {
		return fmt.Errorf("thread ID is required")
	}
	if _, exists := s.threads[thread.ID]; exists {
		return fmt.Errorf("thread already exists: %s", thread.ID)
	}

	now := time.Now()
	if thread.CreatedAt.IsZero() {
		thread.CreatedAt = now
	}
	thread.UpdatedAt = now
	if thread.Messages == nil {
		thread.Messages = make([]ConversationMessage, 0)
	}

	// Store a copy
	threadCopy := *thread
	threadCopy.Messages = make([]ConversationMessage, len(thread.Messages))
	copy(threadCopy.Messages, thread.Messages)
	s.threads[thread.ID] = &threadCopy

	return nil
}

func (s *InMemConversationStore) AddMessage(ctx context.Context, threadID string, msg ConversationMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	thread, exists := s.threads[threadID]
	if !exists {
		return fmt.Errorf("thread not found: %s", threadID)
	}

	s.msgSeq++
	if msg.ID == "" {
		msg.ID = fmt.Sprintf("msg_%d", s.msgSeq)
	}
	msg.ThreadID = threadID
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	thread.Messages = append(thread.Messages, msg)
	thread.UpdatedAt = time.Now()

	return nil
}

func (s *InMemConversationStore) GetThread(ctx context.Context, threadID string) (*ConversationThread, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	thread, exists := s.threads[threadID]
	if !exists {
		return nil, fmt.Errorf("thread not found: %s", threadID)
	}

	// Return a copy
	result := *thread
	result.Messages = make([]ConversationMessage, len(thread.Messages))
	copy(result.Messages, thread.Messages)
	return &result, nil
}

func (s *InMemConversationStore) GetMessages(ctx context.Context, threadID string, limit int) ([]ConversationMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	thread, exists := s.threads[threadID]
	if !exists {
		return nil, fmt.Errorf("thread not found: %s", threadID)
	}

	msgs := thread.Messages
	if limit > 0 && limit < len(msgs) {
		msgs = msgs[len(msgs)-limit:]
	}

	// Return copies
	result := make([]ConversationMessage, len(msgs))
	copy(result, msgs)
	return result, nil
}

func (s *InMemConversationStore) ListThreads(ctx context.Context) ([]*ConversationThread, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*ConversationThread, 0, len(s.threads))
	for _, thread := range s.threads {
		// Return lightweight copies (without messages)
		summary := &ConversationThread{
			ID:        thread.ID,
			Title:     thread.Title,
			AgentRole: thread.AgentRole,
			Metadata:  thread.Metadata,
			CreatedAt: thread.CreatedAt,
			UpdatedAt: thread.UpdatedAt,
		}
		result = append(result, summary)
	}
	return result, nil
}

func (s *InMemConversationStore) DeleteThread(ctx context.Context, threadID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.threads[threadID]; !exists {
		return fmt.Errorf("thread not found: %s", threadID)
	}
	delete(s.threads, threadID)
	return nil
}

// ---------------------------------------------------------------------------
// FormatHistory — Convert Thread to LLM-Ready Prompt Context
// ---------------------------------------------------------------------------

// FormatHistory converts a conversation thread's recent messages into a
// string suitable for injecting into an LLM prompt as conversation context.
func FormatHistory(thread *ConversationThread, maxMessages int) string {
	msgs := thread.Messages
	if maxMessages > 0 && maxMessages < len(msgs) {
		msgs = msgs[len(msgs)-maxMessages:]
	}

	result := ""
	for _, msg := range msgs {
		result += fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Content)
	}
	return result
}
