# Persistence & Checkpoints in Gocrewwai 💾

Gocrewwai provides robust, production-grade persistence for long-running AI workflows. By separating the **definition of state** from the **persistence of execution**, Gocrewwai allows you to pause, resume, and "time-travel" through your agentic flows.

## 🏗️ Core Persistence Concepts

Persistence in Gocrewwai is powered by **Stores**, which automatically save the flow's state at the end of every node execution.

| Feature | Gocrewwai Implementation | Key Advantage |
| :--- | :--- | :--- |
| **Checkpointer** | `persistence.Store` Interface | Model-agnostic state saving (SQLite, Redis, Postgres). |
| **Durable Execution** | Auto-save after every super-step | Resume precisely from the last successful node. |
| **Thread Isolation** | Unique `ThreadID` for each run | Manage 1000s of concurrent, isolated user sessions. |
| **Time-Travel** | Snapshot-based versioning | "Rewind" a flow to a previous state and retry. |

## 🚀 Implementing Persistence (Elite Style)

Using the `gocrew` SDK, you can enable persistence simply by providing a store instance to your flow:

```go
package main

import "github.com/Ecook14/gocrewwai/gocrew"

func main() {
    // 1. Initialize a Persistent Store
    sqliteStore, _ := gocrew.NewSQLiteStore("flow_checkpoints.db")
    defer sqliteStore.Close()

    // 2. Setup Flow with Persistence
    flow := gocrew.NewTypedFlow(MyState{Topic: "AI Agents"})
    flow.SetPersistence(sqliteStore) // That's it!

    // 3. Execution (Auto-checkpoints every node)
    flow.Start(ctx)
}
```

## 🧠 State Recovery & Resume

If your application crashes or you need to resume a flow after a human interrupt, simply provide the `ThreadID` and the store will automatically reload the latest state:

```go
// Resume from existing thread
err := flow.Resume(ctx, threadID)
```

## 📈 Comparison with LangGraph Checkpointers

| Feature | LangGraph Checkpointer | Gocrewwai Store |
| :--- | :--- | :--- |
| **Storage Type** | Key-Value based | Strongly-typed SQL/Redis based. |
| **Concurrency** | Shared lock | Multi-reader, single-writer (WAL mode). |
| **Deployment** | Python-specific | Single-binary Go deployment. |

---

[Back to index](./index.md) | [Next: Human-in-the-Loop](./HUMAN_IN_THE_LOOP.md)
