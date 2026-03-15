# Welcome to Gocrew! 🚀 (Early Alpha)

Hey there! I am absolutely thrilled to introduce you to **Gocrew**, an open-source project designed to bring high-performance, strictly-typed agentic orchestration to the Go ecosystem. Inspired by the incredible work of **CrewAI, LangChain, and LangGraph**, Gocrew is built for developers who demand speed, reliability, and precision.

Think of Gocrew as a framework for assembling a digital "crew." You assign each member a specific persona—like a Lead Researcher, a Senior Data Analyst, or an Expert Coder. You hand them a goal, provide them with tools, and they figure out how to communicate and collaborate to achieve that goal!

> [!IMPORTANT]
> **Status: v0.9 (Autonomous Interoperability).** Gocrew has evolved into a production-hardened framework for heterogeneous agent orchestration. With native A2A protocols, MCP Hub capabilities, and Flow 2.0 persistence, it is ready for complex, long-running digital missions.

---

## 🌟 Why Gocrew?

While many AI tools stay in the Python ecosystem, we chose **Go** for its inherent superpowers:

1. **Massive Concurrency**: Go's native goroutines allow hundreds of agents to work, fetch data, and reason in true parallel without the bottlenecks of a Global Interpreter Lock (GIL).
2. **Rock-Solid Reliability**: Eliminate random `KeyError` crashes. Every LLM response is strictly unmarshaled into your Go structs with type-safe guarantees.
3. **Resilient HTTP Core**: Built-in exponential backoff and retry logic for all major LLM providers (OpenAI, Anthropic, Gemini, Groq, OpenRouter).
4. **Single-Binary Deployment**: Compile your entire orchestrator, including the Glassmorphic Web UI, into a tiny, zero-dependency binary. Drop it in a container or on an edge device and it just works.

---

## 💎 Elite Features

We've built a suite of production-grade features to give your agents true autonomy:

### 🛡️ 1. Safe Code Execution (Sandboxing)
Grant agents the power to write and execute Python, Go, or Shell scripts securely.
- **Docker**: Isolate execution in ephemeral, resource-capped containers.
- **E2B Integration**: Offload execution to remote, secure cloud sandboxes.
- **WASM**: Use WebAssembly (`wazero`) for lightning-fast, zero-dependency isolation.

### 🖥️ 2. Glassmorphic Live Dashboard
Watch the "black box" become transparent. Adding `--ui` to your execution launches a stunning real-time control plane:
- **Live Stream**: Watch agent thoughts, tool logs, and task handoffs via WebSockets.
- **Human-in-the-Loop (HITL)**: Review and approve sensitive actions (file writes, shell commands) directly from your browser.
- **Creator Studio**: Dynamically build agents and tasks from the UI to experiment without recompiling.

### 🧠 3. Unified Memory System
Our agents remember everything that matters, powered by a sophisticated scoring engine (Recency + Relevance + Importance):
- **Short-term Memory**: For step-by-step context.
- **Long-term Memory**: Native Vector store support (SQLite, Redis, Pinecone, Qdrant, Weaviate).
- **Entity Memory**: Persistent tracking of facts about people, companies, and concepts across sessions.

### 🧰 4. Massive Tool Arsenal
Empower your agents with 24+ native tools, including:
- **Full Browser Automation**: Headless `Chromedp` for navigating complex React/Vue SPAs.
- **Ingestion Engine**: Native parsing for PDF, DOCX, CSV, and TXT files.
- **MCP Hub**: First-class support for the Model Context Protocol (MCP). Connect to remote tools via `HTTP`, `Stdio` (local commands), or `SSE` with native tool filtering (Allow/Block lists).

---

## 🚀 Quickstart

### 1. Install the SDK
```bash
go get github.com/Ecook14/gocrewwai
```

### 2. Build Your First Crew
Gocrew uses an ergonomic, builder-based API for maximum clarity:

```go
import (
    "context"
    "github.com/Ecook14/gocrewwai/gocrew"
)

func main() {
    // 1. Setup the Model
    llm := gocrew.NewOpenAI("your-api-key", "gpt-4o")

    // 2. Build the Agent
    researcher := gocrew.NewAgentBuilder().
        Role("Researcher").
        Goal("Find the latest trends in Go 1.24").
        LLM(llm).
        Build()

    // 3. Define the Mission
    task := gocrew.NewTaskBuilder().
        Description("Analyze Go 1.24 Type Aliases and return a summary.").
        Agent(researcher).
        Build()

    // 4. Kickoff!
    myCrew := gocrew.NewCrewBuilder().
        Agents(researcher).
        Tasks(task).
        Verbose(true).
        Build()

    myCrew.Kickoff(context.Background())
}
```

### 3. Rapid Scaffolding (CLI)
Install the global `gocrew` CLI to scaffold projects and launch the Dashboard from anywhere:
```bash
go install github.com/Ecook14/gocrewwai/cmd/gocrew@latest
gocrew create my-awesome-project
```

---

## 📚 Documentation

Dive deep into every corner of the Gocrew ecosystem with our comprehensive feature guides:

### Features
- [**Agents**](docs/features/agents.md): Personas, roles, and backstories.
- [**Tasks**](docs/features/tasks.md): Assignments, context-piping, and output schemas.
- [**Crews**](docs/features/crews.md): Orchestrating teams and process types.
- [**Tools**](docs/features/tools.md): Expanding capabilities with Go functions.
- [**LLMs**](docs/features/llms.md): Provider configurations and advanced options.
- [**Processes**](docs/features/processes.md): Sequential, Hierarchical, and State Machine logic.

- [**Collaboration**](docs/features/collaboration.md): Delegation and coworker communication.
- [**Memory**](docs/features/memory.md): Short-term, Long-term, and Entity storage.
- [**Knowledge**](docs/features/knowledge.md): RAG, vector embeddings, and external data.
- [**Planning**](docs/features/planning.md): Intelligent task decomposition.
- [**Flows**](docs/features/flows.md): Multi-crew orchestration and state machines.
- [**Files**](docs/features/files.md): Multi-modal file handling (PDF, Images, etc.).

- [**Reasoning**](docs/features/reasoning.md): Reflective loops and self-correction.
- [**Training**](docs/features/training.md): Human-in-the-loop advice persistence.
- [**Testing**](docs/features/testing.md): Automated multi-run evaluation frameworks.
- [**Events**](docs/features/events.md): Real-time observability and telemetry.
- [**CLI**](docs/features/cli.md): Management, scaffolding, and replays.
- [**Production**](docs/features/production.md): Sandboxing, scaling, and enterprise patterns.

---

## 🤝 Community & Contribution

I am building the most scalable AI framework for the Go community, and I need your help! Since I am in **Early Alpha**, your contributions will directly shape the future of the framework.

### 🔥 High-Priority Roadmap
I am specifically looking for collaborators to help with:
- **🧬 Gocrew Flows 2.0**: Strictly-typed state machines with native checkpointing and persistence.
- **🛡️ Distributed Execution**: Agent-as-a-Service (AaaS) for cross-node and cross-cloud crew collaboration.
- **🧪 Multi-Agent Testing**: Building frameworks to evaluate agent reasoning at scale.
- **Dashboard Upgrades**: Real-time control plane enhancements and visualizations.

### How to Get Involved
1. **Join the Movement**: Check out our [Contributing Guide](CONTRIBUTING.md).
2. **Roadmap**: Check out [ROADMAP.md](ROADMAP.md) for current priorities.
3. **Share Your Feedback**: Found a bug or have a feature idea? Open an issue!

---
**Gocrew** - High-performance agentic AI, built for Go developers.
