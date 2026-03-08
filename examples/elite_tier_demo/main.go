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
	"github.com/Ecook14/gocrewwai/pkg/tools"
	"strings"
)

func main() {
	ctx := context.Background()
	apiKey := os.Getenv("OPENAI_API_KEY")
	var model llm.Client

	if strings.HasPrefix(apiKey, "sk-or-") {
		fmt.Println("🌐 OpenRouter Key Detected. Switching to OpenRouter Client...")
		model = llm.NewOpenRouterClient(apiKey, "meta-llama/llama-3.1-8b-instruct:free")
	} else {
		if apiKey == "" {
			fmt.Println("⚠️  OPENAI_API_KEY is not set. Please export it before running the demo.")
			fmt.Println("Example: export OPENAI_API_KEY=sk-...")
		}
		model = llm.NewOpenAIClient(apiKey)
	}

	// 1. Setup Memories
	factStore := memory.NewInMemEntityStore()
	longTerm := memory.NewInMemCosineStore()

	// 2. Build Agents using the new Fluent Builder
	researcher := agents.NewAgentBuilder().
		Role("Elite Researcher").
		Goal("Discover hidden gems in the AI landscape.").
		LLM(model).
		EntityMemory(factStore).
		Memory(longTerm).
		Verbose(true).
		Build()

	writer := agents.NewAgentBuilder().
		Role("Technical Blogger").
		Goal("Synthesize research into viral content.").
		LLM(model).
		Tools(tools.NewFileWriteTool()).
		Verbose(true).
		Build()

	// 3. Define Tasks using the Task Builder
	researchTask := tasks.NewTaskBuilder().
		Description("Identify the top 3 trends in multi-agent orchestration for 2026.").
		Agent(researcher).
		OutputFile("research_results.md").
		Build()

	writingTask := tasks.NewTaskBuilder().
		Description("Write a 500-word blog post based on the research. Save the result as 'blog_post.md' using your tool.").
		Agent(writer).
		Context(researchTask).
		OutputFile("blog_post_final.md").
		Build()

	// 4. Create an Event-Driven Flow
	f := flow.NewFlow(nil)

	// Define a node that runs a hierarchical crew
	f.On("research_started", func(ctx context.Context, s flow.State) (flow.State, error) {
		c := crew.NewCrewBuilder().
			Agents(researcher, writer).
			Tasks(researchTask, writingTask).
			Process(crew.Hierarchical). // Auto-manager generation triggered here
			Verbose(true).
			Build()

		result, err := c.Kickoff(ctx)
		if err != nil {
			return nil, err
		}
		s["final_report"] = result
		return s, nil
	})

	// Add a final listener
	f.On("research_complete", func(ctx context.Context, s flow.State) (flow.State, error) {
		result := fmt.Sprintf("%v", s["final_report"])
		fmt.Printf("--- FINAL BLOG POST ---\n%s\n", result)
		return s, nil
	})

	// 5. Execute the Flow
	fmt.Println("🚀 Kicking off the Elite Tier Flow...")
	state, err := f.Start(ctx, "research_started")
	if err != nil {
		log.Fatalf("Flow failed: %v", err)
	}

	// Manually trigger the completion event (in a real flow, a node would Emit this)
	f.Emit(ctx, "research_complete", state)

	// 6. Inspect Entity Memory Facts discovered during the run
	fmt.Println("\n🧠 DISCOVERED ENTITIES:")
	entities, _ := factStore.Search(ctx, "", 10)
	for _, e := range entities {
		fmt.Printf("- %s\n", e.Text)
	}
}
