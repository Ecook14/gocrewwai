package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Ecook14/crewai-go/pkg/agents"
	"github.com/Ecook14/crewai-go/pkg/crew"
	"github.com/Ecook14/crewai-go/pkg/llm"
	"github.com/Ecook14/crewai-go/pkg/tasks"
)

func main() {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY is required")
	}

	client := llm.NewOpenRouterClient(apiKey, "google/gemini-2.0-flash-lite-preview-02-05:free")

	// 1. Define Agents
	researcher := agents.NewAgentBuilder().
		Role("Researcher").
		Goal("Find the latest tech trends.").
		LLM(client).
		Build()

	writer := agents.NewAgentBuilder().
		Role("Writer").
		Goal("Write a blog post.").
		LLM(client).
		Build()

	// 2. Define Tasks
	task1 := &tasks.Task{
		Description: "Identify 3 trends in AI for 2026.",
		Agent:       researcher,
	}

	task2 := &tasks.Task{
		Description: "Write a short summary of the AI trends.",
		Agent:       writer,
	}

	// 3. Create Crew with Planning
	myCrew := crew.NewCrew(
		[]*agents.Agent{researcher, writer},
		[]*tasks.Task{task1, task2},
		crew.WithPlanning(true),
		crew.WithVerbose(true),
	)

	// 4. Kickoff
	fmt.Println("🚀 Kicking off Crew with Advanced Planning Phase...")
	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n--- Final Result ---\n%v\n", result)
}
