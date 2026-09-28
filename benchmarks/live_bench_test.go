// Live-provider benchmarks (opt-in).
//
// These benchmarks hit real LLM endpoints and are SKIPPED unless
// GOCREW_BENCH_LIVE=1. Provider keys come from the standard env vars
// (OPENAI_API_KEY); Ollama runs against http://localhost:11434 and skips
// when unreachable. Never run in CI by default: they cost money/latency.
package bench

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/llm"
)

func liveEnabled(b *testing.B) {
	b.Helper()
	if os.Getenv("GOCREW_BENCH_LIVE") != "1" {
		b.Skip("live benchmarks disabled (set GOCREW_BENCH_LIVE=1)")
	}
}

func benchPrompt() ([]llm.Message, llm.GenerateOptions) {
	return []llm.Message{{Role: "user", Content: "Reply with exactly: ok"}},
		llm.GenerateOptions{Model: "", MaxTokens: 8}
}

// BenchmarkLiveOpenAI measures end-to-end Generate latency against OpenAI.
func BenchmarkLiveOpenAI(b *testing.B) {
	liveEnabled(b)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		b.Skip("OPENAI_API_KEY not set")
	}
	client := llm.NewOpenAIClient(key)
	msgs, opts := benchPrompt()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := client.Generate(ctx, msgs, opts); err != nil {
			b.Fatalf("generate: %v", err)
		}
	}
}

// BenchmarkLiveOllama measures end-to-end Generate latency against a local
// Ollama daemon. Skips when nothing listens on 127.0.0.1:11434.
func BenchmarkLiveOllama(b *testing.B) {
	liveEnabled(b)
	conn, err := net.DialTimeout("tcp", "127.0.0.1:11434", 2*time.Second)
	if err != nil {
		b.Skip("ollama not reachable at 127.0.0.1:11434")
	}
	conn.Close()
	client := llm.NewOllamaClient("llama3")
	msgs, opts := benchPrompt()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := client.Generate(ctx, msgs, opts); err != nil {
			b.Fatalf("generate: %v", err)
		}
	}
}
