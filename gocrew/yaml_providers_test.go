package gocrew

import (
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/llm"
)

func TestBuildLLM_AllProviders(t *testing.T) {
	for _, p := range []string{"openai", "anthropic", "gemini", "groq", "openrouter", "ollama"} {
		c, err := buildLLM(&YAMLLLMConfig{Provider: p, Model: "test-model", APIKey: "k"})
		if err != nil {
			t.Errorf("provider %s: %v", p, err)
			continue
		}
		if c == nil {
			t.Errorf("provider %s: nil client", p)
		}
	}
}

func TestBuildLLM_Failover(t *testing.T) {
	c, err := buildLLM(&YAMLLLMConfig{
		Provider:          "failover",
		FailoverPrimary:   &YAMLLLMConfig{Provider: "openai", Model: "m1", APIKey: "k"},
		FailoverSecondary: &YAMLLLMConfig{Provider: "ollama", Model: "m2"},
	})
	if err != nil {
		t.Fatalf("failover: %v", err)
	}
	if _, ok := c.(*llm.FailoverClient); !ok {
		t.Errorf("expected *llm.FailoverClient, got %T", c)
	}
}

func TestBuildLLM_FailoverMissing(t *testing.T) {
	if _, err := buildLLM(&YAMLLLMConfig{Provider: "failover"}); err == nil {
		t.Error("expected error for failover without primary/secondary")
	}
}

func TestBuildLLM_Unsupported(t *testing.T) {
	if _, err := buildLLM(&YAMLLLMConfig{Provider: "nope"}); err == nil {
		t.Error("expected error for unsupported provider")
	}
}

func TestBuildLLM_DefaultsWrapped(t *testing.T) {
	c, err := buildLLM(&YAMLLLMConfig{Provider: "openai", Model: "m", APIKey: "k", Temperature: 0.5, MaxTokens: 100})
	if err != nil {
		t.Fatalf("defaults: %v", err)
	}
	if _, ok := c.(*llm.DefaultsClient); !ok {
		t.Errorf("expected *llm.DefaultsClient, got %T", c)
	}
	// Without defaults set, no wrapper.
	c2, err := buildLLM(&YAMLLLMConfig{Provider: "openai", Model: "m", APIKey: "k"})
	if err != nil {
		t.Fatalf("plain: %v", err)
	}
	if _, ok := c2.(*llm.DefaultsClient); ok {
		t.Errorf("expected bare client without defaults, got wrapper")
	}
}

func TestLoadFromYAMLString_Providers(t *testing.T) {
	yml := `
llm:
  provider: ollama
  model: llama3
  temperature: 0.7
  max_tokens: 512
agents:
  - role: researcher
    goal: research
    backstory: backstory
tasks:
  - name: t1
    description: "do {{thing}}"
    expected_output: out
    agent: researcher
crew:
  process: sequential
`
	c, err := LoadFromYAMLString(yml)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c == nil {
		t.Fatal("nil crew")
	}
}
