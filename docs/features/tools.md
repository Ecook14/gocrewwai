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
| **Arxiv** | `gocrew.NewArxivTool` | Search academic papers on Arxiv. |
| **Wikipedia** | `gocrew.NewWikipediaTool` | Extract information from Wikipedia. |
| **Shell** | `gocrew.NewShellTool` | Execute local or remote shell commands. |
| **AskHuman** | `gocrew.NewAskHumanTool` | Explicitly prompt for human input mid-task. |

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

## 🛠️ Creating Custom Tools

Building a custom tool in Gocrewwai is straightforward. You only need to implement the `tools.Tool` interface. The `gocrew.BaseTool` makes this easy by handling the boilerplate.

**Example: A Custom Weather Tool**

```go
package main

import (
	"context"
	"fmt"
	"github.com/Ecook14/gocrewwai/gocrew"
)

// 1. Define the Expected Input Schema
type WeatherInput struct {
	City  string `json:"city" jsonschema:"description=The name of the city"`
	Units string `json:"units" jsonschema:"enum=metric,enum=imperial"`
}

func NewWeatherTool() gocrew.Tool {
	return gocrew.NewBaseTool(
		"get_weather",
		"Retrieves the current weather for a given city.",
		WeatherInput{}, // Pass an empty struct so the engine can reflect the JSON schema
		func(ctx context.Context, rawInput interface{}) (string, error) {
			
            // 2. Cast the raw input to your struct
			input, ok := rawInput.(*WeatherInput)
			if !ok {
				return "", fmt.Errorf("invalid input type")
			}

			// 3. Execute your business logic
			// (In a real app, you'd make an HTTP call to a weather API here)
			result := fmt.Sprintf("The weather in %s is currently 72 degrees (%s).", 
                                  input.City, input.Units)
            
			return result, nil
		},
	)
}
```

You can now inject `NewWeatherTool()` directly into any `AgentConfig`. The engine will automatically convert your `WeatherInput` struct into a JSON schema that the LLM understands!

---

[Back to Telemetry Guide](./telemetry.md) | [Next: Files](./files.md)
