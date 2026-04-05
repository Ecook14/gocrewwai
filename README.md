# Gocrewwai ⚓🏆🚀 (v1.0.0 Stable)

High-performance, strictly-typed agentic orchestration for the Go ecosystem. Inspired by **CrewAI, LangChain, and LangGraph**, Gocrewwai is built for developers who demand speed, reliability, and production-ready precision.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Production Hardened).** Gocrewwai has evolved into a stable, high-fidelity framework for heterogeneous agent orchestration. With native A2A protocols, durable flow persistence, and recursive self-correction, it is ready for critical digital missions.

---

## 🌟 Why Gocrewwai?

While many AI tools remain in the Python ecosystem, we chose **Go** for its inherent production superpowers:

1. **⚡ Massive Concurrency**: Go's native goroutines allow hundreds of agents to work, fetch data, and reason in true parallel without the bottlenecks of a Global Interpreter Lock (GIL).
2. **🛡️ Rock-Solid Reliability**: Eliminate random `KeyError` crashes. Every LLM response is strictly unmarshaled into your Go structs with type-safe guarantees.
3. **🧠 Elite Memory & State**: Built-in, vector-indexed memory (SQLite, Redis, Chroma) and durable flow persistence (Checkpoints/Time-Travel).
4. **Single-Binary Deployment**: Compile your entire orchestrator into a tiny, zero-dependency binary. Drop it in a container or on an edge device and it just works.

---

## 💎 Elite Features

### 🛡️ 1. Durable Flows & Checkpoints (LangGraph Parity)
Gocrewwai provides robust persistence, allowing you to pause, resume, and "time-travel" through long-running agentic workflows. State is automatically checkpointed to SQLite or Redis after every node execution.

### 👤 2. Human-in-the-Loop (HITL)
Native support for manual interrupts and approvals. Pause an agent's execution for review and approval through the CLI or the real-time Glassmorphic Dashboard.

### 🛡️ 3. Recursive Self-Correction (CrewAI Parity)
Agents can reflect on their own work using internal reflection loops or peer-review "Reflective Crews," ensuring 100% adherence to task requirements.

### 📊 4. Native Observability (OpenTelemetry)
Standardized, vendor-neutral tracing with built-in OTEL integration. Track every agent thought, tool execution, and token cost with high fidelity.

---

## 🚀 Quickstart (Elite Style)

Initialize your project and install the Gocrewwai SDK:

```bash
go mod init my-agent-app
go get github.com/Ecook14/gocrewwai/gocrew
```

### Build Your First Crew

```go
package main

import (
	"context"
	"github.com/Ecook14/gocrewwai/gocrew"
)

func main() {
	// 1. Setup the Model
	llm := gocrew.NewOpenAI("your-api-key", "gpt-4o")

	// 2. Define an Agent
	researcher := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Researcher",
		Goal:      "Find the latest trends in Go 1.25",
		Backstory: "Expert in performance optimization.",
		LLM:       llm,
	})

	// 3. Define a Task
	task := gocrew.NewTask(gocrew.TaskConfig{
		Description: "Analyze Go 1.25 Type Aliases and return a summary.",
		Agent:       researcher,
	})

	// 4. Assemble and Kickoff!
	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{researcher},
		Tasks:   []*gocrew.Task{task},
		Verbose: true,
	})

	myCrew.Kickoff(context.Background())
}
```

---

## 📚 Documentation Portal

Dive deep into the Gocrewwai ecosystem with our world-class documentation guides:

- **[⚓ Core Concepts](docs/CORE_CONCEPTS.md)**: The "Four Pillars" of Gocrewwai.
- **[🚀 Getting Started](docs/GETTING_STARTED.md)**: Full installation and quickstart guide.
- **[🔄 Migration Guide](docs/MIGRATION.md)**: Transitioning from CrewAI, LangChain, or LangGraph.
- **[🧩 Agents, Tasks & Crews](docs/index.md#core-components)**: Detailed orchestration guides.
- **[💾 Persistence & HITL](docs/PERSISTENCE.md)**: Durable execution and human oversight.
- **[🛡️ Self-Correction](docs/SELF_CORRECTION.md)**: Reflective reasoning and reliability.
- **[📊 Observability](docs/OBSERVABILITY.md)**: Native OTEL tracing and performance metrics.

---

## 🤝 Community & Support

- **Gocrew** - High-performance agentic AI, built for Go developers.
- Follow the development on [GitHub](https://github.com/Ecook14/gocrewwai).
- Join the mission to build the most scalable AI framework in the community! 🚀⚓🛡️🏆🏁
