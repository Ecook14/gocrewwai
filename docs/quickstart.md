# Quickstart Guide: Building Your First Crew 🚀

Welcome to **Crew-GO**! I'm so excited you're ready to start building autonomous agents with me. Whether you're here to build a simple researcher or a massive, distributed reasoning engine, this guide will walk you through building a complete, functioning Crew from scratch.

I've designed this to be as straightforward as possible, while still showing off some of the powerful Go-native features we've packed underneath the hood. Let's dive in!

---

## Phase 1: Installation & Setup

Before we start writing Go code, let's make sure your environment is ready.

### Prerequisites
1. **Go 1.22+**: Required for modern features like Go Generics (which we use for safely parsing AI responses!).
2. **OpenAI API Key**: (Or a compatible proxy like Groq, Ollama, or Anthropic if you change the initialized client).

### Install the Framework
Let's pull the Crew-GO library into a fresh Go module:

```bash
mkdir my-first-crew
cd my-first-crew
go mod init my-first-crew
go get github.com/Ecook14/gocrewwai
```

*(If you ever want to scaffold a project automatically, we also have a CLI you can install via `go install github.com/Ecook14/gocrewwai/cmd/gocrewwai@latest`!)*

---

## Phase 2: Writing the Code

Create a `main.go` file in your new folder. We are going to build a **"Tech News Summarization Crew"**. Our team will consist of two AI agents:
1. A **Researcher** who scours the internet for the latest tutorials.
2. A **Writer** who takes that research and turns it into a catchy blog post.

### Step 1: Initialize the LLM "Brain"
Every agent needs an LLM to think. Crew-GO ships with highly optimized clients out of the box.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/Ecook14/gocrew/pkg/agents"
	"github.com/Ecook14/gocrew/pkg/crew"
	"github.com/Ecook14/gocrew/pkg/llm"
	"github.com/Ecook14/gocrew/pkg/tasks"
	"github.com/Ecook14/gocrew/pkg/tools"
)

func main() {
    // 1. Setup the LLM Client. This client inherently handles Retries, Rate Limits, and Telemetry!
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        panic("OPENAI_API_KEY is required in your environment!")
    }
    client := llm.NewOpenAIClient(apiKey)
```

### Step 2: Define the Agents
Agents are defined by their `Role`, `Goal`, and `Backstory`. We use a "Fluent Builder" pattern in Go to make this incredibly clean to read.

*Notice how we equip the Researcher with a `SearchWebTool` below, so it isn't just generating text—it can actually run live Google searches!*

```go
    // 2. Create the Researcher Agent
    researcher := agents.NewAgentBuilder().
        Role("Senior Tech Researcher").
        Goal("Discover the absolute latest developments in the Go programming language.").
        Backstory("You are a relentless tech journalist who digs deep to find cutting-edge information. You always verify your sources.").
        LLM(client).
        Tools(tools.NewSearchWebTool()). // Give them internet access!
        Verbose(true).                   // Let's watch them think in the console!
        Build()

    // 3. Create the Writer Agent
    writer := agents.NewAgentBuilder().
        Role("Senior Technical Writer").
        Goal("Craft engaging, accurate, and concise blog posts about technology.").
        Backstory("You are an expert copywriter known for your clear and engaging tone. You never plagiarize, and you always summarize complex topics simply.").
        LLM(client).
        Build()
```

### Step 3: Define the Tasks
Agents need jobs to do. Tasks dictate exactly what each agent should accomplish. Tasks can also be chained together so one agent waits for another to finish!

```go
    // 4. Create the Research Task
    researchTask := tasks.NewTaskBuilder().
        Description("Search the web for news about the 'Go 1.24 Release' or 'Go memory management updates'. Gather at least 3 key links and summarize them.").
        Agent(researcher).
        Build()

    // 5. Create the Writing Task
    writingTask := tasks.NewTaskBuilder().
        Description("Using the context provided by the researcher, write a catchy 3-paragraph blog post summarizing the latest Go updates.").
        Agent(writer).
        Context(researchTask). // Explicit dependency: The writer waits for the research to finish!
        Build()
```

### Step 4: Assemble & Kickoff the Crew
A `Crew` is the underlying Go execution engine that manages the flow. We will use the default `Sequential` process, meaning `researchTask` will finish entirely before `writingTask` begins.

```go
    // 6. Assemble the Crew
    techCrew := crew.NewCrewBuilder().
        Agents(researcher, writer).
        Tasks(researchTask, writingTask).
        Process(crew.Sequential).
        Verbose(true).
        Build()

    // 7. Kickoff Execution!
    slog.Info("🚀 Kicking off the Tech News Crew...")
    
    // You can use context.WithTimeout(ctx, 10*time.Minute) to enforce hard execution limits.
    result, err := techCrew.Kickoff(context.Background())
    if err != nil {
        slog.Error("Crew execution failed!", slog.String("error", err.Error()))
        os.Exit(1)
    }

    // 8. Print the Final Output
    fmt.Println("\n==================================")
    fmt.Println("🎉 FINAL BLOG POST 🎉")
    fmt.Println("==================================")
    fmt.Println(result)
}
```

---

## Phase 3: Watch it Run!

To execute your crew, just run:
```bash
export OPENAI_API_KEY="your-api-key-here"
go run main.go
```

You will see `slog` output flooding your terminal as the ReAct loop triggers, the Researcher searches the web, parses the results, and hands the context to the Writer.

### Want to see the beautiful Glassmorphic Web UI?
Watching terminal logs is fun, but observing your agents think in a real-time web dashboard is even better. 

Simply import our server package into your `main.go`:
```go
import "github.com/Ecook14/gocrew/pkg/dashboard"
```
And add this line right before you call `techCrew.Kickoff(...)`:
```go
// Start the real-time websocket dashboard on port 8080!
dashboard.Start("8080")
```

Then, pop open your browser to `http://localhost:8080/web-ui`. You can even **Stop and Start** the engine live or create entire new agents from the **Creator Studio** without restarting your Go process!

---

## What's Next?
You've successfully built your first Crew-GO application! From here, I highly recommend checking out:
- **[Advanced Usage](advanced_usage.md)** to learn how to make agents run in parallel (Hierarchical) or extract strict JSON strings into Go Structs.
- **[Tools Guide](features/tools.md)** to learn how to spin up Docker containers for your agents to execute their own code inside!

If you build something awesome, or notice a way we can improve the framework, **please drop by the GitHub repo and submit a Pull Request!** We are building the smartest Go framework together.
