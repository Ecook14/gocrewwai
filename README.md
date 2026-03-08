# Crew-GO (Elite Tier) 🚀

> **"The Industrial-Grade Orchestration Engine for Autonomous Go Agents"**

**Crew-GO** is the definitive Go-native framework for building, managing, and scaling autonomous AI agent collectives. Engineered for high-throughput, technical precision, and absolute reliability, it brings the "CrewAI" philosophy to the Go ecosystem with a focus on **Type-Safety**, **Massive Parallelism**, and **Elite Multi-modal Sandboxing**.

---

## 🌟 Why Choose Crew-GO?

While Python frameworks are great for prototyping, **production AI systems require guarantees**. Crew-GO provides:
1. **True Concurrency**: Python's GIL limits true parallelism. Crew-GO uses Go's native goroutines to blast through hundreds of parallel agent tasks simultaneously, now with **Smart Concurrency Gating**.
2. **Type Safety**: No more runtime `KeyError` crashes. Every Agent, Task, and Tool in Crew-GO is strictly typed, and now supports **Strongly-Typed Output Extraction via Go Generics**.
3. **Structured Outputs**: Native extraction of LLM JSON responses directly into Go structs.
4. **Resilient Networking**: Built-in **Exponential Backoff & Automatic Retries** for HTTP 429 (Rate Limit) and 5xx errors.
5. **Elite Tier DX**: Premium **Fluent Builder APIs** and **Event-Driven Flows** (Listen/Emit) for elegant choreography.
6. **Extreme Performance**: Compiled binaries that deploy anywhere with a tiny memory footprint.

---

## 💎 Elite Feature Set

### 🛡️ 1. Multi-modal Execution Sandboxing
Execute code-heavy or untrusted tasks with maximum security and absolute isolation. Crew-GO natively supports multiple sandbox environments:
- **Docker Integration**: Run agent-generated code inside ephemeral, resource-limited Docker containers (CPU/Memory caps).
- **WASM (wazero)**: Lightning-fast, zero-dependency local sandboxing for untrusted snippets directly within your Go binary.
- **Local Isolation**: Sub-process isolation for trusted local environments.

### 🖥️ 2. Real-time Telemetry Dashboard (Glassmorphism UI)
Experience unmatched observability with our **Premium Glassmorphic UI**.
- **Live Execution Trace**: Watch every reasoning step, tool invocation, and task completion in real-time.
- **Agent Status Visualization**: Real-time cards showing current agent activity (Idle, Thinking, Working) and system metrics.
- **WebSocket Streaming**: Powered by a high-performance Go backend event bus for zero-latency updates.
- **CLI Integration**: Simply append `--ui` to your `crewai kickoff` command to instantly launch the dashboard.

### 🧠 3. Advanced Persistence (Short-term, Long-term, & Entity Memory)
Beyond standard RAG, Crew-GO implements a sophisticated multi-layer memory system:
- **Short-term Memory**: Contextual thread of the current execution.
- **Long-term Memory**: Historical vector-backed database of past interactions.
- **Entity Memory (Elite)**: Persistent tracking of specific metadata (facts, keys, preferences) across disparate tasks.

### 🧰 3. Industrial Tool Arsenal (24+ Native Tools)
Equip your agents with a massive suite of capabilities out-of-the-box:
- **Web Automation**: `BrowserTool` (Full SPAs via Chromedp), `ScrapeWebsite`, `SearchWeb`.
- **Social & Comm**: `GitHubTool` (Issues, Repos, Prs), `SlackTool`.
- **Search & Research**: `Google Serper`, `Exa.ai`, `Arxiv`, `Wikipedia`, `WolframAlpha`.
- **System & Logic**: `CodeInterpreterTool`, `WASMSandboxTool`, `PostgresTool`, `FileOps`.

### 🛡️ 4. Professional-Grade Guardrails
Ensure your agents adhere to enterprise safety standards:
- **PII Redactor**: Automatically mask emails, IPs, credit cards, and sensitive strings before output is finalized.
- **LLM Review Layer (HITL)**: Configure a "Critic" agent to review and formally approve another agent's work.
- **Toxicity Filters**: Real-time content moderation for safety-critical applications.

---

## 🏗️ Architecture & Orchestration

Crew-GO is built on a **Durable Graph State Machine**, enabling unimaginably complex, stateful orchestration patterns.

- **Sequential**: Standard linear pipeline for procedural generation.
- **Parallel & Hierarchical**: High-throughput concurrent execution managed by an autonomous `ManagerAgent`.
- **Consensual**: Multiple agents attack the same problem, forcing a consensus synthesis.
- **Cyclic Graphs (Elite)**: Supports endless loops, dynamic branching, and stateful backtracking based on real-time evaluation.
- **Reflective**: Mandatory manager review stages forcing agents to revise their work until it meets a quality threshold.

---

## 🚀 Quick Start Instructions

### 1. Prerequisites
- **Go**: `1.22` or higher.
- **Docker**: (Optional) Required if you plan to use `docker` sandboxes.
- **API Keys**: An OpenAI API key (or compatible LLM provider).

### 2. Install the Core Library & CLI
```bash
go get github.com/Ecook14/crewai-go
go install github.com/Ecook14/crewai-go/cmd/crewai@latest
```

### 3. Scaffold a New Project
```bash
~/go/bin/crewai create my-ops-crew
cd my-ops-crew
```

### 4. Run your Crew with the Live UI
```bash
export OPENAI_API_KEY=your_key
~/go/bin/crewai kickoff --ui
```
*Navigate to `http://localhost:8080/web-ui` to watch your agents think!*

---

## 📖 Comprehensive Documentation Hub

To truly master Crew-GO, dive into our extensive guides:

1. 🚀 **[Quickstart Guide](docs/quickstart.md)**: From zero to your first running Go agent in 5 minutes.
2. 📖 **[Detailed Usage Guide](USAGE.md)**: Exhaustive documentation on Tools, Memory, Guardrails, and YAML configs.
3. 🔥 **[Advanced Orchestration](docs/advanced_usage.md)**: Master Cyclic Graphs, Hierarchical Delegation, and Reactive Flows.
4. 🏗️ **[Internal Architecture](docs/architecture.md)**: Read how the ReAct Loop, Global Event Bus, and Graph Engine work under the hood.

---

## 📊 Tier Comparison Matrix

| Capability | Standard Crew Frameworks | Crew-GO (Elite Tier) |
| :--- | :--- | :--- |
| **Concurrency Model** | Blocking / Asyncio Loops | **Native Goroutines (Massive Async)** |
| **Execution Sandboxes**| Local Process | **Docker, WASM, Local** |
| **Observability** | Standard Output logs | **Live WebSocket Glassmorphic Dashboard** |
| **Orchestration Logic**| Sequential Pipelines | **Cyclic Graphs & State Machines** |
| **Safety & Validation**| Basic Prompting | **PII Redaction, LLM Review, Strict Schemas** |

---

## 📄 License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**Crew-GO** - Built for Go developers who demand **Elite Performance**, **Total Control**, and **Production Reliability**.
