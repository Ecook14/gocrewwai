package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"c:/Users/nihar.bhayani/OneDrive/Documents/crewAI/Crew-GO/pkg/llm"
)

func main() {
	// 1. Verify Configuration & Registry
	model := "gpt-4o"
	client, err := llm.GetClient(model)
	if err != nil {
		fmt.Printf("Error getting client for %s: %v\n", model, err)
		os.Exit(1)
	}

	fmt.Printf("Registry resolved %s successfully.\n", model)

	// 2. Verify Generation (Dry Run / Mock)
	ctx := context.Background()
	messages := []llm.Message{
		{Role: "user", Content: "Hello Crew-GO!"},
	}
	options := llm.GenerateOptions{Model: model}

	fmt.Println("Attempting GenerateWithUsage...")
	resp, usage, err := client.GenerateWithUsage(ctx, messages, options)
	if err != nil {
		fmt.Printf("Generation failed: %v\n", err)
		// Note: This will fail if API keys are missing, which is expected in this env.
		// We are primarily checking the integration and variable binding.
	} else {
		fmt.Printf("Response: %s\n", resp)
		if usage != nil {
			fmt.Printf("Usage: %+v\n", usage)
		}
	}

	// 3. Verify Failover Logic (Mock Failure)
	fmt.Println("\nVerifying Failover Middleware...")
	primary := &mockFailClient{Err: fmt.Errorf("rate limit exceeded (mock)")}
	secondary := &mockSuccessClient{Response: "Failover Success!"}
	failover := llm.NewFailoverClient(primary, secondary, slog.Default())

	res, err := failover.Generate(ctx, messages, options)
	if err != nil {
		fmt.Printf("Failover failed: %v\n", err)
	} else {
		fmt.Printf("Failover Result: %s (Expected: Failover Success!)\n", res)
	}
}

// Mock Clients
type mockFailClient struct {
	llm.Client
	Err error
}
func (m *mockFailClient) Generate(ctx context.Context, messages []llm.Message, opts llm.GenerateOptions) (string, error) {
	return "", m.Err
}

type mockSuccessClient struct {
	llm.Client
	Response string
}
func (m *mockSuccessClient) Generate(ctx context.Context, messages []llm.Message, opts llm.GenerateOptions) (string, error) {
	return m.Response, nil
}
func (m *mockSuccessClient) GenerateWithUsage(ctx context.Context, messages []llm.Message, opts llm.GenerateOptions) (string, *llm.Usage, error) {
	return m.Response, &llm.Usage{Model: opts.Model}, nil
}
