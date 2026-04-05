package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/gocrew"
	"github.com/Ecook14/gocrewwai/pkg/dashboard"
)

type StockInfo struct {
	Symbol      string  `json:"symbol"`
	Price       float64 `json:"price"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ Error: Please set OPENAI_API_KEY environment variable.")
		return
	}

	model := gocrew.NewOpenAI(apiKey, "gpt-4o")

	// 1. Define Agent (Elite Style)
	analyst := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Stock Analyst",
		Goal:      "Analyze stock prices and provide structured summaries.",
		Backstory: "A precision-focused financial analyst with deep expertise in market data.",
		LLM:       model,
		Verbose:   true,
	})

	// 2. Define a task with a specific OutputJSON (Elite Style)
	task := gocrew.NewTask(gocrew.TaskConfig{
		Description:    "Get the current stock price and a brief description for NVDA.",
		Agent:          analyst,
		OutputJSON:     &StockInfo{},
		MaxRetryLimit:  3,
	})

	// 3. Assemble and Kickoff the Crew (Elite Style)
	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{analyst},
		Tasks:   []*gocrew.Task{task},
		Verbose: true,
	})

	fmt.Println("🚀 Executing GOCREW Structured Output Task (Elite Style)...")
	
	dashboard.Start("8081")
	fmt.Println("🖥️  Dashboard active at http://localhost:8081/web-ui")

	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// 4. Result should be an instance of StockInfo
	if info, ok := result.(*StockInfo); ok {
		fmt.Printf("\n✨ Successfully Parsed Structured Output:\n")
		fmt.Printf("Symbol: %s\nPrice: %.2f %s\nDesc: %s\n", info.Symbol, info.Price, info.Currency, info.Description)
	} else {
		fmt.Printf("\nRaw Result: %v\n", result)
	}

	fmt.Println("\n✅ Demo finished. Keep the dashboard open to review the logs!")
	select {}
}
