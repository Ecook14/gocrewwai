package crew

import (
	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
)

// CrewBuilder provides a fluent API for constructing Crews.
type CrewBuilder struct {
	crew *Crew
}

func NewCrewBuilder() *CrewBuilder {
	return &CrewBuilder{
		crew: &Crew{
			Process:      Sequential,
			UsageMetrics: make(map[string]int),
		},
	}
}

func (b *CrewBuilder) Agents(a ...*agents.Agent) *CrewBuilder {
	b.crew.Agents = append(b.crew.Agents, a...)
	return b
}

func (b *CrewBuilder) Tasks(t ...*tasks.Task) *CrewBuilder {
	b.crew.Tasks = append(b.crew.Tasks, t...)
	return b
}

func (b *CrewBuilder) Process(p ProcessType) *CrewBuilder {
	b.crew.Process = p
	return b
}

func (b *CrewBuilder) Manager(m *agents.Agent) *CrewBuilder {
	b.crew.ManagerAgent = m
	return b
}

func (b *CrewBuilder) Verbose(v bool) *CrewBuilder {
	b.crew.Verbose = v
	return b
}

func (b *CrewBuilder) StateFile(path string) *CrewBuilder {
	b.crew.StateFile = path
	return b
}

func (b *CrewBuilder) Build() *Crew {
	return b.crew
}
