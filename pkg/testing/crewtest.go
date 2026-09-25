package testing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/llm"
)

// CrewTest represents a multi-run test harness for evaluating crew or flow performance.
type CrewTest struct {
	runs           int
	rubric         string
	expectedSchema string
	judgeLLM       llm.Client
	passThreshold  int
	timeout        time.Duration
	tracCompare    func(r1, r2 *TestResult) bool
}

// NewCrewTest creates a new crew test harness.
// It requires a judge LLM for scoring outputs. Pass nil and it will be set later via WithJudge.
func NewCrewTest(judgeLLM llm.Client) *CrewTest {
	return &CrewTest{
		runs:          1,
		passThreshold: 7,
		timeout:       5 * time.Minute,
	}
}

// WithRuns sets the number of times to execute the crew per test.
func (t *CrewTest) WithRuns(n int) *CrewTest {
	if n < 1 {
		n = 1
	}
	t.runs = n
	return t
}

// WithScorer sets the evaluation rubric and expected output schema.
// The rubric is a natural-language description of what a "good" output looks like.
// The expectedSchema is an optional JSON schema or struct description.
func (t *CrewTest) WithScorer(rubric string, expectedSchema ...string) *CrewTest {
	t.rubric = rubric
	if len(expectedSchema) > 0 {
		t.expectedSchema = expectedSchema[0]
	}
	return t
}

// WithJudge sets the LLM used to score outputs. Overrides any judge set at construction.
func (t *CrewTest) WithJudge(judge llm.Client) *CrewTest {
	t.judgeLLM = judge
	return t
}

// WithTimeout sets the per-run timeout. Default is 5 minutes.
func (t *CrewTest) WithTimeout(d time.Duration) *CrewTest {
	t.timeout = d
	return t
}

// WithPassThreshold sets the minimum score (1-10) to count as a pass. Default is 7.
func (t *CrewTest) WithPassThreshold(threshold int) *CrewTest {
	t.passThreshold = threshold
	return t
}

// WithTraceCompare sets a custom function to compare two test results for trace-level consistency.
// Return true if the two runs produced equivalent traces.
func (t *CrewTest) WithTraceCompare(fn func(r1, r2 *TestResult) bool) *CrewTest {
	t.tracCompare = fn
	return t
}

// Run executes the crew multiple times, scores each run, and returns a PerformanceSuite.
func (t *CrewTest) Run(ctx context.Context, crew KickoffRunner) (*PerformanceSuite, error) {
	if t.judgeLLM == nil {
		return nil, fmt.Errorf("no judge LLM configured — use WithJudge or pass one to NewCrewTest")
	}
	if t.rubric == "" {
		return nil, fmt.Errorf("no scoring rubric set — use WithScorer")
	}

	evaluator := NewEvaluator(EvaluatorConfig{
		JudgeLLM:       t.judgeLLM,
		Rubric:         t.rubric,
		ExpectedSchema: t.expectedSchema,
		Runs:           t.runs,
	})

	// Wrap the crew so it respects per-run timeout
	crewWithTimeout := &timeoutCrew{
		inner:   crew,
		timeout: t.timeout,
	}

	return evaluator.EvaluateCrew(ctx, crewWithTimeout)
}

// KickoffRunner is any type that can execute a crew/flow and return a result.
type KickoffRunner interface {
	Kickoff(ctx context.Context) (interface{}, error)
}

// timeoutCrew wraps a KickoffRunner with a per-run timeout.
type timeoutCrew struct {
	inner   KickoffRunner
	timeout time.Duration
}

func (c *timeoutCrew) Kickoff(ctx context.Context) (interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.inner.Kickoff(ctx)
}

// ScoreThresholdLinter checks that all results meet the pass threshold.
// Returns a human-readable summary of failures.
func ScoreThresholdLinter(suite *PerformanceSuite, threshold int) string {
	var failures []string
	for _, r := range suite.Results {
		if r.Score < threshold {
			failures = append(failures, fmt.Sprintf("run %d: score %d (below %d)", r.Iteration, r.Score, threshold))
		}
	}
	if len(failures) == 0 {
		return "All runs passed the threshold."
	}
	return strings.Join(failures, "; ")
}

// ScoreSummary returns a one-line summary of the suite.
func ScoreSummary(suite *PerformanceSuite) string {
	return fmt.Sprintf("avg=%.1f pass_rate=%.0f%% runs=%d", suite.AverageScore, suite.PassRate*100, len(suite.Results))
}

// TraceCompareAll runs the trace comparison function across all adjacent pairs of results.
// Returns true if every adjacent pair matches according to the comparator.
func TraceCompareAll(suite *PerformanceSuite, cmp func(r1, r2 TestResult) bool) bool {
	if len(suite.Results) < 2 {
		return true
	}
	for i := 1; i < len(suite.Results); i++ {
		if !cmp(suite.Results[i-1], suite.Results[i]) {
			return false
		}
	}
	return true
}
