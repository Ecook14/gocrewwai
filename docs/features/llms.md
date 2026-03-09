# Feature Deep Dive: LLMs 🧠

Gocrew is model-agnostic. We provide high-performance, strictly-typed clients for all major LLM providers, ensuring your agents have the best "brain" for the job.

---

## 🔌 Supported Providers

We ship with native support for:
- **OpenAI**: GPT-4o, o1-preview, etc.
- **Anthropic**: Claude 3.5 Sonnet/Opus.
- **Google**: Gemini 1.5 Pro/Flash.
- **Groq**: Llama 3, Mixtral (ultra-fast inference).
- **OpenRouter**: Access to 100+ models via a single API.

---

## 🏗️ Configuring an LLM

All LLM clients are initialized via the `gocrew` facade with consistent signatures.

```go
// Simple initialization
openai := gocrew.NewOpenAI(apiKey, "gpt-4o")

// Advanced options
anthropic := gocrew.NewAnthropic(apiKey, "claude-3-5-sonnet-20240620")
```

### Common Options (`llm.Options`)

You can fine-tune every request using our strongly-typed options:
- **Temperature**: Control creativity (0.0 to 1.0).
- **MaxTokens**: Limit the response length.
- **TopP / FrequencyPenalty**: Advanced sampling controls.
- **Seed**: For deterministic, reproducible outputs.

---

## 🛡️ Reliability & Resilience

Production AI apps can't break just because a provider is down. Our LLM core includes:
- **Automatic Retries**: Built-in exponential backoff for 429 (Rate Limit) and 5xx (Server Error) responses.
- **Strict Parsing**: We use strict JSON schemas to ensure the model's output always maps correctly to your Go types.
- **Token Tracking**: Real-time monitoring of prompt and completion tokens for cost management.

---

## 🛠️ Custom LLMs

Need to use a local model or a custom enterprise API? Gocrew uses a clean `llm.Client` interface. As long as your client implements the `Generate` and `Stream` methods, it can power any Gocrew agent.

---
**Gocrew** - Bringing any model to the Go ecosystem.
