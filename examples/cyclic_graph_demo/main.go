package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := llm.NewOpenAIClient(apiKey)

	researcher := agents.NewAgent(
		"Researcher",
		"Find a unique fact about a random element in the periodic table.",
		"Science enthusiast",
		model,
	)

	verifier := agents.NewAgent(
		"Fact Verifier",
		"Verify if the fact is truly unique and surprising. If not, ask for a new one.",
		"Strict judge",
		model,
	)

	// Define Tasks
	task1 := &tasks.Task{
		Description: "Research a unique fact about a random element.",
		Agent:       researcher,
	}

	task2 := &tasks.Task{
		Description: "Verify the uniqueness of the fact. Output 'RETRY' if it's too common, or 'FINISH' if it's amazing.",
		Agent:       verifier,
		Dependencies: []*tasks.Task{task1},
	}

	// Elite: Cyclic Logic
	// If task2 returns 'RETRY', go back to task1
	task2.OutputCondition = func(result interface{}) string {
		out := fmt.Sprintf("%v", result)
		if strings.Contains(strings.ToUpper(out), "RETRY") {
			return "retry"
		}
		return "ok"
	}
	task2.NextPaths = map[string]*tasks.Task{
		"retry": task1,
	}
	task2.MaxCycles = 3 // Safety limit

	myCrew := crew.NewCrew(
		[]*agents.Agent{researcher, verifier},
		[]*tasks.Task{task1, task2},
		crew.WithProcess(crew.Graph), // Or StateMachine
		crew.WithVerbose(true),
	)

	fmt.Println("🚀 Starting Elite Cyclic Graph Demo...")
	fmt.Println("(The crew will loop if the fact isn't 'amazing' enough according to the verifier)")
	
	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nFinal Approved Fact: %s\n", result)
}
