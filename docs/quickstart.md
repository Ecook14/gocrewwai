# Quickstart Guide: Building Your First Crew 🚀

Welcome to **Gocrew**! We're thrilled you're ready to build autonomous agent teams. Whether you're building a simple researcher or a complex reasoning engine, this guide will walk you through creating a functioning Crew in minutes.

---

## 🏗️ 1. Installation

Gocrew recommends using our unified SDK facade for the best developer experience.

### Initialize Your Project
```bash
mkdir my-first-crew
cd my-first-crew
go mod init my-first-crew
go get github.com/Ecook14/gocrewwai
```

---

## ✍️ 2. Writing Your First Crew

Create a `main.go`. We’ll build a **"Go Release Analyst"** team:
1. **Researcher**: Finds details about the latest Go release.
2. **Writer**: Summarizes the features for a blog post.

### The Code
```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/gocrew"
)

func main() {
	// 1. Setup the Brain (LLM)
	apiKey, _ := os.Getenv("OPENAI_API_KEY")
	model := gocrew.NewOpenAI(apiKey, "gpt-4o")

	// 2. Build the Agents
	researcher := gocrew.NewAgent(gocrew.AgentConfig{
		Role:            "Technical Researcher",
		Goal:            "Identify 3 key features of the latest Go release.",
		LLM:             model,
		Tools:           []gocrew.Tool{gocrew.NewSearchWebTool()},
		Verbose:         true,
	})

	writer := gocrew.NewAgent(gocrew.AgentConfig{
		Role:            "Tech Content Creator",
		Goal:            "Write a 3-paragraph blog post summarizing the research.",
		LLM:             model,
		Verbose:         true,
	})

	// 3. Define the Mission
	researchTask := gocrew.NewTask(gocrew.TaskConfig{
		Description: "Search for Go release notes and list 3 major changes.",
		Agent:       researcher,
	})

	blogTask := gocrew.NewTask(gocrew.TaskConfig{
		Description: "Write a blog post based on the research provided.",
		Agent:       writer,
		Context:     []*gocrew.Task{researchTask},
	})

	// 4. Assemble and Kickoff
	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{researcher, writer},
		Tasks:   []*gocrew.Task{researchTask, blogTask},
		Verbose: true,
	})

	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Crew failed: %v\n", err)
		return
	}

	fmt.Printf("\n--- MISSION COMPLETE ---\n%s\n", result)
}
```

---

## 🏃 3. Run It

Set your API key and execute:
```bash
export OPENAI_API_KEY=your-key
go run main.go
```

Watch the terminal as your Researcher performs web searches and hands off the data to the Writer!

---

## 🖥️ 4. Using the Dashboard

Want to watch your agents live? Gocrew includes a web UI that runs alongside the server.

1. Start the server: `go run cmd/server/main.go --api-port 8080 --web`
2. Open http://localhost:8080 in your browser
3. The embedded UI shows real-time agent activity via OpenTelemetry events
   dashboard.Start("8080")
   ```
3. Run your app and visit `http://localhost:8080/web-ui`.

---

## 🛠️ 5. Next Steps

- **[Usage Guide](../USAGE.md)**: Explore advanced sandboxing (Docker/E2B) and configuration.
- **[Memory Deep Dive](features/memory.md)**: Give your agents persistent long-term memory.
- **[Tool Alignment](features/tools.md)**: See our full list of 57 built-in tools.
