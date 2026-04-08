# Feature Deep Dive: LLM Providers ⚓🔮

Gocrewwai is model-agnostic, supporting a wide range of LLM providers through a unified `llm.Client` interface. This allows you to mix and match models within a single crew to optimize for cost, speed, or intelligence.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai includes native, high-performance connectors for every major industry provider with built-in retry and failover logic.

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

In Gocrewwai v1.0, LLM clients are initialized via the `gocrew` SDK and then passed to agents:

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
options := gocrew.LLMOptions{
    Model:       "gpt-4o-mini", // Regional/Specific model override
    Temperature: 0.3,           // Lower temperature for structured extraction
    MaxTokens:   2000,
    Stop:        []string{"\n\n"},
}
```

## 🛡️ Resilience & Reliability

### 🚦 Granular Model Routing
A sophisticated crew uses specific models for distinct tasks to balance capability and budget. Gocrewwai supports **Purpose-Driven Routing** maps. By supplying a `Routing` map to the `QueryEngine`, the system dynamically resolves the optimal model based on the active task's required capability.

```go
engine := gocrew.NewQueryEngine(gocrew.EngineConfig{
    DefaultModel: gocrew.NewAnthropic("api-key", "claude-3-haiku"),
    Routing: map[string]gocrew.LLMClient{
        "vision": gocrew.NewOpenAI("api-key", "gpt-4o"),
        "rag":    gocrew.NewPineconeStore("api-key", "index").GetEmbeddingModel(),
        "math":   gocrew.NewOpenAI("api-key", "o1-preview"),
    },
})
```
If a specific route isn't defined for a task requirement, the engine seamlessly fails over to the `DefaultModel`.

### 🔄 Recursive Retries & Fallbacks
The engine includes a native retry handler with exponential backoff. If an LLM is overloaded, Gocrewwai will automatically pause and retry.

For extreme reliability, you can define **Cross-Provider Fallbacks**. If Anthropic goes down completely, your agent can instantly switch to OpenAI without skipping a beat:

```go
claude := gocrew.NewAnthropic(os.Getenv("ANTHROPIC_KEY"), "claude-3.5-sonnet")
gpt4 := gocrew.NewOpenAI(os.Getenv("OPENAI_KEY"), "gpt-4o")

// If Claude returns 503 Service Unavailable, switch to GPT-4o
claude.SetFallback(gpt4)

expert := gocrew.NewAgent(gocrew.AgentConfig{
    Role: "Architect",
    LLM:  claude,
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
