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
	PassRate     float64      `json:"pass_rate"`
}

// ============================================================
// Evaluator
// ============================================================

type EvaluatorConfig struct {
	ExpectedSchema string
	Rubric         string
	JudgeLLM       llm.Client
	Runs           int
}

// Evaluator uses an LLM to score agent outputs against expected criteria.
type Evaluator struct {
	LLM    llm.Client
	Config EvaluatorConfig
}

// NewEvaluator creates a new multi-run test evaluator.
func NewEvaluator(cfg EvaluatorConfig) *Evaluator {
	if cfg.Runs <= 0 {
		cfg.Runs = 1
	}
	return &Evaluator{
		LLM:    cfg.JudgeLLM,
		Config: cfg,
	}
}

// Orchestrator represents any crew or flow workflow that can be kicked off.
type Orchestrator interface {
	Kickoff(ctx context.Context) (interface{}, error)
}

// EvaluateCrew runs an orchestrator multiple times and returns aggregated metrics.
func (e *Evaluator) EvaluateCrew(ctx context.Context, app Orchestrator) (*PerformanceSuite, error) {
	if e.LLM == nil {
		return nil, fmt.Errorf("evaluation LLM not configured")
	}

	suite := &PerformanceSuite{
		Results: make([]TestResult, 0, e.Config.Runs),
	}

	var totalScore int
	var totalDuration time.Duration

	for i := 0; i < e.Config.Runs; i++ {
		start := time.Now()
		
		// Run the crew
		output, err := app.Kickoff(ctx)
		duration := time.Since(start)

		if err != nil {
			return nil, fmt.Errorf("evaluation run %d failed: %w", i+1, err)
		}

		outStr := fmt.Sprintf("%v", output)
		
		// Score the run
		score, feedback, err := e.ScoreResult(ctx, e.Config.Rubric, e.Config.ExpectedSchema, outStr)
		if err != nil {
			return nil, fmt.Errorf("failed to score run %d: %w", i+1, err)
		}

		res := TestResult{
			Iteration:    i + 1,
			Duration:     duration,
			Score:        score,
			Feedback:     feedback,
			OutputSample: outStr,
		}

		suite.Results = append(suite.Results, res)
		totalScore += score
		totalDuration += duration
	}

	if e.Config.Runs > 0 {
		suite.AverageScore = float64(totalScore) / float64(e.Config.Runs)
		suite.AverageTime = time.Duration(int64(totalDuration) / int64(e.Config.Runs))
	}

	// Calculate custom pass rate metric easily extrapolated
	var passes int
	for _, res := range suite.Results {
		if res.Score >= 8 { // Typical benchmark for "Pass"
			passes++
		}
	}
	
	// Dynamically attach pass rate to the suite
	suite.PassRate = float64(passes) / float64(e.Config.Runs)

	return suite, nil
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
	}, llm.GenerateOptions{})

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
