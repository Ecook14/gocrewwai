package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"github.com/Ecook14/gocrewwai/pkg/tools"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := llm.NewOpenAIClient(apiKey)

	researcher := agents.NewAgent("Researcher", "Research the current weather in SF.", "Weather expert", model)
	
	task := &tasks.Task{
		Description: "Find the current weather in San Francisco.",
		Agent:       researcher,
	}

	// Use hierarchical mode so the manager can re-plan
	myCrew := crew.NewCrew(
		[]*agents.Agent{researcher},
		[]*tasks.Task{task},
		crew.WithProcess(crew.Hierarchical),
		crew.WithVerbose(true),
	)

	fmt.Println("🚀 Starting Dynamic Re-planning Demo...")
	fmt.Println("(The manager might decide to add a 'Packing Suggestion' task after seeing the weather)")
	
	_, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Printf("\nFinal Task List Length: %d (Check if a task was added!)\n", len(myCrew.Tasks))
}
