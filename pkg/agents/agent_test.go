package agents

import (
	"context"
	"errors"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/i18n"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tools"
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

type mockTool struct {
	name string
	executeFunc func(ctx context.Context, input map[string]interface{}) (string, error)
}

func (m *mockTool) Name() string { return m.name }
func (m *mockTool) Description() string { return m.name }
func (m *mockTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	return "", nil
}
func (m *mockTool) RequiresReview() bool { return false }
func (m *mockTool) ArgsSchema() []tools.ArgSchema { return nil }
func (m *mockTool) CacheFunction(input map[string]interface{}) string { return "" }

func TestAgentExecute_Basic(t *testing.T) {
	mock := &mockLLM{}
	agent := &Agent{
		Role: "Tester",
		Goal: "Test the agent",
		LLM:  mock,
	}
	i18nInst, _ := i18n.NewI18N("en")
	agent.I18N = i18nInst
	result, err := agent.Execute(context.Background(), "Hello", nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

func TestAgentExecute_SelfHealing(t *testing.T) {
	mock := &mockLLM{}
	failingTool := &mockTool{
		name: "FailingTool",
		executeFunc: func(ctx context.Context, input map[string]interface{}) (string, error) {
			return "", errors.New("tool injection failure")
		},
	}
	agent := &Agent{
		Role:        "Healer",
		Goal:        "Test self-healing",
		LLM:         mock,
		Tools:       []tools.Tool{failingTool},
		SelfHealing: true,
		MaxIterations: 3,
	}
	i18nInst, _ := i18n.NewI18N("en")
	agent.I18N = i18nInst
	_, _ = agent.Execute(context.Background(), "Heal me", nil)
}

func TestAgentExecute_WithReview(t *testing.T) {
	tool := &mockTool{name: "ReviewTool"}
	agent := &Agent{
		Role: "HITLTester",
		LLM:  &mockLLM{},
		Tools: []tools.Tool{tool},
		UsageMetrics: make(map[string]int),
		StepReview: func(toolName string, input interface{}) bool {
			return true
		},
	}
	i18nInst, _ := i18n.NewI18N("en")
	agent.I18N = i18nInst
	result, err := agent.Execute(context.Background(), "Hello", nil)
	if err != nil {
		t.Fatalf("Expected no error with step review, got %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty result")
	}
}
