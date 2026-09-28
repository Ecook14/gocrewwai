package crew

import (
	"context"
	"testing"
)

func TestCrew_TestRequiresJudge(t *testing.T) {
	c := NewCrew(nil, nil)
	if _, err := c.Test(context.Background(), nil, 1, "rubric"); err == nil {
		t.Error("expected error for nil judge")
	}
}

func TestCrew_TestBadRuns(t *testing.T) {
	c := NewCrew(nil, nil)
	if _, err := c.Test(context.Background(), &mockLLM{}, 0, "rubric"); err == nil {
		t.Error("expected error for runs=0")
	}
	if _, err := c.Test(context.Background(), &mockLLM{}, 101, "rubric"); err == nil {
		t.Error("expected error for runs=101")
	}
}
