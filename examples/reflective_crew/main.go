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

	writer := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Creative Writer",
		Goal:      "Write a short story about a robot learning to paint.",
		Backstory: "Whimsical storyteller",
		LLM:       model,
		Verbose:   true,
	})

	// Enable Self-Critique for the agent
	writer.SelfCritique = true

	task := &gocrew.Task{
		Description: "Write a 2-sentence story about a painting robot.",
		Agent:       writer,
	}

	// Use Reflective process for manager review
	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{writer},
		Tasks:   []*gocrew.Task{task},
		Verbose: true,
	})

	fmt.Println("🚀 Starting Reflective Crew (Agent Self-Critique + Manager Review)...")
	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nFinal Approved Story: %s\n", result)
}
