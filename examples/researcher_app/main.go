package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"github.com/Ecook14/gocrewwai/pkg/tools"
)

// BlogSchema maps directly to how Pydantic BaseModels format JSON prompts.
type BlogSchema struct {
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
	Body  string   `json:"body"`
}

func main() {
	slog.Info("Starting the Researcher Crew Example...")

	// 1. Initialize our LLM Client (OpenAI mapped via environment variable)
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		slog.Warn("OPENAI_API_KEY is not set. Execution will fall back to mocked output boundaries.")
	}
	openaiClient := llm.NewOpenAIClient(apiKey)

	// 2. Initialize our Agents
	researcher := &agents.Agent{
		Role:      "Senior Research Analyst",
		Goal:      "Uncover cutting-edge developments in AI and data science",
		Backstory: "You work at a leading tech think tank. Your expertise lies in identifying emerging trends.",
		Verbose:   true,
		LLM:       openaiClient,
	}

	writer := &agents.Agent{
		Role:      "Tech Content Strategist",
		Goal:      "Craft compelling content on tech advancements",
		Backstory: "You are a renowned Content Strategist, known for your insightful and engaging articles.",
		Verbose:   true,
		LLM:       openaiClient,
	}

	// 3. Initialize tasks for the agents
	researchTask := &tasks.Task{
		Description: "Conduct a comprehensive analysis of the latest advancements in AI in 2026. Focus on key trends and breakthroughs.",
		Agent:       researcher,
		Tools:       []tools.Tool{tools.NewAskQuestionTool()}, // Let the researcher ask questions
	}

	writingSchema := &BlogSchema{}
	writingTask := &tasks.Task{
		Description: "Using the insights provided, develop an engaging blog post that highlights the most significant AI advancements.",
		Agent:       writer,
		OutputPydan: writingSchema, // Force strict extraction
	}

	// 4. Form the Crew and Kickoff the execution loop
	techCrew := crew.Crew{
		Process: crew.Sequential,
		Agents:  []core.Agent{researcher, writer},
		Tasks:   []*tasks.Task{researchTask, writingTask},
		Verbose: true,
	}

	slog.Info("Kicking off crew...")
	result, err := techCrew.Kickoff(context.Background())
	if err != nil {
		slog.Error("Crew execution failed", slog.Any("error", err))
		os.Exit(1)
	}

	fmt.Println("\n######################")
	fmt.Println("FINAL Output:")
	fmt.Println("######################")
	fmt.Printf("%v\n", result)
}
