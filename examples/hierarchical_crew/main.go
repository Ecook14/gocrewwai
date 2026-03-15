package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/dashboard"
	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set OPENAI_API_KEY environment variable")
		return
	}

	// Initialize LLM
	model := llm.NewOpenAIClient(apiKey)

	// 1. Define Agents
	researcher := &agents.Agent{
		Role:      "Researcher",
		Goal:      "Find the latest developments in AI agents.",
		Backstory: "You are a seasoned technology researcher with an eye for detail.",
		LLM:       model,
		Verbose:   true,
	}

	writer := &agents.Agent{
		Role:      "Technical Writer",
		Goal:      "Write a compelling blog post about AI agents.",
		Backstory: "You are a skilled writer who can explain complex topics simply.",
		LLM:       model,
		Verbose:   true,
	}

	// 2. Define Tasks
	researchTask := &tasks.Task{
		Description: "Research the current state of autonomous AI agents in 2024.",
		Agent:       researcher,
	}

	writeTask := &tasks.Task{
		Description: "Using the research provided, write a 3-paragraph blog post highlighting the key trends.",
		Agent:       writer,
	}

	// 3. Assemble Crew with Hierarchical Process
	myCrew := crew.Crew{
		Agents:  []core.Agent{researcher, writer},
		Tasks:   []*tasks.Task{researchTask, writeTask},
		Process: crew.Hierarchical,
		Verbose: true,
	}

	// 4. Kickoff
	fmt.Println("## Starting Hierarchical Crew Execution ##")
	
	dashboard.Start("8081")
	fmt.Println("🖥️  Dashboard active at http://localhost:8081/web-ui - Watch the manager orchestrate!")

	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Execution failed: %v\n", err)
		return
	}

	fmt.Printf("\n## Final Result ##\n%v\n", result)
	
	fmt.Println("✅ Demo finished. Keep the dashboard open to review the logs!")
	select {}
}
