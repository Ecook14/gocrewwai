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

	// 2. Feature Toggles at Agent Level
	researcher := gocrew.NewAgent(gocrew.AgentConfig{
		Role:          "Researcher",
		Goal:          "Find the latest AI trends",
		Backstory:     "Expert researcher",
		LLM:           model,
		SelfHealing:   true,
		MaxIterations: 5,
	})

	// 3. Tool-Specific Configuration
	interpreter := gocrew.NewCodeInterpreterTool()
	researcher.Tools = []gocrew.Tool{interpreter}

	// 4. Crew-Level Configuration
	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents: []gocrew.CoreAgent{researcher},
		Tasks: []*gocrew.Task{
			{Description: "Analyze the current state of Go for AI agents."},
		},
		Verbose: true,
	})

	fmt.Println("🚀 Kicking off with developer-defined configuration...")
	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nResult: %s\n", result)
}
