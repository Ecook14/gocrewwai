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
	"github.com/Ecook14/gocrewwai/pkg/tools"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	exaKey := os.Getenv("EXA_API_KEY")
	model := llm.NewOpenAIClient(apiKey)

	// Exa Search Tool
	exa := tools.NewExaTool(exaKey)

	researcher := agents.NewAgent(
		"Exa Researcher",
		"Find high-quality research papers or articles about LLM reasoning architectures.",
		"Advanced AI analyst",
		model,
		agents.WithTools([]agents.Tool{exa}),
	)

	task := &tasks.Task{
		Description: "Use Exa search to find 3 groundbreaking papers on 'Chain of Thought' reasoning and summarize their URLs.",
		Agent:       researcher,
	}

	myCrew := crew.NewCrew(
		[]core.Agent{researcher},
		[]*tasks.Task{task},
		crew.WithVerbose(true),
	)

	fmt.Println("🚀 Starting Exa AI Search Demo...")
	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nFinal Report from Exa Search:\n%s\n", result)
}
