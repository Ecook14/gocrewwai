package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/crewai-go/pkg/agents"
	"github.com/Ecook14/crewai-go/pkg/config"
	"github.com/Ecook14/crewai-go/pkg/crew"
	"github.com/Ecook14/crewai-go/pkg/llm"
	"github.com/Ecook14/crewai-go/pkg/memory"
	"github.com/Ecook14/crewai-go/pkg/tasks"
	"github.com/Ecook14/crewai-go/pkg/telemetry"
	"github.com/Ecook14/crewai-go/pkg/tools"
)

func main() {
	// 1. Framework-Level Configuration
	// Developer can toggle Telemetry globally
	config.DefaultConfig.TelemetryEnabled = true
	
	if config.DefaultConfig.TelemetryEnabled {
		tp, _ := telemetry.InitTelemetry(os.Stdout)
		defer tp.Shutdown(context.Background())
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	model := llm.NewOpenAIClient(apiKey)

	// 2. Feature Toggles at Agent Level
	// Using Functional Options pattern for clean, "well thought" code.
	researcher := agents.NewAgent(
		"Researcher",
		"Find the latest AI trends",
		"Expert researcher",
		model,
		agents.WithMemory(memory.NewInMemCosineStore()),
		agents.WithSelfHealing(true),
		agents.WithMaxIterations(5),
	)

	// 3. Tool-Specific Configuration (e.g., Sandboxing)
	interpreter := tools.NewCodeInterpreterTool(
		tools.WithDocker("python:3.11-slim"),
		tools.WithLimits(1024, 2048), // 1GB Memory, 2k CPU Shares
	)
	researcher.Tools = []tools.Tool{interpreter}

	// 4. Crew-Level Configuration
	myCrew := crew.NewCrew(
		[]*agents.Agent{researcher},
		[]*tasks.Task{
			{Description: "Analyze the current state of Go for AI agents."},
		},
		crew.WithProcess(crew.Sequential),
		crew.WithVerbose(true),
	)

	// Kickoff
	fmt.Println("🚀 Kicking off with developer-defined configuration...")
	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nResult: %s\n", result)
}
