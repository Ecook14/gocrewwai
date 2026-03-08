# Feature Deep Dive: Telemetry & Enterprise Observability 📊

Hey! Let's talk about knowing what your AI is actually doing.

In standard Python scripts, analyzing what an AI is thinking usually involves staring at a chaotic stream of `print()` statements in a black terminal window. If an agent takes 45 seconds to answer a question, you have no idea if it's stuck in a loop, waiting on a slow API, or doing deep math.

For enterprise software, that is completely unacceptable. That's why Gocrew is instrumented natively with a **Global Async Event Bus** and **OpenTelemetry (OTEL)**.

---

## 📡 The Two Pillars of Observability

### 1. The Global Event Bus (`telemetry.GlobalBus`)

We built a blazing-fast, concurrent Pub/Sub broker right into the heart of the engine.

As an Agent performs operations deep within the Go call stack, it fires non-blocking events natively. Every critical stage of execution emits a strongly-typed event:
- `EventAgentStarted`, `EventAgentThinking`, `EventAgentFinished`
- `EventToolStarted`, `EventToolFinished`
- `EventError` (Crucial for tracking self-healing loop failures)

Because we use isolated Go channels, this event publishing has negligible impact on execution latency natively.

**How to tap into the matrix:**
You can subscribe natively in Go to pipe these events into your own logging endpoints or Kafka streams.
```go
subID, eventChannel := telemetry.GlobalBus.Subscribe()
defer telemetry.GlobalBus.Unsubscribe(subID)

go func() {
    for event := range eventChannel {
        switch event.Type {
        case telemetry.EventToolStarted:
            fmt.Printf("[%s] is using the %s tool...\n", event.AgentRole, event.Payload["tool"])
        case telemetry.EventToolFinished:
            fmt.Printf("Tool finished in %v seconds.\n", event.Payload["duration"])
        }
    }
}()
```

### 2. The Live Glassmorphic Dashboard

Using the `--ui` terminal flag bridges this internal Event Bus directly to a WebSocket server (`pkg/dashboard/dashboard.go`). It pushes those exact OTEL traces and Event payloads to a beautiful, real-time React/VanillaJS dashboard out-of-the-box! No polling, just pure WebSocket speed.

---

## 📈 Metric Tracking & Token Cost Accounting

Running autonomous LLMs can get wildly expensive if you aren't paying attention. 

To solve this, every Agent and the overarching `Crew` object holds a `UsageMetrics` map. Every single API call made through the `llm.MiddlewareClient` computes the exact number of **Prompt Tokens** and **Completion Tokens** consumed. 

At the end of a `Crew` kickoff, a summation is provided natively, allowing strict SLA monitoring and cost accounting inside your enterprise infrastructure.

```go
// 1. Kickoff your massive crew
result, _ := myCrew.Kickoff(ctx)

// 2. See exactly how much it cost you!
fmt.Printf("Total Prompt Tokens Used: %d\n", myCrew.UsageMetrics["prompt_tokens"])
fmt.Printf("Total Completion Tokens Used: %d\n", myCrew.UsageMetrics["completion_tokens"])
```

---

## 🤝 Help Me Expand the Telemetry!

If you are an SRE or DevOps engineer who loves observability, **I desperately want your help!**

- Could we natively export these traces to **Datadog, Jaeger, or Honeycomb** using standard OTEL OTLP exporters?
- Should we add Prometheus metrics scraping for active Goroutine task counts?

If you want to help make Gocrew the most strictly monitored AI framework on the planet, hop into `pkg/telemetry/events.go` and submit a Pull Request! Let's build it together.
