package knowledge

import "time"

// ============================================================
// Knowledge Events — Observable lifecycle hooks
// ============================================================

// EventType identifies the kind of knowledge event.
type EventType string

const (
	EventRetrievalStarted   EventType = "knowledge.retrieval.started"
	EventRetrievalCompleted EventType = "knowledge.retrieval.completed"
	EventQueryStarted       EventType = "knowledge.query.started"
	EventQueryCompleted     EventType = "knowledge.query.completed"
	EventQueryFailed        EventType = "knowledge.query.failed"
	EventSearchFailed       EventType = "knowledge.search.failed"
	EventIngestionStarted   EventType = "knowledge.ingestion.started"
	EventIngestionCompleted EventType = "knowledge.ingestion.completed"
)

// Event represents a knowledge lifecycle event.
type Event struct {
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Query     string                 `json:"query,omitempty"`
	Source    string                 `json:"source,omitempty"`
	Results   int                    `json:"results,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EventHandler is a callback function for knowledge events.
type EventHandler func(Event)

// EventBus distributes knowledge events to registered handlers.
type EventBus struct {
	handlers []EventHandler
}

// GlobalKnowledgeBus is the default event bus for knowledge events.
var GlobalKnowledgeBus = &EventBus{}

// Subscribe registers a handler to receive knowledge events.
func (b *EventBus) Subscribe(handler EventHandler) {
	b.handlers = append(b.handlers, handler)
}

// Publish sends an event to all registered handlers.
func (b *EventBus) Publish(event Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	for _, h := range b.handlers {
		h(event)
	}
}
