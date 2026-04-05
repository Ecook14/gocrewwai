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
	if apiKey == "" {
		fmt.Println("❌ Error: Please set OPENAI_API_KEY environment variable.")
		return
	}

	model := gocrew.NewOpenAI(apiKey, "gpt-4o")

	// 1. Define an agent with a streaming callback (Elite Style)
	writer := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Poet",
		Goal:      "Write a short, beautiful haiku about Go programming.",
		Backstory: "A minimalist poet who understands concurrency and pointers.",
		LLM:       model,
		Verbose:   true,
		StepStreamCallback: func(token string) {
			fmt.Print(token)
		},
	})

	// 2. Define a simple Task (Elite Style)
	task := gocrew.NewTask(gocrew.TaskConfig{
		Description: "Write a haiku about Go.",
		Agent:       writer,
	})

	// 3. Assemble and Kickoff the Crew (Elite Style)
	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{writer},
		Tasks:   []*gocrew.Task{task},
		Verbose: true,
	})

	fmt.Println("🚀 Starting GOCREW Streaming Demo (Tokens should appear one by one):")
	
	dashboard.Start("8081")
	fmt.Println("🖥️  Dashboard active at http://localhost:8081/web-ui - Watch the tokens stream!")

	_, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("\nError: %v\n", err)
	}
	fmt.Println("\n\n✅ Done. Keep the dashboard open to review the logs!")
	select {}
}
