# Getting Started with Gocrewwai 🚀

Welcome to Gocrewwai, the high-performance, asynchronous orchestration framework for AI agents. This guide will help you set up your environment and run your first multi-agent crew in minutes.

## 🛠️ Prerequisites

- **Go**: Version 1.21 or higher.
- **API Key**: An API key from a supported provider (e.g., [OpenAI](https://platform.openai.com/), [Anthropic](https://console.anthropic.com/)).

## 📦 Installation

Initialize your Go project and install the Gocrewwai SDK:

```bash
go mod init my-agent-app
go get github.com/Ecook14/gocrewwai/gocrew@latest
```

## 🚀 Your First Crew (Quickstart)

Create a file named `main.go` and paste the following code. This simple crew features a **Researcher** agent and a **Writer** agent working sequentially.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Ecook14/gocrewwai/gocrew"
)

func main() {
	// 1. Setup API Key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set OPENAI_API_KEY environment variable")
	}

	// 2. Initialize LLM via SDK
	model := gocrew.NewOpenAI(apiKey, "gpt-4o")

	// 3. Define Agents
	researcher := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Strategic Researcher",
		Goal:      "Find the key benefits of using Go for AI agents.",
		Backstory: "Expert in performance optimization and systems programming.",
		LLM:       model,
	})

	writer := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Technical Storyteller",
		Goal:      "Write a short, engaging summary based on the research.",
		Backstory: "Award-winning tech writer specializing in simplified explanations.",
		LLM:       model,
	})

	// 4. Define Tasks
	researchTask := gocrew.NewTask(gocrew.TaskConfig{
		Description: "Research why Go's concurrency model is superior for AI orchestration.",
		Agent:       researcher,
	})

	writeTask := gocrew.NewTask(gocrew.TaskConfig{
		Description: "Using the research, write a 3-bullet point summary for a technical blog.",
		Agent:       writer,
		Context:     []*gocrew.Task{researchTask},
	})

	// 5. Assemble and Kickoff the Crew
	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{researcher, writer},
		Tasks:   []*gocrew.Task{researchTask, writeTask},
		Verbose: true,
	})

	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	fmt.Printf("\n--- 🏁 RESULT ---\n%s\n", result)
}
```

## 🏃 Running the Demo

Set your API key and run the application:

```bash
export OPENAI_API_KEY='your-api-key-here'
go run main.go
```

---

[Explore Core Concepts](./CORE_CONCEPTS.md) | [Agents Guide](./features/agents.md) | [Tasks Guide](./features/tasks.md)
