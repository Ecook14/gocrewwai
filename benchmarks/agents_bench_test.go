package benchmarks

import (
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/agents"
)

func BenchmarkAgentCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = agents.NewAgent(agents.AgentConfig{})
	}
}
