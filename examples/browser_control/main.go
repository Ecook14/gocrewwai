package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/gocrew"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ Error: Please set OPENAI_API_KEY environment variable.")
		return
	}

	model := gocrew.NewOpenAI(apiKey, "gpt-4o")

	// 1. Browser Tool via SDK
	browser := gocrew.NewBrowserTool()

	// 2. Define Agent (Elite Style)
	browserAgent := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Web Automator",
		Goal:      "Browse the internet purposefully to extract specific information.",
		Backstory: "An advanced agent capable of real-time multi-modal web navigation and element interaction.",
		LLM:       model,
		Tools:     []gocrew.Tool{browser},
		Verbose:   true,
	})

	// 3. Define Task (Elite Style)
	task := gocrew.NewTask(gocrew.TaskConfig{
		Description: "Navigate to 'https://news.ycombinator.com', find the top story, and return its title.",
		Agent:       browserAgent,
	})

	// 4. Assemble and Kickoff the Crew (Elite Style)
	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{browserAgent},
		Tasks:   []*gocrew.Task{task},
		Verbose: true,
	})

	fmt.Println("🚀 Starting GOCREW Browser Control Demo (Elite Style)...")
	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\n✨ Browser Result Summary:\n%s\n", result)
}
