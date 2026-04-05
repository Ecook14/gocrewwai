# Flows in Gocrewwai 🌊

Flows are the highest level of orchestration in Gocrewwai, allowing you to build complex state machines and multi-step workflows that can span across multiple crews. Flows are designed to move from simple linear scripts to robust, long-running AI applications.

## 🏗️ Core Flow Concepts

A Flow in Gocrewwai is built using three primary primitives:

| Component | Description |
| :--- | :--- |
| **State** | A persistent, typed Go struct that holds the data shared across the workflow. |
| **Nodes** | Functions that represent a step in the workflow (e.g., `GenerateDraft`, `ReviewContent`). |
| **Routers** | Special nodes that determine the next execution step based on the current state. |

## 🚀 Creating a Flow (Elite Style)

Gocrewwai uses a revolutionary, typed approach for flow management through the `TypedFlow` SDK:

```go
package main

import "github.com/Ecook14/gocrewwai/gocrew"

type MyState struct {
    Topic   string
    Result  string
    IsDraft bool
}

func main() {
    // 1. Initialize Flow with Initial State
    flow := gocrew.NewTypedFlow(MyState{Topic: "AI Agents", IsDraft: true})

    // 2. Add Nodes
    flow.AddNode("research", func(ctx context.Background, s *MyState) error {
        // Run a crew here and update state
        s.Result = "Research Completed"
        return nil
    })

    // 3. Kickoff Flow
    flow.Start(ctx)
}
```

## 🧠 Advanced Flow Orchestration

### 1. Multi-Step Chaining
Flows allow you to chain multiple crews together, with the output of one crew serving as the state input for the next.

### 2. State Persistence & Checkpointing
Flows can be checkpointed and restored, allowing for long-running workflows that can survive application restarts or handle **Human-in-the-Loop** interrupts.

### 3. Visual Debugging
All flows integrated with the Gocrewwai **Dashboard** are visually tracked in real-time, allowing you to watch the state move through the graph.

---

[Back to Memory Guide](./MEMORY.md) | [Next: Knowledge Guide](./KNOWLEDGE.md)
