# Gocrewwai ⚓🏆🚀

High-performance, strictly-typed agentic orchestration for the Go ecosystem. Inspired by **CrewAI, LangChain, and LangGraph**, Gocrewwai is built for developers who demand speed, reliability, and production-ready precision.

---

> [!IMPORTANT]
> **Status: v0.9.0 (Alpha → Beta).** The framework is feature-complete against the public roadmap; v1.0.0 will ship once the next CHANGELOG entry is published. Native A2A protocols, durable flow persistence, and recursive self-correction are in production use today.

---

## 🌟 Why Gocrewwai?

While many AI tools remain in the Python ecosystem, Go gives Gocrewwai real production advantages:

1. **⚡ Concurrency**: Native goroutines enable true parallel agent execution without a GIL.
2. **🛡️ Reliability**: Every LLM response is unmarshaled into strictly-typed Go structs — no runtime `KeyError` surprises.
3. **🧠 Memory & State**: Vector-indexed memory (11 backends) with durable flow persistence (checkpoints / time-travel).
4. **Single-Binary Deployment**: Compile the full orchestrator into a zero-dependency binary.

---

## ⚡ Core Features

### 🛡️ 1. Durable Flows & Checkpoints (LangGraph Parity)
Pause, resume, and time-travel through long-running workflows. State is checkpointed to SQLite or Redis after every node.

### 👤 2. Human-in-the-Loop (HITL)
Manual interrupts and approvals via CLI or the real-time Dashboard.

### 🛡️ 3. Recursive Self-Correction
Agents reflect on their own work via internal loops or peer-review crews.

### 📊 4. Native Observability (OpenTelemetry)
Vendor-neutral tracing with built-in OTEL integration.

### 🌐 5. Model Context Protocol (MCP) & Discovery
Standard MCP protocol with local/remote server support and peer health-checks.

### 🤖 6. Agent-to-Agent (A2A) Protocols
Decentralized swarm intelligence with peer health-checks.

### 🛡️ 7. Security-First Tooling
- **Docker sandboxing**: Code execution runs in hardened containers (--network none, --cap-drop ALL, --read-only, --user 1000:1000, --pids-limit).
- **SSRF protection**: URL-fetching tools validate schemes and block private/special IPs before connecting.
- **Shell command whitelist**: Exact basename matching — empty AllowedCommands list denies all commands by default.
- **Human review gates**: Dangerous tools (file writes, HTTP requests, database ops, code execution) require explicit approval before execution.
- **API auth**: Constant-time token comparison (`crypto/subtle`) + 1MB request body limit to prevent memory exhaustion.
- **HTTP timeouts**: All outbound HTTP clients have per-package timeout configs (dial, TLS, response header) — no bare `http.Get` or `http.DefaultClient`.

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

# Show version
gocrew version
```

### 🖥️ Start the Dashboard Server

> **Note**: The `--ui` flag opens the interactive dashboard. The dashboard communicates with the server via WebSocket.

```bash
gocrew kickoff --ui
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
- **[📊 Observability](docs/features/telemetry.md)**: Native OTEL tracing and performance metrics.
- **[🧰 MCP Hub](docs/features/mcp.md)**: Model Context Protocol integration.
- **[🌐 A2A Protocols](docs/features/agent_delegation.md)**: Agent-to-agent communication.
- **[🛡️ Security Architecture](docs/features/production.md)**: Sandboxing, TLS, access control, audit logging.

---

## 🏗️ Architecture

```
gocrewwai/
├── api/proto/           # Protocol buffers
├── benchmarks/          # Benchmarks
├── cmd/
│   ├── gocrew/          # CLI entrypoint
│   └── server/          # HTTP API & dashboard server
├── docs/                # Guides + feature docs
├── examples/            # Demo crews
├── gocrew/              # SDK facade (gocrew.NewAgent, NewCrew, ...)
├── internal/            # Private impl (cli, delegation, guardrails)
├── pkg/
│   ├── agents/          # Agent definitions & reasoning loops
│   ├── api/mesh/        # Gin REST + gRPC mesh
│   ├── compat/          # Compatibility adapters
│   ├── config/          # YAML/JSON config
│   ├── core/            # Interfaces, primitives
│   ├── crew/            # Orchestration engines
│   ├── delegation/      # A2A internal delegation
│   ├── dashboard/       # Dashboard APIs
│   ├── errors/          # Structured errors
│   ├── events/          # GlobalBus event system
│   ├── files/           # File abstraction
│   ├── flow / flows/    # Workflow persistence
│   ├── guardrails/      # Validation hooks + HITL
│   ├── i18n/            # Localization
│   ├── knowledge/       # RAG ingestion
│   ├── llm/             # Provider clients + caching
│   ├── memory/          # Vector + entity memory
│   ├── protocols/       # MCP, A2A, WebMCP
│   ├── sandbox/         # Docker + WASM sandboxing
│   ├── server/          # HTTP server, health, metrics
│   ├── tasks/           # Task lifecycle
│   ├── telemetry/       # OTEL tracing
│   ├── testing/         # Test harnesses
│   ├── tools/           # Built-in tool ecosystem
│   ├── training/        # HITL training data
│   └── utils/           # Shared helpers
├── web/                 # React/Vite Dashboard
└── web-ui/              # Static embeddable UI
```

---

## 🤝 Community & Support

- **Gocrew** - High-performance agentic AI, built for Go developers.
- Follow the development on [GitHub](https://github.com/Ecook14/gocrewwai).
- Join the mission to build the most scalable AI framework in the community! 🚀⚓🛡️🏆🏁
