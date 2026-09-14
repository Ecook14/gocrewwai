package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Ecook14/gocrewwai/gocrew"
	"github.com/Ecook14/gocrewwai/pkg/memory"
	"github.com/Ecook14/gocrewwai/pkg/telemetry"
)

func main() {
	// 1. Initialize Advanced Observability
	tp, err := telemetry.InitTelemetry(telemetry.TelemetryConfig{Enabled: true, ServiceName: "gocrewwai-prod"})
	if err != nil {
		log.Fatalf("failed to init telemetry: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tp.Shutdown(ctx)
	}()

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set OPENAI_API_KEY environment variable")
		return
	}

	model := gocrew.NewOpenAI(apiKey, "gpt-4o")

	// 2. Setup Production Memory (Redis Backend)
	redisStore, err := memory.NewRedisStore([]string{"localhost:6379"}, "", 0, "prod_crew:")
	if err != nil {
		fmt.Printf("Redis not available, falling back to In-Memory: %v\n", err)
	}
	
	var store gocrew.MemoryStore = gocrew.NewInMemCosineStore()
	if redisStore != nil {
		store = redisStore
	}

	// 3. Create Production-Ready Agent
	agent := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Senior Analyst",
		Goal:      "Analyze market trends and provide actionable insights.",
		Backstory: "10 years of experience in financial analysis and market research.",
		LLM:       model,
		Memory:    store,
		Verbose:   true,
	})

	// 4. Define Tasks
	task1 := &gocrew.Task{
		Description: "Research the top 3 trends in AI for 2026.",
		Agent:       agent,
	}

	task2 := &gocrew.Task{
		Description: "Provide a summary of findings with actionable recommendations.",
		Agent:       agent,
		Context:     []*gocrew.Task{task1},
	}

	// 5. Assemble Crew
	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{agent},
		Tasks:   []*gocrew.Task{task1, task2},
		Verbose: true,
	})

	fmt.Println("🚀 Starting Production Usage Demo...")
	_, err = myCrew.Kickoff(context.Background())
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	fmt.Println("✅ Production demo completed successfully.")
}
