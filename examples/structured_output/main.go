package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/gocrew"
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

	analyst := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Stock Analyst",
		Goal:      "Analyze stock prices and provide structured summaries.",
		Backstory: "A precision-focused financial analyst with deep expertise in market data.",
		LLM:       model,
		Verbose:   true,
	})

	task := gocrew.NewTask(gocrew.TaskConfig{
		Description: "Get the current stock price and a brief description for NVDA.",
		Agent:       analyst,
		OutputJSON:  &StockInfo{},
	})

	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{analyst},
		Tasks:   []*gocrew.Task{task},
		Verbose: true,
	})

	fmt.Println("🚀 Executing Structured Output Demo (Elite Style)...")
	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	fmt.Printf("\n--- 🏁 STRUCTURED OUTPUT RESULT ---\n%s\n", result)

	// Parse the JSON result into our struct
	var stock StockInfo
	if err := json.Unmarshal([]byte(fmt.Sprintf("%v", result)), &stock); err == nil {
		fmt.Printf("\n✅ Parsed Stock Info:\n")
		fmt.Printf("  Symbol:      %s\n", stock.Symbol)
		fmt.Printf("  Price:       %.2f %s\n", stock.Price, stock.Currency)
		fmt.Printf("  Description: %s\n", stock.Description)
	} else {
		fmt.Printf("\n⚠️  Could not parse result as StockInfo: %v\n", err)
	}
}
