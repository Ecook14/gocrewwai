package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/dashboard"
	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := llm.NewOpenAIClient(apiKey)

	thinker := agents.NewAgent(
		"Thinker",
		"Solve a complex math riddle slowly.",
		"Deep thinker",
		model,
		agents.WithVerbose(true),
	)

	// Initialize interrupt channel
	thinker.InterruptCh = make(chan string, 1)

	task := &tasks.Task{
		Description: "Solve the riddle: What is 1234 * 5678 but explain it like I'm five with many steps.",
		Agent:       thinker,
	}

	myCrew := crew.NewCrew(
		[]*agents.Agent{thinker},
		[]*tasks.Task{task},
	)

	fmt.Println("🚀 Starting Interrupt Demo...")
	
	// Start Dashboard to visualize the interrupt
	dashboard.Start("8081")
	fmt.Println("🖥️  Dashboard active at http://localhost:8081/web-ui - Open it now!")
	
	// Send an interrupt after 2 seconds
	go func() {
		time.Sleep(2 * time.Second)
		fmt.Println("\n⚠️ SENDING INTERRUPT: 'Actually, just give me the answer quickly, I'm in a hurry!'")
		thinker.InterruptCh <- "Actually, just give me the answer quickly, I'm in a hurry!"
	}()

	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Printf("\nFinal Response after Interrupt: %s\n", result)
	
	fmt.Println("✅ Demo finished. Keep the dashboard open to review the logs!")
	select {}
}
