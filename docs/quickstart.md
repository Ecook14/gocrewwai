# Crew-GO Exhaustive Quickstart Guide 🚀

Welcome to **Crew-GO**, the most advanced Go-native implementation of autonomous agent orchestration. This guide will walk you through building a complete, functioning Crew from scratch, explaining every step in deep detail.

## Phase 1: Installation & Setup

Before we start coding, ensure your environment is ready.

### Prerequisites
1. **Go 1.22+**: Required for modern features.
2. **OpenAI API Key**: (Or a compatible proxy like groq/ollama if you change the base URL).

### Install the Framework
Install the library into your Go module:

```bash
mkdir my-first-crew
cd my-first-crew
go mod init my-first-crew
go get github.com/Ecook14/crewai-go
```

## Phase 2: Building Your First Application

Create a `main.go` file. We are going to build a **"Tech News Summarization Crew"**.

### Step 1: Initialize the Client
Every agent needs an LLM "Brain". Crew-GO ships with a highly optimized OpenAI client out of the box.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/Ecook14/crewai-go/pkg/agents"
	"github.com/Ecook14/crewai-go/pkg/crew"
	"github.com/Ecook14/crewai-go/pkg/llm"
	"github.com/Ecook14/crewai-go/pkg/tasks"
	"github.com/Ecook14/crewai-go/pkg/tools"
)

func main() {
    // 1. Setup the LLM Client
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        panic("OPENAI_API_KEY is required")
    }
    client := llm.NewOpenAIClient(apiKey)
```

### Step 2: Define the Agents
Agents are defined by their `Role`, `Goal`, and `Backstory`. We will create two agents: a Researcher and a Writer.

*Notice how we equip the Researcher with a `SearchWebTool` so it can actually browse the internet!*

```go
    // 2. Create the Researcher Agent using the Fluent Builder (Recommended DX)
    researcher := agents.NewAgentBuilder().
        Role("Senior Tech Researcher").
        Goal("Discover the absolute latest developments in the Go programming language.").
        Backstory("You are a relentless tech journalist who digs deep to find cutting-edge information.").
        LLM(client).
        Tools(tools.NewSearchWebTool()).
        Verbose(true).
        Build()

    // 3. Create the Writer Agent
    writer := agents.NewAgentBuilder().
        Role("Senior Technical Writer").
        Goal("Craft engaging, accurate, and concise blog posts about technology.").
        Backstory("You are an expert copywriter known for your clear and engaging tone.").
        LLM(client).
        Build()
```

### Step 3: Define the Tasks
Tasks dictate exactly what each agent should do. Tasks can be chained together using `Context`.

```go
    // 4. Create the Research Task using the Task Builder
    researchTask := tasks.NewTaskBuilder().
        Description("Search the web for news about the 'Go 1.24 Release' or 'Go memory management updates in 2026'. Gather key links and summaries.").
        Agent(researcher).
        Build()

    // 5. Create the Writing Task
    writingTask := tasks.NewTaskBuilder().
        Description("Using the context provided by the researcher, write a 3-paragraph blog post summarizing the latest Go updates.").
        Agent(writer).
        Context(researchTask). // Explicit dependency mapping!
        Build()
```

### Step 4: Assemble & Kickoff the Crew
A `Crew` manages the execution flow. We will use the default `Sequential` process, meaning `researchTask` will finish entirely before `writingTask` begins.

```go
    // 6. Assemble the Crew using the Crew Builder
    techCrew := crew.NewCrewBuilder().
        Agents(researcher, writer).
        Tasks(researchTask, writingTask).
        Process(crew.Sequential).
        Verbose(true).
        Build()

    // 7. Kickoff Execution
    slog.Info("🚀 Kicking off the Tech News Crew...")
    
    // We use context.Background(), but you could use context.WithTimeout(ctx, 10*time.Minute) to enforce hard limits!
    result, err := techCrew.Kickoff(context.Background())
    if err != nil {
        slog.Error("Crew execution failed", slog.String("error", err.Error()))
        os.Exit(1)
    }

    // 8. Print the Final Output
    fmt.Println("\n==================================")
    fmt.Println("🎉 FINAL BLOG POST 🎉")
    fmt.Println("==================================")
    fmt.Println(result)
}
```

## Phase 3: Execution & The Live Dashboard

You can simply run `go run main.go`, and watch the logs pour into your terminal.
But Crew-GO features an **Elite Real-time Dashboard**. 

To use it, you can programmatically start the server in your `main.go`, OR use the CLI.

### Option A: Using the CLI (Recommended)
If you built your project using the `crewai create` scaffolding tool, simply run:
```bash
~/go/bin/crewai kickoff --ui
```

### Option B: Programmatically in Code
Add this to the top of your `main.go` right after imports:
```go
import "github.com/Ecook14/crewai-go/internal/server"

func main() {
    // Start the dashboard server in the background
    go server.StartDashboardServer("8080")
    slog.Info("🖥️ Dashboard running at http://localhost:8080/web-ui")
    
    // ... rest of your code
}
```

Run your code, open your browser to `http://localhost:8080/web-ui`, and watch your agents think, search, and write in real-time!

---

## Next Steps

You've built a basic sequential Crew. To unlock the true power of Crew-GO, check out:
- **[USAGE.md](../USAGE.md)**: For detailed breakdowns of all 24 tools (including the Code Interpreter & Docker Sandboxes).
- **[Advanced Orchestration](advanced_usage.md)**: To learn how to execute tasks in parallel, use Hierarchical delegation, or build looping Cyclic Graphs!
