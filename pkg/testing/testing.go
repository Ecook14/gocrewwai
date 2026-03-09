package testing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/llm"
)

// ============================================================
// Testing Data Structures
// ============================================================

// TestResult captures a single execution run metrics.
type TestResult struct {
	Iteration    int           `json:"iteration"`
	Duration     time.Duration `json:"duration"`
	Score        int           `json:"score"`         // 1-10
	Feedback     string        `json:"feedback"`
	OutputSample string        `json:"output_sample"`
}

// PerformanceSuite aggregates metrics across multiple test runs.
type PerformanceSuite struct {
	Results      []TestResult `json:"results"`
	AverageScore float64      `json:"average_score"`
	AverageTime  time.Duration `json:"average_time"`
	TotalTokens  int          `json:"total_tokens"`
}

// ============================================================
// Evaluator
// ============================================================

// Evaluator uses an LLM to score agent outputs against expected criteria.
type Evaluator struct {
	LLM llm.Client
}

// ScoreResult evaluates a single task output.
func (e *Evaluator) ScoreResult(ctx context.Context, taskDesc, expected, output string) (int, string, error) {
	if e.LLM == nil {
		return 0, "", fmt.Errorf("evaluation LLM not configured")
	}

	prompt := fmt.Sprintf(`Evaluate the following AI agent output based on the provided task and expected outcome.
TASK: %s
EXPECTED: %s
ACTUAL OUTPUT: %s

Provide your evaluation in exactly this format:
SCORE: [1-10]
FEEDBACK: [1-2 sentences explaining the score]`, taskDesc, expected, output)

	response, err := e.LLM.Generate(ctx, []llm.Message{
		{Role: "system", Content: "You are an expert AI quality assurance judge."},
		{Role: "user", Content: prompt},
	}, nil)

	if err != nil {
		return 0, "", err
	}

	score := 0
	feedback := ""
	fmt.Sscanf(response, "SCORE: %d", &score)
	
	feedbackIdx := fmt.Sprintln("FEEDBACK:")
	_ = feedbackIdx // Placeholder for parsing logic
	
	// Basic parsing
	lines := splitLines(response)
	for _, line := range lines {
		if strings.HasPrefix(strings.ToUpper(line), "FEEDBACK:") {
			feedback = strings.TrimSpace(line[9:])
		}
	}

	return score, feedback, nil
}

func splitLines(s string) []string {
	// Simple helper for line splitting
	return strings.Split(s, "\n")
}
