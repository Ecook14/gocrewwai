package crew

import (
	"context"
	"fmt"

	"github.com/Ecook14/gocrewwai/pkg/llm"
	crewtesting "github.com/Ecook14/gocrewwai/pkg/testing"
)

// Test runs the crew through LLM-judged evaluation: the crew kicks off
// `runs` times and a judge LLM scores each output against rubric.
// It is the programmatic counterpart of `gocrew test` (CrewAI `Crew.test()`).
func (c *Crew) Test(ctx context.Context, judge llm.Client, runs int, rubric string) (*crewtesting.PerformanceSuite, error) {
	if judge == nil {
		return nil, fmt.Errorf("crew: Test requires a judge LLM client")
	}
	if runs <= 0 || runs > 100 {
		return nil, fmt.Errorf("crew: Test runs must be 1-100, got %d", runs)
	}
	evaluator := crewtesting.NewEvaluator(crewtesting.EvaluatorConfig{
		JudgeLLM: judge,
		Runs:     runs,
		Rubric:   rubric,
	})
	return evaluator.EvaluateCrew(ctx, c)
}
