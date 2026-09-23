# Feature Deep Dive: LLM Providers ⚓🔮

Gocrewwai is model-agnostic, supporting a wide range of LLM providers through a unified `llm.Client` interface. This allows you to mix and match models within a single crew to optimize for cost, speed, or intelligence.

---

> [!IMPORTANT]
> **Status: v0.9.0 (Alpha → Beta).** Gocrewwai LLM providers include native connectors for OpenAI, Anthropic, Gemini, Groq, OpenRouter, Ollama, and Failover with built-in retry and failover logic.

---

## 🏗️ Supported Providers

| Provider | SDK Constructor | Key Notes |
| :--- | :--- | :--- |
| **OpenAI** | `gocrew.NewOpenAI` | Optimized for `gpt-4o` and `o1` series. |
| **Anthropic** | `gocrew.NewAnthropic` | High-fidelity reasoning with Claude 3.5. |
| **Google** | `gocrew.NewGemini` | Access Gemini 1.5 Pro (2M Context) and Flash. |
| **Groq** | `gocrew.NewGroq` | Blazing-fast inference for Llama 3 and Mixtral. |
| **OpenRouter** | `gocrew.NewOpenRouter` | Unified gateway to 100+ open-source models. |
| **Local / Custom** | `llm.NewCustomClient` | Connect to Ollama, vLLM, or any OpenAI-compatible API. |

## 🚀 Basic Configuration

In Gocrewwai v0.9, LLM clients are initialized via the `gocrew` SDK and then passed to agents:

```go
// 1. Initialize OpenAI
model := gocrew.NewOpenAI(os.Getenv("OPENAI_API_KEY"), "gpt-4o")

// 2. Pass to Agent
expert := gocrew.NewAgent(gocrew.AgentConfig{
    Role: "Architect",
    LLM:  model,
})
```

## 🧠 Advanced Generation Options

Every LLM call in Gocrewwai uses a strictly typed `llm.GenerateOptions` struct. You can override global defaults at the individual call level:

```go
options := llm.GenerateOptions{
    Model:       "gpt-4o-mini", // Regional/Specific model override
    Temperature: 0.3,           // Lower temperature for structured extraction
    MaxTokens:   2000,
    Stop:        []string{"\n\n"},
}
```

## 🛡️ Resilience & Reliability

### 🔄 Multi-Provider Patterns

Different tasks benefit from different models. Here's a pattern using separate models for planning and execution:

```go
plannerLLM := gocrew.NewAnthropic("api-key", "claude-3.5-sonnet")
executorLLM := gocrew.NewOpenAI("api-key", "gpt-4o-mini")

planner := gocrew.NewAgent(gocrew.AgentConfig{
    Role: "Mission Planner",
    LLM:  plannerLLM,
})

executor := gocrew.NewAgent(gocrew.AgentConfig{
    Role: "Quick Researcher",
    LLM:  executorLLM,
})
```
If a specific route isn't defined for a task requirement, the engine seamlessly fails over to the `DefaultModel`.

### 🔄 Retry with Backoff

The engine includes a native retry handler with exponential backoff. If an LLM is overloaded, Gocrewwai will automatically pause and retry.

For extreme reliability, you can use different providers for different task criticality levels:

```go
criticalLLM := gocrew.NewAnthropic("api-key", "claude-3.5-sonnet")
fastLLM := gocrew.NewOpenAI("api-key", "gpt-4o-mini")

criticalAgent := gocrew.NewAgent(gocrew.AgentConfig{
    Role: "Critical Decision Maker",
    LLM:  criticalLLM,
})

fastAgent := gocrew.NewAgent(gocrew.AgentConfig{
    Role: "Quick Researcher",
    LLM:  fastLLM,
})
```

### 💾 Async Caching
To reduce costs and latency, enable **LLM Caching**. Every request/response pair is hashed and stored in your preferred backend (File, Redis, SQLite).

```go
cache := gocrew.NewFileCache("./data/cache")
agent := gocrew.NewAgent(gocrew.AgentConfig{
    LLM:   model,
    Cache: cache,
})
```

---

[Back to index](../index.md) | [Next: Memory](./memory.md)
