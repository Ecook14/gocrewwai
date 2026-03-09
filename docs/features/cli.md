# Feature Deep Dive: Gocrew CLI ⚡

The `gocrew` CLI is the primary gateway for project management, scaffolding, and advanced operations like training and testing.

---

## 🏗️ Core Commands

### `gocrew create`
Generates a production-ready project structure based on modern Gocrew standards.
- Sets up `pkg/`, `cmd/`, and `internal/` directories.
- Creates a sample `main.go` using the unified SDK.

### `gocrew kickoff`
Executes your crew. 
- **`--ui`**: Launches the real-time observability dashboard.
- **`--verbose`**: Outputs the full agent thought stream to the terminal.

---

## 🎓 Training & Testing

The CLI handles the complex state management required for optimization:

### `gocrew train`
Runs a training session for your crew.
- **`--n 5`**: Run 5 iterations.
- Automatically pauses for human feedback and saves the "Advice Layer."

### `gocrew test`
Benchmarks your crew's performance.
- **`--model <model>`**: Specify the model to use as the "Grader."
- Outputs a comprehensive performance report with scores and consistency metrics.

---

## 🔄 Debugging & Replay

### `gocrew replay`
Replays a previous execution from a `StateFile`. Perfect for debugging specific tool failures or edge cases without re-running the entire crew.

### `gocrew reset-memories`
Wipes the persistent memory stores (SQL/Vector) for your agents, allowing you to start with a "clean slate."

---

## 🛠️ Global Installation

Install the CLI globally on your system:

```bash
go install github.com/Ecook14/gocrewwai/cmd/gocrew@latest
```

---
**Gocrew CLI** - Commanding excellence in agentic orchestration.
