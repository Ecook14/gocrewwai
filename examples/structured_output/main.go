package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/dashboard"
	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
)

type StockInfo struct {
	Symbol      string  `json:"symbol"`
	Price       float64 `json:"price"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := llm.NewOpenAIClient(apiKey)

	analyst := agents.NewAgent(
		"Stock Analyst",
		"Analyze stock prices",
		"Financial expert",
		model,
		agents.WithVerbose(true),
	)

	// Define a task with a specific OutputSchema
	task := &tasks.Task{
		Description:  "Get the current stock price and a brief description for NVDA.",
		Agent:        analyst,
		OutputJSON:    &StockInfo{},
		MaxRetries:   3,
	}

	myCrew := crew.NewCrew(
		[]core.Agent{analyst},
		[]*tasks.Task{task},
	)

	fmt.Println("🚀 Executing Structured Output Task...")
	
	dashboard.Start("8081")
	fmt.Println("🖥️  Dashboard active at http://localhost:8081/web-ui")

	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Result should be an instance of StockInfo
	if info, ok := result.(*StockInfo); ok {
		fmt.Printf("\nSuccessfully Parsed Structured Output:\n")
		fmt.Printf("Symbol: %s\nPrice: %.2f %s\nDesc: %s\n", info.Symbol, info.Price, info.Currency, info.Description)
	} else {
		fmt.Printf("\nRaw Result: %v\n", result)
	}

	fmt.Println("✅ Demo finished. Keep the dashboard open to review the logs!")
	select {}
}
