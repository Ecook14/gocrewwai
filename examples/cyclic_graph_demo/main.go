package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Ecook14/gocrewwai/gocrew"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := gocrew.NewOpenAI(apiKey, "gpt-4o")

	researcher := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Researcher",
		Goal:      "Find a unique fact about a random element in the periodic table.",
		Backstory: "Science enthusiast",
		LLM:       model,
	})

	verifier := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Fact Verifier",
		Goal:      "Verify if the fact is truly unique and surprising. If not, ask for a new one.",
		Backstory: "Strict judge",
		LLM:       model,
	})

	task1 := &gocrew.Task{
		Description: "Research a unique fact about a random element.",
		Agent:       researcher,
	}

	task2 := &gocrew.Task{
		Description:  "Verify the uniqueness of the fact. Output 'RETRY' if it's too common, or 'FINISH' if it's amazing.",
		Agent:        verifier,
		Dependencies: []*gocrew.Task{task1},
	}

	task2.OutputCondition = func(result interface{}) string {
		out := fmt.Sprintf("%v", result)
		if strings.Contains(strings.ToUpper(out), "RETRY") {
			return "retry"
		}
		return "ok"
	}
	task2.NextPaths = map[string]*gocrew.Task{
		"retry": task1,
	}
	task2.MaxCycles = 3

	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{researcher, verifier},
		Tasks:   []*gocrew.Task{task1, task2},
		Verbose: true,
	})

	fmt.Println("🚀 Starting Elite Cyclic Graph Demo...")
	fmt.Println("(The crew will loop if the fact isn't 'amazing' enough according to the verifier)")

	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nFinal Approved Fact: %s\n", result)
}
