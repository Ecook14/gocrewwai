# Feature Deep Dive: Flows ⚓🌊

Flows in Gocrewwai represent the highest level of orchestration. They allow you to build complex state machines and multi-step workflows that can span across multiple crews, with native support for durable persistence and type-safe state management.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai Flows 2.0 utilize Go Generics for **Typed State** management and support **Durable Checkpointing** to SQLite or Redis.

---

## 🏗️ The Typed Flow (Elite Style)

In Gocrewwai v1.0, flows are built using a revolutionary, typed approach that ensures the source of truth for your data is always consistent.

```go
package main

import "github.com/Ecook14/gocrewwai/gocrew"

type MyState struct {
    Topic   string
    Result  string
    IsDraft bool
}

func main() {
    // 1. Initialize Flow with Initial State and Persistence
    flow := gocrew.NewTypedFlow(MyState{Topic: "AI Agents", IsDraft: true})
    flow.SetPersistence(sqliteStore)

    // 2. Add Processing Nodes
    flow.AddNode("research", func(ctx context.Context, s *MyState) error {
        // Run a crew here and update state
        s.Result = "Research Completed"
        return nil
    })

    // 3. Kickoff Flow
    flow.Start(ctx)
}
```

## 🧠 Flow 2.0 Feature Highlights

### 💾 1. Durable Persistence (Checkpoints)
Flows automatically save their state after every node execution using LangGraph-style checkpointing. If a process crashes or is interrupted, you can resume precisely from the last successful node using a `ThreadID`.

```go
// Intialize a Checkpointer backed by PostgreSQL or SQLite
ckpt := gocrew.NewSQLiteCheckpointer("./db/checkpoints.db")
flow.SetCheckpointer(ckpt)

// Run the flow specifying a ThreadID
flow.Start(ctx, gocrew.FlowConfig{ThreadID: "pr-review-1042"})
```

### 👤 2. Human-in-the-Loop (HITL)
Add `Interrupt` nodes to your flow to pause execution for manual review. The engine will halt until a payload is submitted via the REST API or Dashboard.

```go
flow.AddInterrupt("human_review", func(ctx context.Context, s *MyState, input interface{}) error {
    // Process input sent by the human
    userInput := input.(string)
    s.Result += "\nHuman Feedback: " + userInput
    return nil
})
```

### 📈 3. Graph-Based Routing (Multi-Crew)
Flows transcend the standard linear sequence. Use router nodes to determine the "Next" execution step based on real-time state, effectively wiring multiple independent Crews together.

```go
flow.AddRouter("quality_gate", func(ctx context.Context, s *MyState) (string, error) {
    if !s.IsDraft {
        return "publish_crew", nil // Route to the publisher crew
    }
    return "research_crew", nil    // Route back to research
})

// Wire the workflow edges
flow.AddEdge("research", "quality_gate")
flow.AddEdge("quality_gate", "publish_crew")
```

---

[Back to Processes Guide](./processes.md) | [Next: LLMs](./llms.md)
