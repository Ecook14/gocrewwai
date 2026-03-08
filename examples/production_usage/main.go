package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/dashboard"
	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/memory"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"github.com/Ecook14/gocrewwai/pkg/telemetry"
	"github.com/Ecook14/gocrewwai/pkg/tools"
)

func main() {
	// 1. Initialize Advanced Observability (Stdout for demo)
	tp, err := telemetry.InitTelemetry(os.Stdout)
	if err != nil {
		log.Fatalf("failed to init telemetry: %v", err)
	}
	defer tp.Shutdown(context.Background())

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set OPENAI_API_KEY environment variable")
		return
	}

	model := llm.NewOpenAIClient(apiKey)

	// 2. Setup Production Memory (Redis Backend)
	// Assuming local Redis for this example
	redisStore, err := memory.NewRedisStore([]string{"localhost:6379"}, "", 0, "prod_crew:")
	if err != nil {
		fmt.Printf("Redis not available, falling back to In-Memory: %v\n", err)
	}
	
	var store memory.Store = memory.NewInMemCosineStore()
	if redisStore != nil {
		store = redisStore
	}

	// 3. Define Specialized Agents
	
	// A Researcher who uses Docker to run data analysis scripts securely
	interpreter := tools.NewCodeInterpreterTool()
	interpreter.UseDocker = true
	interpreter.Image = "python:3.11-slim"

	researcher := &agents.Agent{
		Role:             "Data Scientist",
		Goal:             "Perform secure data analysis and generate visualizations.",
		Backstory:        "Expert in Python and Docker environments.",
		LLM:              model,
		Tools:            []tools.Tool{interpreter},
		Memory:           store,
		AllowDelegation:  true,
	}

	// A Vision Analyst who can process images (Multimodal)
	visionAnalyst := &agents.Agent{
		Role:      "Vision Analyst",
		Goal:      "Analyze visual data and incorporate insights into reports.",
		Backstory: "Specializes in multimodal data synthesis.",
		LLM:       model, // Uses GPT-4o internally
	}

	// 4. Define Tasks
	task1 := &tasks.Task{
		Description: "Run a Python script to calculate the growth of AI agents and return a summary.",
		Agent:       researcher,
	}

	task2 := &tasks.Task{
		Description: "Analyze the provided image-based trend chart and merge it with the researched data.",
		Agent:       visionAnalyst,
	}

	// 5. Kickoff the Crew using CONSENSUAL process
	prodCrew := &crew.Crew{
		Agents:  []*agents.Agent{researcher, visionAnalyst},
		Tasks:   []*tasks.Task{task1, task2},
		Process: crew.Consensual,
		Verbose: true,
	}

	dashboard.Start("8081")
	fmt.Println("🖥️  Dashboard active at http://localhost:8081/web-ui - Watch production execution!")

	result, err := prodCrew.Kickoff(context.Background())
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	fmt.Printf("\n--- PRODUCTION CREW FINAL CONSENSUS ---\n%s\n", result)
	
	fmt.Println("✅ Demo finished. Keep the dashboard open to review the logs!")
	select {}
}
