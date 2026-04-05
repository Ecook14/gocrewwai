package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/gocrew"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	exaKey := os.Getenv("EXA_API_KEY")
	if apiKey == "" || exaKey == "" {
		fmt.Println("❌ Error: Please set OPENAI_API_KEY and EXA_API_KEY environment variables.")
		return
	}

	model := gocrew.NewOpenAI(apiKey, "gpt-4o")

	// 1. Exa Search Tool via SDK
	exa := gocrew.NewExaTool(exaKey)

	// 2. Define Agent (Elite Style)
	researcher := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Exa Researcher",
		Goal:      "Find high-quality research papers or articles about LLM reasoning architectures.",
		Backstory: "Advanced AI analyst specializing in vector-indexed search.",
		LLM:       model,
		Tools:     []gocrew.Tool{exa},
		Verbose:   true,
	})

	// 3. Define Task (Elite Style)
	task := gocrew.NewTask(gocrew.TaskConfig{
		Description: "Use Exa search to find 3 groundbreaking papers on 'Chain of Thought' reasoning and summarize their URLs.",
		Agent:       researcher,
	})

	// 4. Assemble and Kickoff Crew (Elite Style)
	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{researcher},
		Tasks:   []*gocrew.Task{task},
		Verbose: true,
	})

	fmt.Println("🚀 Starting Exa AI Search Demo (Elite Style)...")
	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nFinal Report from Exa Search:\n%s\n", result)
}
