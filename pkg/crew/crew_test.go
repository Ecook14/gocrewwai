package crew

import (
	"context"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/i18n"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
)

type mockLLM struct {
	generateFunc func(ctx context.Context, messages []llm.Message, options llm.GenerateOptions) (string, error)
}

func (m *mockLLM) Generate(ctx context.Context, messages []llm.Message, options llm.GenerateOptions) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, messages, options)
	}
	return "Success", nil
}
func (m *mockLLM) GenerateWithUsage(ctx context.Context, messages []llm.Message, options llm.GenerateOptions) (string, *llm.Usage, error) { return "", nil, nil }
func (m *mockLLM) GenerateStructured(ctx context.Context, messages []llm.Message, schema interface{}, options llm.GenerateOptions) (interface{}, error) { return nil, nil }
func (m *mockLLM) StreamGenerate(ctx context.Context, messages []llm.Message, options llm.GenerateOptions) (<-chan string, error) { return nil, nil }
func (m *mockLLM) GenerateEmbedding(ctx context.Context, text string, options llm.GenerateOptions) ([]float32, error) { return nil, nil }

func TestCrewKickoff_Sequential(t *testing.T) {
	mock := &mockLLM{}
	agent := &agents.Agent{
		Role: "Worker",
		LLM:  mock,
	}
	i18nInst, _ := i18n.NewI18N("en")
	agent.I18N = i18nInst
	task := &tasks.Task{Description: "Job", Agent: agent}
	c := &Crew{
		Agents:   []core.Agent{agent},
		Tasks:    []*tasks.Task{task},
		Planning: true,
		Process:  Sequential,
	}
	_, err := c.Kickoff(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestCrewKickoff_Planning(t *testing.T) {
	mock := &mockLLM{}
	agent := &agents.Agent{
		Role: "Worker",
		LLM:  mock,
	}
	i18nInst, _ := i18n.NewI18N("en")
	agent.I18N = i18nInst
	task := &tasks.Task{Description: "Job", Agent: agent}
	c := &Crew{
		Agents:   []core.Agent{agent},
		Tasks:    []*tasks.Task{task},
		Planning: true,
		Process:  Sequential,
	}
	_, err := c.Kickoff(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestCrewNew(t *testing.T) {
	mock := &mockLLM{}
	agent := &agents.Agent{
		Role: "Worker",
		LLM:  mock,
	}
	i18nInst, _ := i18n.NewI18N("en")
	agent.I18N = i18nInst
	task := &tasks.Task{Description: "Job", Agent: agent}
	c := NewCrew([]core.Agent{agent}, []*tasks.Task{task})
	if c == nil {
		t.Fatal("Expected non-nil crew")
	}
	if len(c.Agents) != 1 {
		t.Errorf("Expected 1 agent, got %d", len(c.Agents))
	}
}
