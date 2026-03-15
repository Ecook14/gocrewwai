package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"github.com/Ecook14/gocrewwai/pkg/tools"
)

// MarketReport is the structured JSON output we want our final agent to produce
type MarketReport struct {
	Trend     string   `json:"trend"`
	Keywords  []string `json:"keywords"`
	Summary   string   `json:"summary"`
}

func main() {
	slog.Info("Starting Advanced Crew-GO Integration Pipeline...")

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		slog.Warn("No OPENAI_API_KEY found. Reverting to mocked outputs.")
	}

	// 1. Initialize our bindings
	openaiClient := llm.NewOpenAIClient(apiKey)

	// 2. Initialize built-in tools
	scraperTool := tools.NewScrapeWebsiteTool()
	writerTool := tools.NewFileWriteTool()

	// 3. Define the Agents
	researcher := &agents.Agent{
		Role:      "Internet Data Scraper",
		Goal:      "Extract content from URLs accurately.",
		Backstory: "You are a specialized parser agent.",
		Verbose:   true,
		LLM:       openaiClient,
		Tools:     []tools.Tool{scraperTool},
	}

	analyst := &agents.Agent{
		Role:      "Financial Analyst",
		Goal:      "Synthesize raw data into a structured market report.",
		Backstory: "You identify trends and generate rigid JSON schemas.",
		Verbose:   true,
		LLM:       openaiClient,
		Tools:     []tools.Tool{writerTool},
	}

	// 4. Define the Tasks
	scrapeTask := &tasks.Task{
		Description: `Extract the textual content from 'https://example.com' using the Web Scraper tool.`,
		Agent:       researcher,
	}

	// We pass a pointer to our Go struct into the analyst's task to force structured extraction
	reportStructTemplate := &MarketReport{}

	analyzeTask := &tasks.Task{
		Description: `Analyze the extracted text from the previous task. Write a summary, identify 3 keywords, and provide the overarching trend. Output exactly in JSON format, and then save the JSON string to './market_report.json' using the File Write tool.`,
		Agent:       analyst,
		OutputPydan: reportStructTemplate,
	}

	// 5. Connect the Crew Orchestra (Sequential logic)
	marketPipeline := crew.Crew{
		Agents:  []core.Agent{researcher, analyst},
		Tasks:   []*tasks.Task{scrapeTask, analyzeTask},
		Process: crew.Sequential,
		Verbose: true,
	}

	// 6. Execute with a safety context timeout!
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	slog.Info("🚀 Kicking off Pipeline.")
	
	_, err := marketPipeline.Kickoff(ctx)
	if err != nil {
		slog.Error("Pipeline failed", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("✅ Pipeline fully complete.")
	fmt.Printf("\nExtracted Structured JSON into Go Pointer: %+v\n", reportStructTemplate)
}
