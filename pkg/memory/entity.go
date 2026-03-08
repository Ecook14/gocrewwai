package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// Entity represents a tracked metadata object (e.g., "Project name", "User's tech stack").
type Entity struct {
	Name        string
	Value       string
	Description string
}

// EntityStore provides a structured way to manage named entities found during execution.
type EntityStore interface {
	Upsert(ctx context.Context, name, value, description string) error
	Get(ctx context.Context, name string) (*Entity, error)
	Search(ctx context.Context, query string, limit int) ([]*MemoryItem, error)
}

// InMemEntityStore is a simple map-based implementation of EntityStore.
type InMemEntityStore struct {
	entities map[string]Entity
	mu       sync.RWMutex
}

func NewInMemEntityStore() *InMemEntityStore {
	return &InMemEntityStore{
		entities: make(map[string]Entity),
	}
}

func (s *InMemEntityStore) Add(ctx context.Context, item *MemoryItem) error {
	// Internal helper for entity extraction hits
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if name, ok := item.Metadata["entity"].(string); ok {
		desc := ""
		if d, ok := item.Metadata["description"].(string); ok {
			desc = d
		}
		s.entities[strings.ToLower(name)] = Entity{
			Name:        name,
			Value:       item.Text,
			Description: desc,
		}
	}
	return nil
}

func (s *InMemEntityStore) Search(ctx context.Context, query string, limit int) ([]*MemoryItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*MemoryItem
	query = strings.ToLower(query)
	
	for name, entity := range s.entities {
		if strings.Contains(name, query) || strings.Contains(strings.ToLower(entity.Value), query) {
			results = append(results, &MemoryItem{
				Text: fmt.Sprintf("%s: %s (%s)", entity.Name, entity.Value, entity.Description),
				Metadata: map[string]interface{}{
					"type": "entity",
					"name": entity.Name,
				},
			})
			if len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}

func (s *InMemEntityStore) Upsert(ctx context.Context, name, value, description string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entities[strings.ToLower(name)] = Entity{
		Name:        name,
		Value:       value,
		Description: description,
	}
	return nil
}

func (s *InMemEntityStore) Get(ctx context.Context, name string) (*Entity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if e, ok := s.entities[strings.ToLower(name)]; ok {
		return &e, nil
	}
	return nil, fmt.Errorf("entity not found: %s", name)
}
