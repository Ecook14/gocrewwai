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
Flows automatically save their state after every node execution. If a process crashes or is interrupted, you can resume precisely from the last successful node using a `ThreadID`.

### 👤 2. Human-in-the-Loop (HITL)
Add `HumanInterrupt` nodes to your flow to pause execution for manual review. This is essential for workflows that require expert sign-off on AI-generated content.

### 📈 3. Graph-Based Routing
Use router nodes to determine the "Next" execution step based on real-time state. This enables complex, non-linear workflows and cyclic reasoning loops.

### 📊 4. Real-Time Dashboard Integration
Every flow execution is visually tracked on the Glassmorphic Dashboard, allowing you to watch the state move through the nodes and inspect data at every step.

---

[Back to Processes Guide](./processes.md) | [Next: LLMs](./llms.md)
