# Feature Deep Dive: Gocrew CLI ⚓💻🚀

The Gocrewwai CLI (`gocrew`) is the primary interface for scaffolding projects, managing agentic missions, and launching the real-time **Dashboard**. Built with Go's native binary capabilities, it ensures a lightning-fast and reliable management experience.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** The `gocrew` CLI supports **Rapid Scaffolding**, **Replay Logic**, and **Headless Dashboard** deployment.

---

## 🏗️ Installation

Install the global Gocrewwai binary directly from the GitHub repository:

```bash
go install github.com/Ecook14/gocrewwai/cmd/gocrew@latest
```

## 🚀 Key Commands

### 1. Project Scaffolding (`create`)
Scaffold a complete, production-ready Gocrewwai project in seconds. This creates a standard folder structure with `agents/`, `tasks/`, and a `main.go` using the **Elite Style** configuration.

```bash
gocrew create my-awesome-project
```

### 2. Live Dashboard (`--ui`)
Launch the real-time **Glassmorphic Dashboard** to watch your agents thought process and handle **Human-in-the-Loop** approvals. This command is typically used when running a crew mission from the CLI.

```bash
gocrew run mission.go --ui
```

### 3. Crew Replay (`replay`)
If a crew execution fails or you need to reproduce a specific behavior, use the `replay` command. It uses the persistence layer to "rewind" the execution to a specific **ThreadID** or **TaskID**.

```bash
gocrew replay thread_abc123
```

---

## 🛡️ Production Deployment (Headless Mode)

For servers and CI/CD environments, Gocrewwai supports a **Headless Mode**. This allows you to run crews without the interactive TUI, while still providing full **OpenTelemetry** tracing and logging to your remote O11y collector.

```bash
gocrew run mission.go --headless
```

## 📊 CLI Observability

All CLI commands in Gocrewwai are automatically instrumented. You can monitor the performance of your `create`, `run`, and `replay` commands using the same **OTEL** standards used in the core engine.

---

[Back to Production Guide](./production.md) | [Next: Testing](./testing.md)
