package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/crewai-go/internal/server"
	"github.com/Ecook14/crewai-go/pkg/agents"
	"github.com/Ecook14/crewai-go/pkg/crew"
	"github.com/Ecook14/crewai-go/pkg/llm"
	"github.com/Ecook14/crewai-go/pkg/tasks"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := llm.NewOpenAIClient(apiKey)

	analyst := agents.NewAgent("Analyst", "Analyze data", "Expert analyst", model)
	coder := agents.NewAgent("Coder", "Write code", "Senior developer", model)
	reviewer := agents.NewAgent("Reviewer", "Review work", "Detailed reviewer", model)

	task1 := &tasks.Task{Description: "Analyze the stock market trends for AI.", Agent: analyst}
	task2 := &tasks.Task{Description: "Write a Python script to track these trends.", Agent: coder}
	
	// Complex Dependency: Task 3 starts ONLY after 1 and 2 complete
	task3 := &tasks.Task{
		Description:  "Review the analysis and the code for accuracy.",
		Agent:        reviewer,
		Dependencies: []*tasks.Task{task1, task2},
	}

	myCrew := crew.NewCrew(
		[]*agents.Agent{analyst, coder, reviewer},
		[]*tasks.Task{task1, task2, task3},
		crew.WithProcess(crew.Graph),
		crew.WithVerbose(true),
	)

	fmt.Println("🚀 Starting Graph (DAG) Demo (Task 1 & 2 will run in parallel):")
	
	go server.StartDashboardServer("8081")
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
