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
		"Technical Writer",
		"Explain cloud computing to a 5-year-old.",
		"Patient and clear teacher",
		model,
	)

	task := &tasks.Task{
		Description: "Explain 'serverless' using a lemonade stand analogy.",
		Agent:       writer,
	}

	myCrew := crew.NewCrew(
		[]core.Agent{writer},
		[]*tasks.Task{task},
		crew.WithVerbose(true),
	)

	// 1. Training Mode
	fmt.Println("🎓 Entering Training Mode (Iterations: 1)...")
	err := myCrew.Train(context.Background(), 1, nil)
	if err != nil {
		fmt.Printf("Training Error: %v\n", err)
		return
	}

	// 2. Replay/Checkpoint Demo
	stateFile := "crew_checkpoint.json"
	myCrew.StateFile = stateFile
	fmt.Printf("\n📍 Saving state to %s and running...\n", stateFile)
	
	_, err = myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Kickoff Error: %v\n", err)
	}

	// Save final state
	myCrew.SaveState(stateFile, 0)
	fmt.Println("\n✅ Demo Complete. Check crew_checkpoint.json for state data.")
}
