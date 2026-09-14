package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/gocrew"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := gocrew.NewOpenAI(apiKey, "gpt-4o")

	writer := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Technical Writer",
		Goal:      "Explain cloud computing to a 5-year-old.",
		Backstory: "Patient and clear teacher",
		LLM:       model,
	})

	task := &gocrew.Task{
		Description: "Explain 'serverless' using a lemonade stand analogy.",
		Agent:       writer,
	}

	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{writer},
		Tasks:   []*gocrew.Task{task},
		Verbose: true,
	})

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
