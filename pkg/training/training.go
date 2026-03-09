package training

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ============================================================
// Training Data Structures
// ============================================================

// IterationData captures a single training pass.
type IterationData struct {
	InitialOutput string `json:"initial_output"`
	HumanFeedback string `json:"human_feedback"`
	ImprovedOutput string `json:"improved_output"`
}

// AgentTrainingData stores iterations for a specific agent role.
type AgentTrainingData struct {
	Iterations   []IterationData `json:"iterations"`
	Suggestions  []string        `json:"suggestions"`
	QualityScore float64         `json:"quality_score"`
	Summary      string          `json:"summary"`
}

// ============================================================
// Training Store
// ============================================================

// Store manages persistence of training results.
type Store struct {
	Dir string
	mu  sync.RWMutex
}

// NewStore creates a new training data store.
func NewStore(dir string) *Store {
	os.MkdirAll(dir, 0755)
	return &Store{Dir: dir}
}

// SaveAgentData persists training results for an agent role.
func (s *Store) SaveAgentData(role string, data *AgentTrainingData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := filepath.Join(s.Dir, fmt.Sprintf("%s.json", role))
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, bytes, 0644)
}

// LoadAgentData retrieves training results for an agent role.
func (s *Store) LoadAgentData(role string) (*AgentTrainingData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filename := filepath.Join(s.Dir, fmt.Sprintf("%s.json", role))
	bytes, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return &AgentTrainingData{}, nil
		}
		return nil, err
	}

	var data AgentTrainingData
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// ============================================================
// Training Engine Helpers
// ============================================================

// ConsolidateFeedback uses an LLM-like logic (or simple heuristic) to distill suggestions.
func ConsolidateFeedback(data *AgentTrainingData) {
	if len(data.Iterations) == 0 {
		return
	}

	uniqueSuggestions := make(map[string]bool)
	for _, it := range data.Iterations {
		if it.HumanFeedback != "" {
			uniqueSuggestions[it.HumanFeedback] = true
		}
	}

	data.Suggestions = nil
	for s := range uniqueSuggestions {
		data.Suggestions = append(data.Suggestions, s)
	}

	data.Summary = fmt.Sprintf("Agent improved over %d iterations based on human feedback.", len(data.Iterations))
	data.QualityScore = 8.5 // Default placeholder for now
}
