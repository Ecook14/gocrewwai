# Gocrewwai

Strictly-typed agentic orchestration for Go. Inspired by CrewAI, LangChain, and LangGraph.

> **Status: v1.0.0-beta.3.** The framework is feature-complete against the public roadmap; v1.0.0 will ship once the next CHANGELOG entry is published.

---

## Why Go?

While most AI orchestration tooling lives in the Python ecosystem, Go offers real production advantages:

1. **Concurrency**: Native goroutines for parallel agent execution — no GIL.
2. **Type safety**: Every LLM response is unmarshaled into strictly-typed Go structs. No runtime `KeyError` surprises.
3. **Memory & State**: Vector-indexed memory (12 backends) with durable flow checkpoints.
4. **Single binary**: Compile the full orchestrator into a zero-dependency binary.

---

## Core Features

1. **Durable Flows & Checkpoints** — Pause, resume, and time-travel through long-running workflows. State checkpointed to SQLite or Redis after every node.
2. **Human-in-the-Loop (HITL)** — Manual interrupts and approvals via CLI or Dashboard.
3. **Recursive Self-Correction** — Agents reflect on their work via internal loops or peer-review crews.
4. **OpenTelemetry Tracing** — Vendor-neutral observability.
5. **MCP Protocol** — Standard MCP with local/remote servers and peer health-checks.
6. **A2A Protocols** — Agent-to-agent swarm communication with health-checks.
7. **Security-First Tooling** — Docker sandboxing (`--network none`, `--cap-drop ALL`, `--read-only`, `--user 1000:1000`, `--pids-limit`), SSRF protection, shell command whitelist, human review gates for dangerous tools, constant-time API auth, and per-client HTTP timeouts.

---

## 🚀 Quickstart

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
	"log"
	"os"

	"github.com/Ecook14/gocrewwai/gocrew"
)

func main() {
	// 1. Setup the Model
	llm := gocrew.NewOpenAI(os.Getenv("OPENAI_API_KEY"), "gpt-4o")

	// 2. Define an Agent
	researcher := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Researcher",
		Goal:      "Find the latest trends in Go 1.25",
		Backstory: "Expert in performance optimization.",
		LLM:       llm,
	})

	// 3. Define a Task with Strict JSON Output
	type SummaryResult struct {
		Trends []string `json:"trends"`
		Impact string   `json:"impact"`
	}

	task := gocrew.NewTask(gocrew.TaskConfig{
		Description: "Analyze Go 1.25 Type Aliases and return a summary.",
		Agent:       researcher,
		OutputJSON:  &SummaryResult{},
	})

	// 4. Assemble and Kickoff!
	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{researcher},
		Tasks:   []*gocrew.Task{task},
		Verbose: true,
	})

	// 5. Execute and extract the strongly-typed result natively
	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		log.Fatalf("Crew execution failed: %v", err)
	}

	// If you configured OutputJSON on the task, extract it:
	// summary := gocrew.GetOutput[SummaryResult](task)
}
```

### 🖥️ CLI Commands

```bash
# Scaffold a new project
gocrew create my-project

# Run a project
gocrew run

# Execute the demo crew
gocrew kickoff

# Train / smoke-test / replay against your project
gocrew train -n 5
gocrew test -n 3
gocrew replay -t task_1

# Reset a SQLite memory store / chat with your project
gocrew reset-memories --store sqlite --conn memory.db
gocrew chat

# Show version
gocrew version
```

### 🖥️ Start the Dashboard Server

> **Note**: The `--ui` flag opens the interactive dashboard. The dashboard communicates with the server via WebSocket.

```bash
gocrew kickoff --ui
```

---

## 📚 Documentation

- **[Core Concepts](docs/CORE_CONCEPTS.md)** — the "Four Pillars" of Gocrewwai.
- **[Getting Started](docs/GETTING_STARTED.md)** — installation and quickstart.
- **[Migration Guide](docs/MIGRATION.md)** — transitioning from CrewAI, LangChain, or LangGraph.
- **[Agents, Tasks & Crews](docs/index.md#core-components)** — orchestration guides.
- **[Persistence & HITL](docs/PERSISTENCE.md)** — durable execution and human oversight.
- **[Self-Correction](docs/SELF_CORRECTION.md)** — reflective reasoning and reliability.
- **[Observability](docs/features/telemetry.md)** — OTEL tracing and metrics.
- **[MCP Hub](docs/features/mcp.md)** — Model Context Protocol integration.
- **[A2A Protocols](docs/features/agent_delegation.md)** — agent-to-agent communication.
- **[Security Architecture](docs/features/production.md)** — sandboxing, TLS, access control.
- **[Architecture](ARCHITECTURE.md)** — module layout and dependency flow.

---

## 🤝 Community

- [GitHub](https://github.com/Ecook14/gocrewwai)
- Report issues, propose features, and discuss on the repository.
