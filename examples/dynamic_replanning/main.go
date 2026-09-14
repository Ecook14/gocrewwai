package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/gocrew"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := gocrew.NewOpenAI(apiKey, "gpt-4o")

	researcher := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Researcher",
		Goal:      "Research the current weather in SF.",
		Backstory: "Weather expert",
		LLM:       model,
	})

	task := &gocrew.Task{
		Description: "Find the current weather in San Francisco.",
		Agent:       researcher,
	}

	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{researcher},
		Tasks:   []*gocrew.Task{task},
		Verbose: true,
	})

	fmt.Println("🚀 Starting Dynamic Re-planning Demo...")
	fmt.Println("(The manager might decide to add a 'Packing Suggestion' task after seeing the weather)")
	
	_, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Printf("\nFinal Task List Length: %d (Check if a task was added!)\n", len(myCrew.Tasks))
}
