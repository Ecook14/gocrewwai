# Feature Deep Dive: Events ⚓📢⚡

Gocrewwai features a high-performance, asynchronous event system that allows you to hook into every aspect of an agent's lifecycle. Built with Go's native `chan` and `select` patterns, it ensures that your application stays responsive and informed during complex orchestrations.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai events support **Real-time Callbacks**, **WebSocket Streams**, and **OpenTelemetry** audit logs.

---

## 🏗️ Core Event Types

Gocrewwai categorizes events based on the level of orchestration:

| Category | Typical Events | Best Use Case |
| :--- | :--- | :--- |
| **Agent Events** | `Thought`, `Action`, `Observation` | Updating a TUI or dashboard terminal. |
| **Task Events** | `Assigned`, `Completed`, `Failed` | Logging progress and handling task retries. |
| **Crew Events** | `Started`, `Planning`, `Finished` | Tracking the overall mission lifecycle. |
| **Tool Events** | `Call`, `Response`, `Error` | Monitoring high-risk tool interactions. |

---

## 🚀 Hooking into Events (Elite Style)

Using the `gocrew` SDK, you can register global event handlers or individual step callbacks:

```go
agent := gocrew.NewAgent(gocrew.AgentConfig{
    Role: "Event-Aware Researcher",
    // 1. Step Callback (Fires on every thought/action)
    StepCallback: func(event *gocrew.StepEvent) {
        fmt.Printf("Agent Thought: %s\n", event.Thought)
    },
    // 2. Stream Callback (Fires for every new token)
    StepStreamCallback: func(token string) {
        fmt.Print(token) // Real-time token streaming
    },
})
```

## 📊 Event Persistence & Auditing

All events generated during a crew execution are automatically:
1. **Logged**: Available in the `Verbose` output and the **Dashboard**.
2. **Traced**: Linked directly to the parent task and agent spans in **OpenTelemetry**.
3. **Persisted**: (If enabled) Stored in the checkpoint database for historical audit logs.

---

[Back to MCP Guide](./mcp.md) | [Next: Telemetry](./telemetry.md)
