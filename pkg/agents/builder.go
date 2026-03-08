package agents

import (
	"github.com/Ecook14/crewai-go/pkg/llm"
	"github.com/Ecook14/crewai-go/pkg/memory"
	"github.com/Ecook14/crewai-go/pkg/tools"
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

func (b *AgentBuilder) Build() *Agent {
	return b.agent
}
