package tasks

import (
	"context"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/i18n"
	"github.com/Ecook14/gocrewwai/pkg/llm"
)

type mockLLMClient struct {
	generateFunc func(ctx context.Context, messages []llm.Message, options llm.GenerateOptions) (string, error)
}

func (m *mockLLMClient) Generate(ctx context.Context, messages []llm.Message, options llm.GenerateOptions) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, messages, options)
	}
	return "Task Output", nil
}
func (m *mockLLMClient) GenerateWithUsage(ctx context.Context, messages []llm.Message, options llm.GenerateOptions) (string, *llm.Usage, error) { return "", nil, nil }
func (m *mockLLMClient) GenerateStructured(ctx context.Context, messages []llm.Message, schema interface{}, options llm.GenerateOptions) (interface{}, error) { return nil, nil }
func (m *mockLLMClient) StreamGenerate(ctx context.Context, messages []llm.Message, options llm.GenerateOptions) (<-chan string, error) { return nil, nil }
func (m *mockLLMClient) GenerateEmbedding(ctx context.Context, text string, options llm.GenerateOptions) ([]float32, error) { return nil, nil }

func TestTaskExecute(t *testing.T) {
	mockLLM := &mockLLMClient{}
	i18nInst, _ := i18n.NewI18N("en")
	agent := agents.NewAgent(agents.AgentConfig{
		Role:    "Tester",
		Goal:    "Test",
		Backstory: "Test agent",
		LLM:     mockLLM,
		Tools:   nil,
	})
	agent.I18N = i18nInst
	task := &Task{
		Description: "Perform test",
		Agent:       agent,
	}
	res, err := task.Execute(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if res != "Task Output" {
		t.Errorf("Expected 'Task Output', got %v", res)
	}
	if !task.Processed {
		t.Errorf("Expected task to be marked as processed")
	}
}

func TestTaskExecute_NoAgent(t *testing.T) {
	task := &Task{Description: "No agent test"}
	_, err := task.Execute(context.Background())
	if err == nil {
		t.Error("Expected error for task with no agent")
	}
}
