package agents

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/guardrails"
)

// ---------------------------------------------------------------------------
// Agent Cloning & Templating
// ---------------------------------------------------------------------------

// Clone creates a deep copy of the agent with an optional new role.
// The clone shares the same LLM client and tools but has independent state.
func (a *Agent) Clone(newRole string) *Agent {
	role := a.Role
	if newRole != "" {
		role = newRole
	}

	clone := &Agent{
		Role:                 role,
		Goal:                 a.Goal,
		Backstory:            a.Backstory,
		Verbose:              a.Verbose,
		LLM:                 a.LLM,
		Tools:                make([]Tool, len(a.Tools)),
		MaxIterations:        a.MaxIterations,
		MaxRetryLimit:        a.MaxRetryLimit,
		MaxRPM:               a.MaxRPM,
		RespectContextWindow: a.RespectContextWindow,
		Memory:               a.Memory,
		EntityMemory:         a.EntityMemory,
		SelfHealing:          a.SelfHealing,
		Cache:                a.Cache,
		UsageMetrics:         make(map[string]int),
		AllowDelegation:      a.AllowDelegation,
		SelfCritique:         a.SelfCritique,
		Sandbox:              a.Sandbox,
	}

	copy(clone.Tools, a.Tools)

	// Deep copy guardrails
	if len(a.Guardrails) > 0 {
		clone.Guardrails = make([]guardrails.Guardrail, len(a.Guardrails))
		copy(clone.Guardrails, a.Guardrails)
	}

	// Deep copy knowledge bases
	if len(a.KnowledgeBases) > 0 {
		clone.KnowledgeBases = make([]string, len(a.KnowledgeBases))
		copy(clone.KnowledgeBases, a.KnowledgeBases)
	}

	// Deep copy few-shot examples
	if len(a.FewShotExamples) > 0 {
		clone.FewShotExamples = make([]string, len(a.FewShotExamples))
		copy(clone.FewShotExamples, a.FewShotExamples)
	}

	return clone
}

// ---------------------------------------------------------------------------
// Agent Template — Serializable Agent Definition
// ---------------------------------------------------------------------------

// AgentTemplate is a serializable definition for creating agents.
type AgentTemplate struct {
	Role        string            `json:"role" yaml:"role"`
	Goal        string            `json:"goal" yaml:"goal"`
	Backstory   string            `json:"backstory" yaml:"backstory"`
	Tools       []string          `json:"tools,omitempty" yaml:"tools,omitempty"`     // Tool names
	Verbose     bool              `json:"verbose,omitempty" yaml:"verbose,omitempty"`
	MaxIter     int               `json:"max_iterations,omitempty" yaml:"max_iterations,omitempty"`
	SelfHealing bool              `json:"self_healing,omitempty" yaml:"self_healing,omitempty"`
	SelfCritique bool             `json:"self_critique,omitempty" yaml:"self_critique,omitempty"`
	Delegation  bool              `json:"allow_delegation,omitempty" yaml:"allow_delegation,omitempty"`
	Knowledge   []string          `json:"knowledge_bases,omitempty" yaml:"knowledge_bases,omitempty"`
	FewShot     []string          `json:"few_shot_examples,omitempty" yaml:"few_shot_examples,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// ToTemplate exports an agent's configuration as a reusable template.
func (a *Agent) ToTemplate() AgentTemplate {
	toolNames := make([]string, len(a.Tools))
	for i, t := range a.Tools {
		toolNames[i] = t.Name()
	}

	return AgentTemplate{
		Role:         a.Role,
		Goal:         a.Goal,
		Backstory:    a.Backstory,
		Tools:        toolNames,
		Verbose:      a.Verbose,
		MaxIter:      a.MaxIterations,
		SelfHealing:  a.SelfHealing,
		SelfCritique: a.SelfCritique,
		Delegation:   a.AllowDelegation,
		Knowledge:    a.KnowledgeBases,
		FewShot:      a.FewShotExamples,
	}
}

// SaveTemplate writes the agent template to a JSON file.
func (a *Agent) SaveTemplate(path string) error {
	tmpl := a.ToTemplate()
	data, err := json.MarshalIndent(tmpl, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// LoadTemplate reads an agent template from a JSON file.
func LoadTemplate(path string) (*AgentTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	var tmpl AgentTemplate
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &tmpl, nil
}

// ---------------------------------------------------------------------------
// Agent Pool — Pre-Created Agent Collections
// ---------------------------------------------------------------------------

// AgentPool manages a collection of reusable agent instances.
type AgentPool struct {
	Templates map[string]AgentTemplate
	agents    map[string]*Agent
}

// NewAgentPool creates an empty pool.
func NewAgentPool() *AgentPool {
	return &AgentPool{
		Templates: make(map[string]AgentTemplate),
		agents:    make(map[string]*Agent),
	}
}

// AddTemplate registers a template in the pool.
func (p *AgentPool) AddTemplate(name string, tmpl AgentTemplate) {
	p.Templates[name] = tmpl
}

// Register stores a live agent in the pool for reuse.
func (p *AgentPool) Register(name string, agent *Agent) {
	p.agents[name] = agent
}

// Get retrieves a registered agent by name.
func (p *AgentPool) Get(name string) (*Agent, bool) {
	a, ok := p.agents[name]
	return a, ok
}

// CloneFrom creates a new agent by cloning an existing pool member.
func (p *AgentPool) CloneFrom(sourceName, newRole string) (*Agent, error) {
	source, ok := p.agents[sourceName]
	if !ok {
		return nil, fmt.Errorf("agent not found in pool: %s", sourceName)
	}
	return source.Clone(newRole), nil
}
