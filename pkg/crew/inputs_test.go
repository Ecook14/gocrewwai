package crew

import (
	"context"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/i18n"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
)

func TestInterpolateInputs_Basic(t *testing.T) {
	out, err := InterpolateInputs("Research {{topic}} in {{lang}}", map[string]string{"topic": "AI", "lang": "Go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "Research AI in Go" {
		t.Errorf("got %q", out)
	}
}

func TestInterpolateInputs_MissingFails(t *testing.T) {
	if _, err := InterpolateInputs("Research {{topic}}", map[string]string{}); err == nil {
		t.Error("expected error for missing placeholder")
	}
	if _, err := InterpolateInputs("Research {{topic}", map[string]string{"topic": "x"}); err == nil {
		t.Error("expected error for unterminated placeholder")
	}
}

func TestCrew_KickoffWithInputs(t *testing.T) {
	agent := &agents.Agent{Role: "researcher", LLM: &mockLLM{}}
	i18nInst, _ := i18n.NewI18N("en")
	agent.I18N = i18nInst
	task := &tasks.Task{Description: "Research {{topic}}", ExpectedOutput: "Report on {{topic}}", Agent: agent}
	task.I18N, _ = i18n.NewI18N("en")
	c := NewCrew([]core.Agent{agent}, []*tasks.Task{task})
	if _, err := c.KickoffWithInputs(context.Background(), map[string]string{"topic": "agents"}); err != nil {
		t.Fatalf("kickoff: %v", err)
	}
	if task.Description != "Research agents" {
		t.Errorf("description = %q", task.Description)
	}
	if task.ExpectedOutput != "Report on agents" {
		t.Errorf("expected_output = %q", task.ExpectedOutput)
	}
}

func TestCrew_KickoffWithInputs_MissingFails(t *testing.T) {
	agent := &agents.Agent{Role: "researcher", LLM: &mockLLM{}}
	i18nInst, _ := i18n.NewI18N("en")
	agent.I18N = i18nInst
	task := &tasks.Task{Description: "Research {{topic}}", Agent: agent}
	task.I18N, _ = i18n.NewI18N("en")
	c := NewCrew([]core.Agent{agent}, []*tasks.Task{task})
	if _, err := c.KickoffWithInputs(context.Background(), map[string]string{}); err == nil {
		t.Error("expected error for missing input")
	}
}
