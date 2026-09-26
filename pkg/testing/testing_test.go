package testing

import (
	"context"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/testutil"
)

type stubRunner struct{ out string }

func (s stubRunner) Kickoff(ctx context.Context) (interface{}, error) { return s.out, nil }

func TestCrewTestChaining(t *testing.T) {
	ct := NewCrewTest(testutil.NewSimpleMock("SCORE: 9\nFEEDBACK: good"))
	same := ct.WithRuns(3).WithScorer("rubric").WithPassThreshold(8).WithTimeout(1000000000)
	if same != ct {
		t.Fatal("chaining must return same pointer")
	}
}

func TestCrewTestRunRequiresJudgeAndRubric(t *testing.T) {
	if _, err := NewCrewTest(nil).WithScorer("r").Run(context.Background(), stubRunner{"x"}); err == nil {
		t.Fatal("expected no-judge error")
	}
	if _, err := NewCrewTest(testutil.NewSimpleMock("x")).Run(context.Background(), stubRunner{"x"}); err == nil {
		t.Fatal("expected no-rubric error")
	}
}

func TestScoreResultWithMockJudge(t *testing.T) {
	e := NewEvaluator(EvaluatorConfig{JudgeLLM: testutil.NewSimpleMock("SCORE: 9\nFEEDBACK: great work")})
	score, fb, err := e.ScoreResult(context.Background(), "task", "expected", "output")
	if err != nil {
		t.Fatalf("score: %v", err)
	}
	if score != 9 {
		t.Fatalf("score = %d, want 9", score)
	}
	if fb != "great work" {
		t.Fatalf("feedback = %q", fb)
	}
}

func TestEvaluateCrewAggregates(t *testing.T) {
	e := NewEvaluator(EvaluatorConfig{
		JudgeLLM: testutil.NewSimpleMock("SCORE: 8\nFEEDBACK: ok"),
		Runs:     2,
	})
	suite, err := e.EvaluateCrew(context.Background(), stubRunner{"out"})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if len(suite.Results) != 2 {
		t.Fatalf("results = %d, want 2", len(suite.Results))
	}
	if suite.AverageScore != 8 {
		t.Fatalf("avg = %v, want 8", suite.AverageScore)
	}
}

func TestSplitLines(t *testing.T) {
	got := splitLines("a\nb\nc")
	if len(got) != 3 || got[1] != "b" {
		t.Fatalf("splitLines = %q", got)
	}
}
