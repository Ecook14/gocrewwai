# Feature Deep Dive: Flows ⚓🌊

Flows in Gocrewwai represent the highest level of orchestration. They allow you to build complex state machines and multi-step workflows that can span across multiple crews, with native support for durable persistence and type-safe state management.

---

> [!IMPORTANT]
> **Status: v0.9.0 (Alpha → Beta).** Gocrewwai Flows 2.0 utilize Go Generics for **Typed State** management and support **Durable Checkpointing** to SQLite or Redis.

---

## 🏗️ The Typed Flow 

In Gocrewwai v0.9, flows are built using a revolutionary, typed approach that ensures the source of truth for your data is always consistent.

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
    flow := gocrew.NewTypedFlow(MyState{Topic: "AI Agents", IsDraft: true}).WithPersistence(
        "pr-review-1042",
        gocrew.NewJSONFilePersistence("./db/checkpoints"),
    )

    // 2. Add Processing Nodes
    flow.AddNode(func(ctx context.Context, s MyState) (MyState, error) {
        // Run a crew here and update state
        s.Result = "Research Completed"
        return s, nil
    })

    // 3. Kickoff Flow
    out, err := flow.Kickoff(ctx)
}
```

## 🧠 Flow 2.0 Feature Highlights

### 💾 1. Durable Persistence (Checkpoints)
Flows automatically save their state after every node execution using LangGraph-style checkpointing. If a process crashes or is interrupted, you can resume precisely from the last successful node using the flow ID.

```go
// Back a typed flow with file persistence under a stable flow ID.
flow := gocrew.NewTypedFlow(MyState{Topic: "AI Agents"}).WithPersistence(
    "pr-review-1042",
    gocrew.NewJSONFilePersistence("./db/checkpoints"),
)
out, err := flow.Kickoff(ctx)
```

### 👤 2. Human-in-the-Loop (HITL)
Pause execution for manual review with `AddHumanFeedback` (console prompt) or gate individual nodes on human approval. The engine halts at the node until input is provided; browser approve/reject via the Dashboard review endpoint (`POST /api/review`) is supported where the dashboard server runs.

```go
flow.AddHumanFeedback(flow.HumanFeedbackConfig{
    Message:        "Approve the research draft?",
    PossibleRoutes: []string{"approved", "rejected"},
    DefaultOutcome: "rejected",
})
```

### 📈 3. Graph-Based Routing (Multi-Crew)
Flows transcend the standard linear sequence. Use router nodes to determine the next execution step based on real-time state, effectively wiring multiple independent Crews together. On `*flow.Flow`, register conditional branches with `AddEventRouter`/`AddRouter`; on the `flows` DAG engine, use `NodeRouter` nodes (see [Flow vs Flows](./flow-vs-flows.md)).

```go
// flow.Flow: conditional branch via event router
flow.AddEventRouter(func(ctx context.Context, s flow.State) (string, error) {
    if done, _ := s["is_draft"].(bool); !done {
        return "publish_crew", nil // Route to the publisher crew
    }
    return "research_crew", nil    // Route back to research
})

// flows DAG engine: router node in a graph
node := &flows.FlowNode{
    ID:     "quality_gate",
    Type:   flows.NodeRouter,
    Router: func(s flows.State) string { return nextNodeID(s) },
}
```

---

[Back to Processes Guide](./processes.md) | [Next: LLMs](./llms.md)
