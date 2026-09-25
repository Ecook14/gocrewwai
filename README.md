# Gocrewwai ⚓🏆🚀

High-performance, strictly-typed agentic orchestration for the Go ecosystem. Inspired by **CrewAI, LangChain, and LangGraph**, Gocrewwai is built for developers who demand speed, reliability, and production-ready precision.

---

> [!IMPORTANT]
> **Status: v0.9.0 (Alpha → Beta).** The framework is feature-complete against the public roadmap; v1.0.0 will ship once the next CHANGELOG entry is published. Native A2A protocols, durable flow persistence, and recursive self-correction are in production use today.

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

### 🌐 5. Model Context Protocol (MCP) & Discovery
Seamlessly connect your agents to external tools and knowledge sources via the standardized MCP protocol. Native support for local and remote MCP servers with auto-discovery.

### 🤖 6. Agent-to-Agent (A2A) Protocols
Enable true decentralized swarm intelligence. Agents can discover each other on the network, negotiate tasks, and collaborate autonomously using standardized communication protocols.

### 🛡️ 7. Security-First Tooling
- **Docker sandboxing**: Code execution runs in hardened containers (--network none, --cap-drop ALL, --read-only, --user 1000:1000, --pids-limit).
- **SSRF protection**: URL-fetching tools validate schemes and block private/special IPs before connecting.
- **Shell command whitelist**: Exact basename matching — empty AllowedCommands list denies all commands by default.
- **Human review gates**: Dangerous tools (file writes, HTTP requests, database ops, code execution) require explicit approval before execution.
- **API auth**: Constant-time token comparison (`crypto/subtle`) + 1MB request body limit to prevent memory exhaustion.
- **HTTP timeouts**: All outbound HTTP clients have per-package timeout configs (dial, TLS, response header) — no bare `http.Get` or `http.DefaultClient`.

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
	"log"

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
	// summary := gocrew.GetOutput[SummaryResult](result)
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
├── cmd/
│   ├── gocrew/        # CLI entrypoint (gocrew create/run/kickoff)
│   └── server/        # API server with mesh + dashboard support
├── gocrew/            # Ergonomic SDK facade (gocrew.NewAgent, gocrew.NewCrew, etc.)
|   ├── pkg/
│   │   ├── agents/        # Agent definitions and lifecycle
│   │   ├── crew/          # Crew orchestration + checkpoint stores (SQLite/Redis)
│   │   ├── llm/           # LLM clients (OpenAI, Anthropic, Gemini, Groq, OpenRouter, Ollama, Failover) and caching
│   │   ├── memory/        # Unified memory with vector search (12 store types: SQLite, Redis, Chroma, Pinecone, Qdrant, Weaviate, InMemCosine, Conversation, Entity, ShortTerm, LongTerm, Unified)
│   │   ├── tools/         # 57 built-in tools (search, browser, DB, code interp, SaaS integrations)
│   │   ├── api/           # Gin REST API + gRPC mesh server
│   │   ├── flow/          # Multi-crew orchestration flows
│   │   ├── knowledge/     # RAG knowledge sources (PDFs, URLs, text, directories, CSV, JSON)
│   │   ├── events/        # Event bus for cross-component communication
│   │   ├── guardrails/    # Input/output validation guardrails (6 types: MaxToken, ContentFilter, Schema, PIIRedaction, Toxicity, HumanReview)
│   │   ├── protocols/     # MCP, A2A, WebMCP protocol implementations
│   │   ├── sandbox/       # Docker and WASM code sandboxing
│   │   ├── server/        # Production HTTP server with health/metrics/graceful shutdown
│   │   ├── telemetry/     # Native OpenTelemetry tracing and metrics
│   │   ├── config/        # YAML/JSON configuration loading
│   │   ├── core/          # Core types, interfaces, and primitives
│   │   ├── errors/        # Structured error types
│   │   ├── files/         # File abstraction with provider backends
│   │   ├── i18n/          # Internationalization and localization
│   │   ├── training/      # Human-in-the-loop training data and advice
│   │   ├── utils/         # Shared utilities and helpers
│   │   └── ... (31 core packages total)
```

---

## 🤝 Community & Support

- **Gocrew** - High-performance agentic AI, built for Go developers.
- Follow the development on [GitHub](https://github.com/Ecook14/gocrewwai).
- Join the mission to build the most scalable AI framework in the community! 🚀⚓🛡️🏆🏁
