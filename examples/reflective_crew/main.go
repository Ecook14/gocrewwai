package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := llm.NewOpenAIClient(apiKey)

	writer := agents.NewAgent(
		"Creative Writer",
		"Write a short story about a robot learning to paint.",
		"Whimsical storyteller",
		model,
		agents.WithVerbose(true),
	)

	// Enable Self-Critique for the agent
	writer.SelfCritique = true

	task := &tasks.Task{
		Description: "Write a 2-sentence story about a painting robot.",
		Agent:       writer,
	}

	// Use Reflective process for manager review
	myCrew := crew.NewCrew(
		[]core.Agent{writer},
		[]*tasks.Task{task},
		crew.WithProcess(crew.Reflective),
		crew.WithVerbose(true),
	)

	fmt.Println("🚀 Starting Reflective Crew (Agent Self-Critique + Manager Review)...")
	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nFinal Approved Story: %s\n", result)
}
