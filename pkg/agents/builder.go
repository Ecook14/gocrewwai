package agents

import (
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/memory"
	"github.com/Ecook14/gocrewwai/pkg/tools"
	"strings"
)

// AgentBuilder provides a fluent API for constructing Agents.
type AgentBuilder struct {
	agent *Agent
}

func NewAgentBuilder() *AgentBuilder {
	return &AgentBuilder{
		agent: &Agent{
			MaxIterations: 15,
			MaxRetryLimit: 3,
			UsageMetrics:  make(map[string]int),
		},
	}
}

func (b *AgentBuilder) Role(role string) *AgentBuilder {
	b.agent.Role = strings.Clone(role)
	return b
}

func (b *AgentBuilder) Goal(goal string) *AgentBuilder {
	b.agent.Goal = strings.Clone(goal)
	return b
}

func (b *AgentBuilder) Backstory(backstory string) *AgentBuilder {
	b.agent.Backstory = backstory
	return b
}

func (b *AgentBuilder) LLM(client llm.Client) *AgentBuilder {
	b.agent.LLM = client
	return b
}

func (b *AgentBuilder) Tools(t ...tools.Tool) *AgentBuilder {
	b.agent.Tools = append(b.agent.Tools, t...)
	return b
}

func (b *AgentBuilder) Memory(store memory.Store) *AgentBuilder {
	b.agent.Memory = store
	return b
}

func (b *AgentBuilder) EntityMemory(store memory.EntityStore) *AgentBuilder {
	b.agent.EntityMemory = store
	return b
}

func (b *AgentBuilder) Verbose(v bool) *AgentBuilder {
	b.agent.Verbose = v
	return b
}

func (b *AgentBuilder) SelfHealing(v bool) *AgentBuilder {
	b.agent.SelfHealing = v
	return b
}

func (b *AgentBuilder) AllowDelegation(v bool) *AgentBuilder {
	b.agent.AllowDelegation = v
	return b
}

func (b *AgentBuilder) Cache(cache llm.Cache) *AgentBuilder {
	b.agent.Cache = cache
	return b
}

func (b *AgentBuilder) FunctionCallingLLM(client llm.Client) *AgentBuilder {
	b.agent.FunctionCallingLLM = client
	return b
}

func (b *AgentBuilder) SystemTemplate(t string) *AgentBuilder {
	b.agent.SystemTemplate = t
	return b
}

func (b *AgentBuilder) PromptTemplate(t string) *AgentBuilder {
	b.agent.PromptTemplate = t
	return b
}

func (b *AgentBuilder) ResponseTemplate(t string) *AgentBuilder {
	b.agent.ResponseTemplate = t
	return b
}

func (b *AgentBuilder) AllowCodeExecution(v bool) *AgentBuilder {
	b.agent.AllowCodeExecution = v
	return b
}

func (b *AgentBuilder) CodeExecutionMode(mode string) *AgentBuilder {
	b.agent.CodeExecutionMode = mode
	return b
}

func (b *AgentBuilder) Multimodal(v bool) *AgentBuilder {
	b.agent.Multimodal = v
	return b
}

func (b *AgentBuilder) InjectDate(v bool) *AgentBuilder {
	b.agent.InjectDate = v
	return b
}

func (b *AgentBuilder) DateFormat(format string) *AgentBuilder {
	b.agent.DateFormat = format
	return b
}

func (b *AgentBuilder) Reasoning(v bool) *AgentBuilder {
	b.agent.Reasoning = v
	return b
}

func (b *AgentBuilder) MaxReasoningAttempts(max int) *AgentBuilder {
	b.agent.MaxReasoningAttempts = max
	return b
}

func (b *AgentBuilder) Embedder(cfg map[string]interface{}) *AgentBuilder {
	b.agent.EmbedderConfig = cfg
	return b
}

func (b *AgentBuilder) KnowledgeSources(sources ...memory.KnowledgeSource) *AgentBuilder {
	b.agent.KnowledgeSources = append(b.agent.KnowledgeSources, sources...)
	return b
}

func (b *AgentBuilder) UseSystemPrompt(v bool) *AgentBuilder {
	b.agent.UseSystemPrompt = v
	return b
}

func (b *AgentBuilder) Build() *Agent {
	return b.agent
}
