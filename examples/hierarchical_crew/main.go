package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/gocrew"
	"github.com/Ecook14/gocrewwai/pkg/dashboard"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set OPENAI_API_KEY environment variable")
		return
	}

	// 1. Initialize LLM via SDK
	model := gocrew.NewOpenAI(apiKey, "gpt-4o")

	// 2. Define Agents (Elite Style)
	researcher := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Researcher",
		Goal:      "Find the latest developments in AI agents.",
		Backstory: "You are a seasoned technology researcher with an eye for detail.",
		LLM:       model,
		Verbose:   true,
	})

	writer := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Technical Writer",
		Goal:      "Write a compelling blog post about AI agents.",
		Backstory: "You are a skilled writer who can explain complex topics simply.",
		LLM:       model,
		Verbose:   true,
	})

	// 3. Define Tasks (Elite Style)
	researchTask := gocrew.NewTask(gocrew.TaskConfig{
		Description: "Research the current state of autonomous AI agents in 2024.",
		Agent:       researcher,
	})

	writeTask := gocrew.NewTask(gocrew.TaskConfig{
		Description: "Using the research provided, write a 3-paragraph blog post highlighting the key trends.",
		Agent:       writer,
	})

	// 4. Assemble Crew with Hierarchical Process (Elite Style)
	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:      []gocrew.CoreAgent{researcher, writer},
		Tasks:       []*gocrew.Task{researchTask, writeTask},
		Process:     gocrew.Hierarchical,
		ManagerLLM:  model, // Required for hierarchical mode
		Verbose:     true,
	})

	// 5. Kickoff
	fmt.Println("## Starting Hierarchical Crew Execution (Elite Style) ##")
	
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
