# Deep Dive: Using Gocrew 📖

Gocrew is designed for two types of users:
1. **Dynamic Operators**: Who use the **Web UI Dashboard** to monitor and manage agent fleets.
2. **Elite Architects**: Who build production-grade AI systems using our strictly-typed Go library.

---

## 1. Ergonomic SDK: The `gocrew` Facade

To simplify development, we provide a unified `gocrew` package that exports all core components through type aliases and ergonomic constructors.

```go
import "github.com/Ecook14/gocrewwai/gocrew"

// Build agents fluently
agent := gocrew.NewAgentBuilder().
    Role("Lead Strategist").
    Goal("Create a 5-step growth plan").
    LLM(model).
    Build()

// Define structured tasks
task := gocrew.NewTaskBuilder().
    Description("Draft the plan").
    Agent(agent).
    OutputJSON(&MyStruct{}). // Type-safe output extraction!
    Build()
```

---

## 2. Advanced Sandboxing & Code Execution

Gocrew allows agents to solve problems by writing and running code. Safety is our priority.

### E2B (Recommended for Production)
The E2B sandbox provides a remote, secure cloud environment for code execution.
```go
interpreter := tools.NewCodeInterpreterTool(
    tools.WithE2B("your-e2b-api-key"),
)
```

### Docker (Local Isolation)
Run code in ephemeral containers with strict resource limits.
```go
interpreter := tools.NewCodeInterpreterTool(
    tools.WithDocker("python:3.11-slim"),
    tools.WithLimits(512, 1024), // 512MB RAM, 1024 CPU Shares
)
```

---

## 3. Unified Memory: Recency, Relevance, Importance

Gocrew doesn't just store text; it uses a **Unified Memory** system to provide agents with the most relevant context.

- **SQLite**: Great for local persistence without infrastructure overhead.
- **Redis/Pinecone/Qdrant**: Distributed vector stores for production scaling.

```go
// Setup long-term memory with a vector store
store := memory.NewSQLiteStore("my_memory.db")
agent.Memory = store
```

### Entity Memory
The engine automatically extracts entities (people, places, concepts) and maintains a persistent "fact sheet" that agents can reference across tasks.

---

## 4. Human-in-the-Loop (HITL)

For sensitive actions, Gocrew allows you to insert a human approval step.
```go
agent := gocrew.NewAgentBuilder().
    // ...
    Guardrails(guardrails.NewHumanReviewGuardrail("RoleName", "ToolName")).
    Build()
```
When triggered, the execution pauses, and a notification appears on the **Dashboard (default: port 8080)**. You can approve, reject, or provide feedback for the agent to retry.

---

## 5. YAML Configuration

Manage complex crews without cluttering your Go files.

### `crew.yaml`
```yaml
llm:
  provider: "openai"
  model: "gpt-4o"

agents:
  - role: "Senior Researcher"
    goal: "Fact check all claims"
    tools: ["SearchWeb", "Arxiv"]

tasks:
  - name: "research_task"
    description: "Verify the safety profile of new energy tech."
    agent: "Senior Researcher"
```

### Loading in Go
```go
myCrew, _ := gocrew.LoadFromYAML("crew.yaml")
result, _ := myCrew.Kickoff(context.Background())
```

---

## 6. Real-Time Observability (Telemetry)

Gocrew emits over 40 types of lifecycle events. You can subscribe to the `GlobalBus` to pipe these into your own monitoring systems.

```go
subID, events := gocrew.GlobalBus.Subscribe()
go func() {
    for e := range events {
        fmt.Printf("Event: %s - %v\n", e.Type, e.Payload)
    }
}()
```

---

## 7. The Documentation Index 📚

For detailed implementation details on any specific feature, refer to our specialized guides:

- **Core**: [Agents](docs/features/agents.md) | [Tasks](docs/features/tasks.md) | [Crews](docs/features/crews.md) | [Tools](docs/features/tools.md) | [LLMs](docs/features/llms.md) | [Processes](docs/features/processes.md)
- **Intermediate**: [Collaboration](docs/features/collaboration.md) | [Memory](docs/features/memory.md) | [Knowledge](docs/features/knowledge.md) | [Planning](docs/features/planning.md) | [Flows](docs/features/flows.md) | [Files](docs/features/files.md)
- **Advanced**: [Reasoning](docs/features/reasoning.md) | [Training](docs/features/training.md) | [Testing](docs/features/testing.md) | [Events](docs/features/events.md) | [CLI](docs/features/cli.md) | [Production](docs/features/production.md)

---
**Gocrew** - Built for scale, reliability, and developer happiness.
