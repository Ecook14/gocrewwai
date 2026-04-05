package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Ecook14/gocrewwai/gocrew"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set OPENAI_API_KEY")
		return
	}

	model := gocrew.NewOpenAI(apiKey, "gpt-4o")

	// 1. Setup Persistent Memory via SDK
	sqliteStore, err := gocrew.NewSQLiteStore("crew_memory.db")
	if err != nil {
		fmt.Printf("Failed to setup persistent memory: %v\n", err)
		return
	}
	defer sqliteStore.Close()

	// 2. Define Advanced Agents (Elite Style)
	researcher := gocrew.NewAgent(gocrew.AgentConfig{
		Role:             "Strategic Researcher",
		Goal:             "Deeply analyze market trends and provide data-driven insights.",
		Backstory:        "Expert in synthesis and trend forecasting with a decade of experience.",
		LLM:              model,
		Tools:            []gocrew.Tool{gocrew.NewSearchWebTool(), gocrew.NewCalculatorTool()},
		AllowDelegation:  true,
		Memory:           sqliteStore,
		Verbose:          true,
	})

	writer := gocrew.NewAgent(gocrew.AgentConfig{
		Role:             "Technical Storyteller",
		Goal:             "Translate complex research into engaging, actionable content.",
		Backstory:        "Award-winning writer known for making technology relatable.",
		LLM:              model,
		Memory:           sqliteStore,
		Verbose:          true,
	})

	// 3. Define Parallel Tasks (Elite Style)
	marketTask := gocrew.NewTask(gocrew.TaskConfig{
		Description: "Analyze the current 2024 GPU market trends and calculate the YoY growth of the top 3 players.",
		Agent:       researcher,
	})

	contentTask := gocrew.NewTask(gocrew.TaskConfig{
		Description: "Craft a technical summary of the GPU market for an executive audience.",
		Agent:       writer,
		Context:     []*gocrew.Task{marketTask},
	})

	// 4. Assemble Advanced Crew (Elite Style)
	execCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{researcher, writer},
		Tasks:   []*gocrew.Task{marketTask, contentTask},
		Process: gocrew.Hierarchical, // Dynamic delegation via Manager
		ManagerLLM: model,
		Verbose: true,
	})

	// 5. Execution with Orchestration
	fmt.Println("🚀 ## Starting Advanced Level Crew Execution (Elite Style) ##")
	start := time.Now()
	ctx := context.WithValue(context.Background(), "timestamp", start.Unix())
	
	result, err := execCrew.Kickoff(ctx)
	if err != nil {
		fmt.Printf("❌ Execution failed: %v\n", err)
		return
	}

	fmt.Printf("\n✨ ## Final Orchestrated Result ##\n%v\n", result)
	fmt.Printf("\n⏱️  Duration: %v\n", time.Since(start))
}
