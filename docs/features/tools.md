# Feature Deep Dive: Tools ⚓🧰🛠️

Tools are the interface between your agents and the outside world. Gocrewwai agents can use tools to search the web, execute code, browse websites, and interact with external APIs.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai includes 24+ high-performance, built-in tools with native support for **Sandboxed Code Execution** and **Browser Automation**.

---

## 🏗️ Built-in Tools (Elite Style)

Gocrewwai includes a rich set of production-ready tools available directly via the `gocrew` SDK.

| Tool | SDK Constructor | Description |
| :--- | :--- | :--- |
| **SearchWeb** | `gocrew.NewSearchWebTool` | Generic web search (Serper, Google, etc.). |
| **Exa Search** | `gocrew.NewExaTool` | Vector-indexed neural search for high-quality results. |
| **Browser** | `gocrew.NewBrowserTool` | Automated web navigation and scraping (multi-modal). |
| **Calculator** | `gocrew.NewCalculatorTool` | Precise mathematical operations. |
| **Code Interpreter** | `gocrew.NewCodeInterpreter` | Safe, sandboxed Python and Shell execution. |
| **File Systems** | `gocrew.NewFileReadTool` | Chrooted file reading, writing, and editing. |

## 🚀 Using a Tool

Simply pass the tool instances to your agent's configuration:

```go
package main

import "github.com/Ecook14/gocrewwai/gocrew"

func main() {
    searchTool := gocrew.NewSearchWebTool()
    browserTool := gocrew.NewBrowserTool()

    researcher := gocrew.NewAgent(gocrew.AgentConfig{
        Tools: []gocrew.Tool{searchTool, browserTool},
    })
}
```

## 🛡️ Tool Guardrails & Security

### 1. Manual Approval (HITL)
Configure specific tools (e.g., `github.delete_issue`) to require manual approval before execution. The engine will pause and wait for a signal from the **Dashboard**.

### 2. Execution Sandboxing
All code-execution tools (Python, Shell) are isolated from the host system. Gocrewwai supports **Docker**, **WASM**, and **E2B** sandboxing out-of-the-box.

### 3. Tool Error Feedback
If a tool fails, the error message is automatically fed back to the agent's reasoning loop. The agent can then analyze the error, adjust its parameters, and attempt a retry.

---

[Back to Telemetry Guide](./telemetry.md) | [Next: Files](./files.md)
