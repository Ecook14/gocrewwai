# Feature Deep Dive: Gocrew CLI ⚓💻🚀

The Gocrewwai CLI (`gocrew`) is the primary interface for scaffolding projects, managing agentic missions, and launching the real-time **Dashboard**. Built with Go's native binary capabilities, it ensures a lightning-fast and reliable management experience.

---

> [!IMPORTANT]
> **Status: v1.0.0-beta.4.** The `gocrew` CLI supports **Rapid Scaffolding**, **Crew Kickoff**, and **Embedded Dashboard** deployment.

---

## 🏗️ Installation

Install the global Gocrewwai binary directly from the GitHub repository:

```bash
go install github.com/Ecook14/gocrewwai/cmd/gocrew@latest
```

## 🚀 Key Commands

### 1. Project Scaffolding (`create`)
Scaffold a complete, production-ready Gocrewwai project in seconds. This creates a standard folder structure with `src/`, `config/`, and `tools/` using the standard configuration.

```bash
gocrew create my-awesome-project
```

### 2. Live Dashboard & Server
Launch the backend REST API and the real-time **Dashboard (web/)** to watch your agents' thought processes and handle **Human-in-the-Loop** approvals.

```bash
go run cmd/server/main.go --api-port 8080 --web
```

The `--api-port` flag sets the REST API port (default: 8080 or `API_PORT` env). The `--web` flag enables the embedded Visual Builder from `web/src/`. The `--mesh-port` flag sets the gRPC Agent Mesh port (default: 50051 or `MESH_PORT` env).

### 3. Single-Binary Distribution
Because Crew-GO is idiomatic Go, you can embed the entire React
`web/src/` dashboard directly into the server binary using `go:embed`.
This yields a highly portable, single ~44MB stripped binary
deployable anywhere:

```bash
CGO_ENABLED=0 go build -ldflags="-w -s" -o gocrew-agent cmd/server/main.go
./gocrew-agent --api-port 8080 --web
```

### 4. Publisher Operations (`kickoff`)
Run the demo crew. `kickoff` executes a built-in demo agent against
`OPENAI_API_KEY` with full OpenTelemetry tracing; pass `--ui` to also start
the dashboard on port 8080 (execution pauses until START is clicked).
Project-config-driven kickoff (merging `agents.yaml`/`tasks.yaml`) is roadmap.

```bash
gocrew kickoff
gocrew kickoff --ui
```

### 5. Runner Execution (`run`)
Run the `main.go` of the current project (`go run main.go`, extra args passed
through except `--ui`). Arbitrary mission files (`gocrew run ./mission.go`)
and `-k` key overrides are **not** supported — set keys via environment
(`OPENAI_API_KEY`, etc.) or `config.json`.

```bash
gocrew run
```

### 6. Training / Testing / Replay / Memory / Chat
`train`, `test`, `replay`, and `chat` validate args, then delegate to the
project (`go run main.go <subcommand>`) with a secrets-stripped environment —
the project owns its agents/config. Scaffolded projects (`gocrew create`)
ship a subcommand-aware `main.go`: `train -n` → `Crew.Train`, `test -n` →
N kickoffs with pass count, `replay -t` → `Crew.Replay`, `chat` → interactive
agent loop. Outside a project dir these fail closed (`main.go not found`).
`reset-memories` executes directly: `--store sqlite --conn <basename>` resets
the SQLite memory store (basename only, no traversal).

```bash
gocrew train -n 5
gocrew test -n 3
gocrew replay -t task_1
gocrew reset-memories --store sqlite --conn memory.db
gocrew chat
```

### 7. Deploy (`deploy`)
Build release binaries (`gocrew`, `gocrewwai-server`) into `--out DIR`
(default `./dist`) from the repo root. Webhook triggers for managed hosting
remain roadmap; binary + `Dockerfile` artifacts are the deploy unit.

```bash
gocrew deploy --out ./dist
```

### 8. Version Check (`version`)
Print the current Gocrewwai CLI version and build information.

```bash
gocrew version
```

---\n\n
## 🛡️ Production Deployment (Headless Mode)
\n\n
For servers and CI/CD environments, Gocrewwai supports a **Headless Mode**.
This allows you to run crews without the interactive TUI, while still
providing full **OpenTelemetry** tracing and logging to your remote O11y
collector.

```bash
gocrew run
```

## 📊 CLI Observability
\n\n
All CLI commands in Gocrewwai are automatically instrumented. You can
monitor the performance of your `create`, `run`, `kickoff`, and `version`
commands using the same **OTEL** standards used in the core engine.

---\n\n
[Back to Production Guide](./production.md) | [Next: Testing](./testing.md)
