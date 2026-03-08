package crew

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"
)

// ---------------------------------------------------------------------------
// Training Data Export & Evaluation
// ---------------------------------------------------------------------------

// TrainingExample represents a single training data point.
type TrainingExample struct {
	TaskDescription string `json:"task_description"`
	AgentRole       string `json:"agent_role"`
	Input           string `json:"input"`
	Output          string `json:"output"`
	Feedback        string `json:"feedback,omitempty"`   // Human correction
	ToolsUsed       []string `json:"tools_used,omitempty"`
	TokensUsed      int    `json:"tokens_used,omitempty"`
	LatencyMs       int64  `json:"latency_ms,omitempty"`
	Timestamp       string `json:"timestamp"`
}

// TrainingDataset holds a collection of training examples with metadata.
type TrainingDataset struct {
	Name      string            `json:"name"`
	Version   string            `json:"version"`
	CreatedAt string            `json:"created_at"`
	Examples  []TrainingExample `json:"examples"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// NewTrainingDataset creates an empty dataset.
func NewTrainingDataset(name string) *TrainingDataset {
	return &TrainingDataset{
		Name:      name,
		Version:   "1.0",
		CreatedAt: time.Now().Format(time.RFC3339),
		Examples:  make([]TrainingExample, 0),
		Metadata:  make(map[string]string),
	}
}

// AddExample appends a training example.
func (ds *TrainingDataset) AddExample(ex TrainingExample) {
	if ex.Timestamp == "" {
		ex.Timestamp = time.Now().Format(time.RFC3339)
	}
	ds.Examples = append(ds.Examples, ex)
}

// SaveJSON exports the dataset to a JSON file.
func (ds *TrainingDataset) SaveJSON(path string) error {
	data, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dataset: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// SaveJSONL exports to JSON Lines format (one example per line).
func (ds *TrainingDataset) SaveJSONL(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, ex := range ds.Examples {
		data, err := json.Marshal(ex)
		if err != nil {
			continue
		}
		f.Write(data)
		f.WriteString("\n")
	}
	return nil
}

// SaveOpenAIFormat exports in OpenAI fine-tuning format.
func (ds *TrainingDataset) SaveOpenAIFormat(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, ex := range ds.Examples {
		entry := map[string]interface{}{
			"messages": []map[string]string{
				{"role": "system", "content": fmt.Sprintf("You are %s.", ex.AgentRole)},
				{"role": "user", "content": ex.Input},
				{"role": "assistant", "content": ex.Output},
			},
		}
		data, _ := json.Marshal(entry)
		f.Write(data)
		f.WriteString("\n")
	}
	return nil
}

// LoadDataset reads a training dataset from JSON.
func LoadDataset(path string) (*TrainingDataset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ds TrainingDataset
	if err := json.Unmarshal(data, &ds); err != nil {
		return nil, err
	}
	return &ds, nil
}

// ---------------------------------------------------------------------------
// Evaluation Metrics
// ---------------------------------------------------------------------------

// EvalResult holds evaluation metrics for a training dataset.
type EvalResult struct {
	TotalExamples     int     `json:"total_examples"`
	WithFeedback      int     `json:"with_feedback"`     // Examples that needed correction
	AccuracyRate      float64 `json:"accuracy_rate"`     // % accepted without feedback
	AvgTokensUsed     float64 `json:"avg_tokens_used"`
	AvgLatencyMs      float64 `json:"avg_latency_ms"`
	TotalTokens       int     `json:"total_tokens"`
	UniqueAgents      int     `json:"unique_agents"`
	UniqueTools       int     `json:"unique_tools"`
	ToolUsageRate     float64 `json:"tool_usage_rate"`   // % of examples using tools
	CostEstimateUSD   float64 `json:"cost_estimate_usd"` // Estimated at $0.01/1K tokens
}

// Evaluate computes metrics for the dataset.
func (ds *TrainingDataset) Evaluate() EvalResult {
	result := EvalResult{
		TotalExamples: len(ds.Examples),
	}

	if result.TotalExamples == 0 {
		return result
	}

	agents := make(map[string]bool)
	allTools := make(map[string]bool)
	toolUsageCount := 0
	totalLatency := int64(0)

	for _, ex := range ds.Examples {
		if ex.Feedback != "" {
			result.WithFeedback++
		}
		result.TotalTokens += ex.TokensUsed
		totalLatency += ex.LatencyMs
		agents[ex.AgentRole] = true
		if len(ex.ToolsUsed) > 0 {
			toolUsageCount++
			for _, t := range ex.ToolsUsed {
				allTools[t] = true
			}
		}
	}

	result.AccuracyRate = float64(result.TotalExamples-result.WithFeedback) / float64(result.TotalExamples) * 100
	result.AvgTokensUsed = float64(result.TotalTokens) / float64(result.TotalExamples)
	result.AvgLatencyMs = float64(totalLatency) / float64(result.TotalExamples)
	result.UniqueAgents = len(agents)
	result.UniqueTools = len(allTools)
	result.ToolUsageRate = float64(toolUsageCount) / float64(result.TotalExamples) * 100
	result.CostEstimateUSD = math.Round(float64(result.TotalTokens)/1000*0.01*100) / 100

	// Round percentages
	result.AccuracyRate = math.Round(result.AccuracyRate*100) / 100
	result.ToolUsageRate = math.Round(result.ToolUsageRate*100) / 100

	return result
}

// EvalReport generates a human-readable evaluation report.
func (ds *TrainingDataset) EvalReport() string {
	eval := ds.Evaluate()
	return fmt.Sprintf(`Training Evaluation Report: %s
═══════════════════════════════════════
Total Examples:     %d
Accuracy Rate:      %.1f%% (accepted without correction)
Corrections Needed: %d

Token Usage:
  Total Tokens:     %d
  Avg per Example:  %.0f
  Cost Estimate:    $%.2f

Performance:
  Avg Latency:      %.0f ms
  Tool Usage Rate:  %.1f%%
  Unique Agents:    %d
  Unique Tools:     %d
═══════════════════════════════════════`,
		ds.Name,
		eval.TotalExamples,
		eval.AccuracyRate, eval.WithFeedback,
		eval.TotalTokens, eval.AvgTokensUsed, eval.CostEstimateUSD,
		eval.AvgLatencyMs, eval.ToolUsageRate, eval.UniqueAgents, eval.UniqueTools,
	)
}
