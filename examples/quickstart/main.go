package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Ecook14/gocrewwai/gocrew"
)

func main() {
	// 1. Setup API Key (Uses OpenAI by default for this quickstart)
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ Error: Please set OPENAI_API_KEY environment variable.")
		return
	}

	model := gocrew.NewOpenAI(apiKey, "gpt-4o")

	// 2. Define a simple Agent (Elite Style)
	researcher := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Technical Researcher",
		Goal:      "Summarize the key benefits of using Go for AI agents.",
		Backstory: "Expert Go developer turned AI architect.",
		LLM:       model,
	})

	// 3. Define a simple Task (Elite Style)
	task := gocrew.NewTask(gocrew.TaskConfig{
		Description: "Write a 3-bullet point summary of why Go is great for AI.",
		Agent:       researcher,
	})

	// 4. Assemble and Kickoff the Crew (Elite Style)
	quickCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{researcher},
		Tasks:   []*gocrew.Task{task},
		Verbose: true,
	})

	fmt.Println("🚀 GOCREW QUICKSTART INITIATED (Elite Style)...")
	result, err := quickCrew.Kickoff(context.Background())
	if err != nil {
		log.Fatalf("❌ Execution failed: %v", err)
	}

	fmt.Printf("\n--- 🏁 QUICKSTART RESULT ---\n%s\n", result)
}
