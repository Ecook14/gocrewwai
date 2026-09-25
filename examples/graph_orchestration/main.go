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
	model := gocrew.NewOpenAI(apiKey, "gpt-4o")

	analyst := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Analyst",
		Goal:      "Analyze data",
		Backstory: "Expert analyst",
		LLM:       model,
	})
	coder := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Coder",
		Goal:      "Write code",
		Backstory: "Senior developer",
		LLM:       model,
	})
	reviewer := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Reviewer",
		Goal:      "Review work",
		Backstory: "Detailed reviewer",
		LLM:       model,
	})

	task1 := &gocrew.Task{Description: "Analyze the stock market trends for AI.", Agent: analyst}
	task2 := &gocrew.Task{Description: "Write a Python script to track these trends.", Agent: coder}

	task3 := &gocrew.Task{
		Description:  "Review the analysis and the code for accuracy.",
		Agent:        reviewer,
		Dependencies: []*gocrew.Task{task1, task2},
	}

	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{analyst, coder, reviewer},
		Tasks:   []*gocrew.Task{task1, task2, task3},
		Verbose: true,
	})

	fmt.Println("🚀 Starting Graph (DAG) Demo (Task 1 & 2 will run in parallel):")

	dashboard.Start("8081")
	fmt.Println("🖥️  Dashboard active at http://localhost:8081/web-ui - Watch the parallel execution traces!")

	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nFinal Result: %s\n", result)

	fmt.Println("✅ Demo finished. Keep the dashboard open to review the logs!")
	select {}
}
