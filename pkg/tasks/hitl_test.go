package tasks

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/tools"
)

type fakeAgent struct{ role string }

func (f *fakeAgent) GetRole() string                 { return f.role }
func (f *fakeAgent) GetGoal() string                 { return "" }
func (f *fakeAgent) GetBackstory() string            { return "" }
func (f *fakeAgent) GetMaxRPM() int                  { return 0 }
func (f *fakeAgent) SetMaxRPM(int)                   {}
func (f *fakeAgent) GetUsageMetrics() map[string]int { return nil }
func (f *fakeAgent) GetToolCount() int               { return 0 }
func (f *fakeAgent) Equip(t ...tools.Tool)           {}
func (f *fakeAgent) Execute(ctx context.Context, input string, options map[string]interface{}) (interface{}, error) {
	return "done", nil
}

var _ core.Agent = (*fakeAgent)(nil)

func TestPreHumanFeedbackInjected(t *testing.T) {
	var out bytes.Buffer
	tk := &Task{
		Description: "do thing",
		Agent:       &fakeAgent{role: "r"},
		HumanInput:  true,
		Stdin:       strings.NewReader("make it blue\n"),
		Stdout:      &out,
	}
	got := tk.applyPreHumanFeedback("do thing")
	if !strings.Contains(got, "HUMAN FEEDBACK OVERRIDE: make it blue") {
		t.Fatalf("feedback not folded: %q", got)
	}
}

func TestPreHumanApproveEmpty(t *testing.T) {
	var out bytes.Buffer
	tk := &Task{
		Description: "do thing",
		Agent:       &fakeAgent{role: "r"},
		HumanInput:  true,
		Stdin:       strings.NewReader("\n"),
		Stdout:      &out,
	}
	if got := tk.applyPreHumanFeedback("do thing"); got != "do thing" {
		t.Fatalf("empty approval changed description: %q", got)
	}
}

func TestPostHumanEditOverride(t *testing.T) {
	var out bytes.Buffer
	tk := &Task{
		Agent:      &fakeAgent{role: "r"},
		HumanInput: true,
		Stdin:      strings.NewReader("edit\nfinal answer\nEOF\n"),
		Stdout:     &out,
	}
	got := tk.applyPostHumanReview("draft")
	if got != "final answer" {
		t.Fatalf("override = %v", got)
	}
	if tk.Output != "final answer" {
		t.Fatalf("task output = %v", tk.Output)
	}
}
