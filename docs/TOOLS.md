# Tools in Gocrewwai 🛠️

Tools are the interface between your agents and the outside world. Gocrewwai agents can use tools to search the web, execute code, browse websites, and interact with external APIs.

## 🏗️ Built-in Tools

Gocrewwai includes a rich set of production-ready tools available directly via the `gocrew` SDK.

| Tool | SDK Constructor | Description |
| :--- | :--- | :--- |
| **SearchWeb** | `gocrew.NewSearchWebTool` | Generic web search (Serper, Google, etc.). |
| **Exa Search** | `gocrew.NewExaTool` | Vector-indexed neural search for high-quality results. |
| **Browser** | `gocrew.NewBrowserTool` | Automated web navigation and scraping (multi-modal). |
| **Calculator** | `gocrew.NewCalculatorTool` | Precise mathematical operations. |
| **Code Interpreter** | `gocrew.NewCodeInterpreter` | Safe, sandboxed Python and Shell execution. |
| **File Systems** | `gocrew.NewFileReadTool` | Chrooted file reading, writing, and editing. |
| **Arxiv/Wiki** | `gocrew.NewArxivTool` | Specialized academic and general knowledge search. |

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

## 🧠 Creating Custom Tools

Gocrewwai makes it incredibly easy to wrap any Go function as a tool.

### 1. Implement the `Tool` Interface
Any struct that implements the following method can be used as a tool:
```go
Execute(ctx context.Context, input map[string]interface{}) (string, error)
```

### 2. Example: Custom Weather Tool

```go
type WeatherTool struct {
    gocrew.BaseTool // Provides Name and Description
}

func (t *WeatherTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
    city := input["city"].(string)
    // Your custom API logic here...
    return "The weather in " + city + " is 72°F and sunny.", nil
}
```

---

[Back to LLM Providers Guide](./LLM_PROVIDERS.md) | [Next: Memory Guide](./MEMORY.md)
