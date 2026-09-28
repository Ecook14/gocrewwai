# Gocrewwai

Strictly-typed agentic orchestration for Go. Inspired by CrewAI, LangChain, and LangGraph.

> **Status: v1.0.0-beta.3.** Feature-complete against the public roadmap; v1.0.0 ships when the
> next CHANGELOG entry is published. Released binaries are on the
> [releases page](https://github.com/Ecook14/gocrewwai/releases).

---

## Two binaries — know which one you want

Gocrewwai ships **two separate programs** with similar names. They are not two modes of one
tool, and neither is a superset of the other. Picking the wrong one is the most common
source of confusion, so decide this first:

| | `gocrew-cli` | `gocrewwai-server` |
|---|---|---|
| **What it is** | A local developer tool | A long-running service |
| **Command** | `gocrew` | `gocrewwai-server` |
| **Lifetime** | Runs a task, then exits | Runs until you stop it (SIGINT/SIGTERM) |
| **Listens for others?** | No (only `--ui` on :8080) | Yes — REST `:8080`, gRPC mesh `:50051`, SSE |
| **Auth** | None. It runs as *you*, on your machine | `API_AUTH_TOKEN` / `API_AUTH_TOKENS` / `JWT_SECRET` |
| **Multi-tenant** | No | Yes — sessions are owner-scoped per token |
| **State** | Your local project and memory | Shared Redis / Dragonfly / Valkey |
| **Use it to** | Build and iterate on a crew | Deploy that crew to other people |

**The security model lives only on the server.** Owner-scoped sessions, `Idempotency-Key`
handling, JWT bearer validation, and signed outbound webhooks are all enforced by
`gocrewwai-server`. The CLI has no network surface, so it has nothing to authenticate —
it calls your LLM provider directly with your own key. **Never hand the CLI binary to an
end user**; it carries no access control.

If you are exposing Gocrewwai to anyone, you are running `gocrewwai-server`.

---

## Why Go?

While most AI orchestration tooling lives in the Python ecosystem, Go offers real production
advantages:

1. **Concurrency**: Native goroutines for parallel agent execution — no GIL.
2. **Type safety**: Every LLM response is unmarshaled into strictly-typed Go structs. No
   runtime `KeyError` surprises.
3. **Memory & State**: Vector-indexed memory (12 backends) with durable flow checkpoints.
4. **Single binary**: Compile the full orchestrator into a zero-dependency binary.

---

## Core Features

1. **Durable Flows & Checkpoints** — Pause, resume, and time-travel through long-running
   workflows. State checkpointed to SQLite or Redis after every node.
2. **Human-in-the-Loop (HITL)** — Manual interrupts and approvals via CLI or Dashboard.
3. **Recursive Self-Correction** — Agents reflect on their work via internal loops or
   peer-review crews.
4. **OpenTelemetry Tracing** — Vendor-neutral observability.
5. **MCP Protocol** — Standard MCP with local/remote servers and peer health-checks.
6. **A2A Protocols** — Agent-to-agent swarm communication with health-checks.
7. **Multi-tenant HTTP API** — Owner-scoped sessions, per-request idempotency, SSE
   streaming, and constant-time token comparison.
8. **Security-First Tooling** — Docker sandboxing (`--network none`, `--cap-drop ALL`,
   `--read-only`, `--user 1000:1000`, `--pids-limit`), SSRF protection with per-hop redirect
   revalidation, shell command whitelist, human review gates for dangerous tools, and
   per-client HTTP timeouts.

---

## 🚀 Quickstart

### Use it as a library

```bash
go mod init my-agent-app
go get github.com/Ecook14/gocrewwai/gocrew
```

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
	if _, err := myCrew.Kickoff(context.Background()); err != nil {
		log.Fatalf("Crew execution failed: %v", err)
	}

	// If you configured OutputJSON on the task, extract it:
	// summary := gocrew.GetOutput[SummaryResult](task)
}
```

### Install the CLI

Download from the [releases page](https://github.com/Ecook14/gocrewwai/releases) and
verify against the published `SHA256SUMS`. Binaries are published for linux, macOS, and
windows (amd64).

```bash
# linux example
curl -LO https://github.com/Ecook14/gocrewwai/releases/latest/download/gocrew-cli-linux
curl -LO https://github.com/Ecook14/gocrewwai/releases/latest/download/SHA256SUMS
sha256sum -c SHA256SUMS --ignore-missing
chmod +x gocrew-cli-linux && sudo mv gocrew-cli-linux /usr/local/bin/gocrew
gocrew version
```

> The Linux binaries are built with `CGO_ENABLED=1` (the embedded SQLite store needs it), so
> they are dynamically linked against glibc. They will not run on Alpine or older
> distributions without a `gcompat` layer. The macOS and Windows builds are static.

---

## 🖥️ The CLI (`gocrew`)

Every subcommand runs locally and exits when finished.

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

# Build release binaries of your own project
gocrew deploy --out ./dist

# Show version
gocrew version
```

> `gocrew kickoff --ui` additionally serves a local dashboard on `:8080` and pauses
> execution until you click START in the browser. It talks to the CLI process, not to a
> remote server.

Full reference: **[CLI reference](docs/features/cli.md)**.

---

## 🌐 The server (`gocrewwai-server`)

```bash
gocrewwai-server --api-port 8080 --mesh-port 50051 --web
```

It starts three things and stays up until stopped, shutting them all down cleanly on
SIGINT/SIGTERM:

- **REST API** (Gin) on `:8080` — kickoff, session reads, and SSE event streaming
- **gRPC Agent Mesh** on `:50051` — agent-to-agent communication
- **Visual Builder** — served from embedded files, only with `--web`

### Endpoints

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/v1/health` | Liveness and KV backend status |
| `POST` | `/api/v1/crews/kickoff` | Start a crew run (async, returns `202`) |
| `GET` | `/api/v1/sessions/:id` | Read session status and result |
| `GET` | `/api/v1/stream/:id` | SSE stream of session events |

### Kickoff a crew

```bash
curl -X POST http://localhost:8080/api/v1/crews/kickoff \
  -H "Authorization: Bearer $API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: run-$(uuidgen)" \
  -d '{
        "session_id": "run-0001",
        "agent_role": "Researcher",
        "agent_goal": "Summarize Go 1.25 type aliases",
        "agent_backstory": "Performance specialist",
        "agent_model": "gpt-4o",
        "task_description": "Write a summary.",
        "task_expected_output": "A short markdown summary."
      }'
```

Returns `202` immediately and executes in the background. Crew events stream from
`/api/v1/stream/:id`.

**`Idempotency-Key` matters.** It makes retries safe:

- While the first run is still in flight, a second kickoff with the same key gets **`409`**
  and is *not* executed — even if it names a different `session_id`. This is what prevents
  duplicate crews from double-clicks, cron retries, or racing clients.
- Once the run has finished, replaying the key returns **`200`** with the stored session id
  and status instead of starting new work.

Responses:

| Status | Meaning |
|---|---|
| `202` | Accepted, crew is running |
| `409` | Same key, first run still in progress |
| `200` | Same key, run already finished — returns the stored result |
| `401` | Missing or invalid bearer token |
| `503` | `agent_model` requested but `OPENAI_API_KEY` is not configured |

### Configuration

| Variable | Default | Purpose |
|---|---|---|
| `API_PORT` | `8080` | REST API port (`--api-port` overrides) |
| `MESH_PORT` | `50051` | gRPC mesh port (`--mesh-port` overrides) |
| `API_AUTH_TOKEN` | — | Single bearer token |
| `API_AUTH_TOKENS` | — | Comma-separated tokens; each becomes a separate tenant |
| `JWT_SECRET` | — | When set, also accepts HS256 JWT bearer tokens |
| `REDIS_BACKEND` | `redis` | `redis`, `dragonfly`, or `valkey` |
| `REDIS_ADDR` | `localhost:6379` | Comma-separated addresses |
| `REDIS_PASSWORD` | — | Redis auth |
| `REDIS_DB` | `0` | Database index |
| `OPENAI_API_KEY` | — | Required if a kickoff requests `agent_model` |
| `CREW_CONFIG_PATH` | `config.json` | Config file location |
| `ALLOW_INSECURE_DEV` | `0` | `1` permits running with the default token. Dev only. |

> **The server refuses to start on the insecure default token** unless you set
> `API_AUTH_TOKEN`/`API_AUTH_TOKENS` or explicitly opt in with `ALLOW_INSECURE_DEV=1`.
> Sessions are owner-scoped to the presenting token, so a caller cannot read or stream
> another tenant's session — cross-owner access returns `404`, not `403`.

Reference: **[Security & production guide](docs/features/production.md)** ·
**[Dragonfly deployment](docs/ops/dragonfly.md)**

---

## 📚 Documentation

**Concepts**
- [Core Concepts](docs/CORE_CONCEPTS.md) — the "Four Pillars" of Gocrewwai
- [Getting Started](docs/GETTING_STARTED.md) — installation and quickstart
- [Migration Guide](docs/MIGRATION.md) — transitioning from CrewAI, LangChain, or LangGraph
- [Architecture](ARCHITECTURE.md) — module layout and dependency flow

**Building crews**
- [Agents](docs/features/agents.md) · [Tasks](docs/features/tasks.md) · [Crews](docs/features/crews.md)
- [Processes](docs/features/processes.md) · [Reasoning](docs/features/reasoning.md)
- [Self-Correction](docs/SELF_CORRECTION.md) · [Planning](docs/features/planning.md)
- [Human Review](docs/features/collaboration.md)

**Infrastructure**
- [Persistence & HITL](docs/PERSISTENCE.md) · [Memory](docs/features/memory.md)
- [LLM Providers](docs/features/llms.md) · [Knowledge](docs/features/knowledge.md)
- [Tools](docs/features/tools.md) · [Files](docs/features/files.md)
- [MCP Hub](docs/features/mcp.md) · [A2A Protocols](docs/features/agent_delegation.md)
- [ADK Adapter](docs/features/adk.md) · [Flows vs Flow Type](docs/features/flow-vs-flows.md)

**Operating it**
- [Security & Production](docs/features/production.md) — sandboxing, TLS, access control
- [CLI reference](docs/features/cli.md) · [Events](docs/features/events.md)
- [Observability](docs/features/telemetry.md) · [Config errors](docs/features/config-errors.md)
- [Training](docs/features/training.md) · [Testing](docs/features/testing.md)
- [Guardrails](docs/features/guardrails.md) · [Sandboxing](docs/features/sandbox.md)
- [Dragonfly deployment](docs/ops/dragonfly.md)

---

## 🤝 Community

- [GitHub](https://github.com/Ecook14/gocrewwai)
- Report issues, propose features, and discuss on the repository.
