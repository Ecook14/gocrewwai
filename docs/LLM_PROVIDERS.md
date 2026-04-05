# LLM Providers in Gocrewwai 🔮

Gocrewwai is model-agnostic, supporting a wide range of LLM providers through a unified `llm.Client` interface. This allows you to mix and match models within a single crew to optimize for cost, speed, or intelligence.

## 🏗️ Supported Providers

| Provider | SDK Constructor | Key Notes |
| :--- | :--- | :--- |
| **OpenAI** | `gocrew.NewOpenAI` | Supports `gpt-4o`, `gpt-3.5-turbo`. |
| **Anthropic** | `gocrew.NewAnthropic` | Optimized for long contexts and Claude 3. |
| **Google** | `gocrew.NewGemini` | Access Gemini 1.5 Pro and Flash. |
| **Groq** | `gocrew.NewGroq` | Superior speed for Llama 3 and Mixtral. |
| **OpenRouter** | `gocrew.NewOpenRouter` | Unified gateway to 100+ models. |

## 🚀 Basic Integration

```go
package main

import "github.com/Ecook14/gocrewwai/gocrew"

func main() {
    // 1. OpenAI Integration
    model := gocrew.NewOpenAI(os.Getenv("OPENAI_API_KEY"), "gpt-4o")

    // 2. Groq Integration (High Speed)
    fastModel := gocrew.NewGroq(os.Getenv("GROQ_API_KEY"), "llama3-70b-8192")
}
```

## 🧠 High-Performance Caching

Gocrewwai includes built-in, asynchronous caching to reduce costs and latency for repetitive tasks.

### 💾 File Caching
Ideal for local development and single-server deployments.

```go
cache := gocrew.NewFileCache("./data/llm_cache")
agent := gocrew.NewAgent(gocrew.AgentConfig{
    LLM:   model,
    Cache: cache,
})
```

### 🏎️ Redis Caching
Recommended for production environments requiring shared state across multiple agent nodes.

```go
cache, _ := gocrew.NewRedisCache("localhost:6379", "", 0, 24*time.Hour)
```

## 🛡️ Generate Options (Typed Config)

All generation calls in Gocrewwai use a strictly typed `LLMOptions` (alias for `llm.GenerateOptions`) struct to ensure reliability:

```go
options := gocrew.LLMOptions{
    Model:       "gpt-4o",
    Temperature: 0.7,
    MaxTokens:   4096,
    Stop:        []string{"\n\n"},
}
```

---

[Back to Crews Guide](./CREWS.md) | [Next: Tools Guide](./TOOLS.md)
