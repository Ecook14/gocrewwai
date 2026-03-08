package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Ecook14/crewai-go/pkg/agents"
	"github.com/Ecook14/crewai-go/pkg/crew"
	"github.com/Ecook14/crewai-go/pkg/llm"
	"github.com/Ecook14/crewai-go/pkg/memory"
	"github.com/Ecook14/crewai-go/pkg/tasks"
	"github.com/Ecook14/crewai-go/pkg/tools"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set OPENAI_API_KEY")
		return
	}

	model := llm.NewOpenAIClient(apiKey)

	// 1. Setup Persistent Memory
	sqliteStore, err := memory.NewSQLiteStore("crew_memory.db")
	if err != nil {
		fmt.Printf("Failed to setup persistent memory: %v\n", err)
		return
	}
	defer sqliteStore.Close()


	// 2. Define Advanced Agents
	researcher := &agents.Agent{
		Role:             "Strategic Researcher",
		Goal:             "Deeply analyze market trends and provide data-driven insights.",
		Backstory:        "Expert in synthesis and trend forecasting with a decade of experience.",
		LLM:              model,
		Tools:            []tools.Tool{tools.NewSearchWebTool(), tools.NewCalculatorTool()},
		AllowDelegation:  true,
		Memory:           sqliteStore,
		Verbose:          true,
	}

	writer := &agents.Agent{
		Role:             "Technical Storyteller",
		Goal:             "Translate complex research into engaging, actionable content.",
		Backstory:        "Award-winning writer known for making technology relatable.",
		LLM:              model,
		Memory:           sqliteStore,
		Verbose:          true,
	}

	// 3. Define Parallel Tasks
	marketTask := &tasks.Task{
		Description: "Analyze the current 2024 GPU market trends and calculate the YoY growth of the top 3 players.",
		Agent:       researcher,
	}

	contentTask := &tasks.Task{
		Description: "Craft a technical summary of the GPU market for an executive audience.",
		Agent:       writer,
		Context:     []*tasks.Task{marketTask},
	}

	// 4. Assemble Advanced Crew
	execCrew := crew.Crew{
		Agents:  []*agents.Agent{researcher, writer},
		Tasks:   []*tasks.Task{marketTask, contentTask},
		Process: crew.Hierarchical, // Dynamic delegation via Manager
		Verbose: true,
	}

	// 5. Execution with Orchestration
	fmt.Println("🚀 ## Starting Advanced Level Crew Execution ##")
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
