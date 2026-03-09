# Feature Deep Dive: Event Lifecycle 📡

Gocrew is built on a high-performance, asynchronous **Event Bus**. Every single action, thought, and tool call within the framework is broadcast as a typed event, enabling deep observability and real-time monitoring.

---

## 🏗️ The Global Event Bus

The `pkg/events` package provides a centralized message bus that you can subscribe to from anywhere in your application.

```go
events.GlobalBus.Subscribe(func(ev events.Event) {
    fmt.Printf("[%s] %s: %v\n", ev.Type, ev.Source, ev.Payload)
})
```

---

## 🚦 Common Event Types

Gocrew emits over 40 lifecycle events. Here are the most critical ones:

### Agent Events
- **AgentTaskStarted**: When an agent begins a new mission.
- **AgentThought**: A single reasoning step (the "inner monologue").
- **AgentToolCall**: When an agent decides to use a tool.
- **AgentReasoningStarted/Completed**: Tracks the Reflect -> Evaluate loop.

### Task & Crew Events
- **TaskCompleted**: Final result of a task is ready.
- **CrewKickoffStarted**: The entire orchestration begins.
- **CrewTaskParallelStarted**: Parallel execution branch starts in Hierarchical mode.

### System Events
- **LLMCallStarted/Completed**: Detailed timing and token usage for LLM requests.
- **KnowledgeIngestionCompleted**: Confirmation that data is vectorized and ready.
- **ErrorEvent**: Centralized bubbling of all non-fatal and fatal errors.

---

## 📊 Powering the Dashboard

The **Glassmorphic Dashboard** is a direct consumer of the Global Event Bus. By subscribing to the bus via WebSockets, the dashboard provides:
- **Real-time Thought Streams**: See what the agent is thinking *as it happens*.
- **Tool Traceability**: Watch tool inputs and outputs in the browser.
- **Progress Tracking**: Visual Gantt-style charts of task completion.

---

## 🛠️ Custom Observers

Because the bus is a simple subscription model, you can easily pipe Gocrew events into:
- **OpenTelemetry**: For distributed tracing.
- **Prometheus**: For monitoring token usage and latency.
- **Slack/Discord**: For real-time notifications of task completions or errors.

---
**Gocrew** - Total visibility through event-driven design.
