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

type asyncMockLLM struct {
	generateFunc func(ctx context.Context, messages []llm.Message, options llm.GenerateOptions) (string, error)
	response     string
}

func (m *asyncMockLLM) Generate(ctx context.Context, messages []llm.Message, options llm.GenerateOptions) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, messages, options)
	}
	return m.response, nil
}
func (m *asyncMockLLM) GenerateWithUsage(ctx context.Context, messages []llm.Message, options llm.GenerateOptions) (string, *llm.Usage, error) { return "", nil, nil }
func (m *asyncMockLLM) GenerateStructured(ctx context.Context, messages []llm.Message, schema interface{}, options llm.GenerateOptions) (interface{}, error) { return nil, nil }
func (m *asyncMockLLM) StreamGenerate(ctx context.Context, messages []llm.Message, options llm.GenerateOptions) (<-chan string, error) { return nil, nil }
func (m *asyncMockLLM) GenerateEmbedding(ctx context.Context, text string, options llm.GenerateOptions) ([]float32, error) { return nil, nil }

func TestAsyncExecution(t *testing.T) {
	mock := &asyncMockLLM{response: "async-result"}
	agent := &agents.Agent{
		Role: "Tester",
		LLM:  mock,
	}
	i18nInst, _ := i18n.NewI18N("en")
	agent.I18N = i18nInst
	task := &tasks.Task{Description: "Test async execution", Agent: agent}
	crew := NewCrew([]core.Agent{agent}, []*tasks.Task{task})
	future := crew.KickoffAsync(context.Background())
	_, err := future.Result()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}
