package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
)

func main() {
	// 1. Setup API Key (Uses OpenAI by default for this quickstart)
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ Error: Please set OPENAI_API_KEY environment variable.")
		return
	}

	model := llm.NewOpenAIClient(apiKey)

	// 2. Define a simple Agent
	researcher := agents.NewAgent(agents.AgentConfig{
		Role:      "Technical Researcher",
		Goal:      "Summarize the key benefits of using Go for AI agents.",
		Backstory: "Expert Go developer turned AI architect.",
		LLM:       model,
	})

	// 3. Define a simple Task
	task := tasks.NewTask("Write a 3-bullet point summary of why Go is great for AI.", researcher)

	// 4. Assemble and Kickoff the Crew
	quickCrew := crew.NewCrew([]core.Agent{researcher}, []*tasks.Task{task}, crew.WithVerbose(true))

	fmt.Println("🚀 CREW-GO QUICKSTART INITIATED...")
	result, err := quickCrew.Kickoff(context.Background())
	if err != nil {
		log.Fatalf("❌ Execution failed: %v", err)
	}

	fmt.Printf("\n--- 🏁 QUICKSTART RESULT ---\n%s\n", result)
}
