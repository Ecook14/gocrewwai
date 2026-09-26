# Feature Deep Dive: Gocrew CLI ⚓💻🚀

The Gocrewwai CLI (`gocrew`) is the primary interface for scaffolding projects, managing agentic missions, and launching the real-time **Dashboard**. Built with Go's native binary capabilities, it ensures a lightning-fast and reliable management experience.

---

> [!IMPORTANT]
> **Status: v0.9.0 (Alpha → Beta).** The `gocrew` CLI supports **Rapid Scaffolding**, **Crew Kickoff**, and **Embedded Dashboard** deployment.

---

## 🏗️ Installation

Install the global Gocrewwai binary directly from the GitHub repository:

```bash
go install github.com/Ecook14/gocrewwai/cmd/gocrew@latest
```

## 🚀 Key Commands

### 1. Project Scaffolding (`create`)
Scaffold a complete, production-ready Gocrewwai project in seconds. This creates a standard folder structure with `src/`, `config/`, and `tools/` using the **Elite Style** configuration.

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
Run a crew with your project config. Gocrew merges the project config
with environment variables and executes the crew in a dedicated process
with full OpenTelemetry tracing.

```bash
gocrew kickoff
```

### 5. Runner Execution (`run`)
Execute a specific mission file. This is the delegator's default workflow
for running agent assignments with full tool access and CLI output.

```bash
# Run a mission file directly
gocrew run ./mission.go

# Run with an explicit API key (overrides config)
gocrew run ./mission.go -k $OPENAI_API_KEY
```

### 6. Version Check (`version`)
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
