# Crew-GO Quickstart Guide 🚀

Welcome to **Crew-GO**, the unofficial Go-native implementation of the popular CrewAI framework. Building agentic applications in Go ensures high performance, native static compilation, and deep structured output parsing.

This guide will show you how to set up your first simple Crew.

## 1. Installation

Install the package directly into your Go Module:

```bash
go get github.com/Ecook14/crewai-go
```

*(Optional)* If you want to automatically scaffold a standard project layout (`/src`, `/config/agents.yaml`), install the CLI mapping tool:

```bash
go install github.com/Ecook14/crewai-go/cmd/crew-go@latest
crew-go create my_ai_project
```

## 2. Setting Up an Agent

Agents are the independent actors in your Crew. Provide them a `Role`, a `Goal`, and a `Backstory`. 

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/crewai-go/pkg/agents"
	"github.com/Ecook14/crewai-go/pkg/llm"
)

func main() {
    // 1. Initialize the official OpenAI Client mappings
    client := llm.NewOpenAIClient(os.Getenv("OPENAI_API_KEY"))

    // 2. Define your Agents
    coder := &agents.Agent{
        Role:      "Senior Go Engineer",
        Goal:      "Write impeccable, fast, and secure Go code.",
        Backstory: "You are a veteran engineer analyzing software architecture.",
        Verbose:   true,
        LLM:       client,
    }
}
```

## 3. Defining Tasks

Tasks dictate what the `Agent` should do. A Task MUST be bound to a specific agent.

```go
import "github.com/Ecook14/crewai-go/pkg/tasks"

// ... inside main()
task := &tasks.Task{
    Description: "Analyze the benefits of migrating from Python to Go for backend services.",
    Agent:       coder,
}
```

## 4. Assembling the Crew

Crews organize Agents and Tasks and define the orchestration pattern. By default, tasks execute **Sequentially**. 

```go
import "github.com/Ecook14/crewai-go/pkg/crew"

// ... inside main()
techCrew := crew.Crew{
    Agents:  []*agents.Agent{coder},
    Tasks:   []*tasks.Task{task},
    Process: crew.Sequential, // Tasks run one by one
    Verbose: true, // Output rich slog telemetry
}

// 5. Kickoff!
// Use Contexts to set upper-level timeouts and tracing!
result, err := techCrew.Kickoff(context.Background())
if err != nil {
    panic(err)
}

fmt.Printf("Execution Result:\n%v\n", result)
```

## Next Steps
Now that you know how to build a simple sequential loop, learn how to enforce **JSON Outputs**, assign **Tools**, and orchestrate **Parallel Execution** in the [Advanced Usage Guide](./advanced_usage.md).
