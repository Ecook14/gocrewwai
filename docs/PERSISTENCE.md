# Persistence & Checkpoints in Gocrewwai 💾

Gocrewwai provides robust, production-grade persistence for long-running AI workflows. By separating the **definition of state** from the **persistence of execution**, Gocrewwai allows you to pause, resume, and "time-travel" through your agentic flows.

## 🏗️ Core Persistence Concepts

Persistence in Gocrewwai is powered by **Stores**, which automatically save the flow's state at the end of every node execution.

| Feature | Gocrewwai Implementation | Key Advantage |
| :--- | :--- | :--- |
| **Checkpointer** | File/SQLite/Redis checkpoint stores | Model-agnostic state saving. |
| **Durable Execution** | Auto-save after every super-step | Resume precisely from the last successful node. |
| **Run Isolation** | Stable flow/session IDs per run | Manage 1000s of concurrent, isolated user sessions. |
| **Time-Travel** | Snapshot-based versioning | "Rewind" a flow to a previous state and retry. |

## 🚀 Implementing Persistence 

Using the `gocrew` SDK, you can enable persistence simply by providing a store instance to your flow:

```go
package main

import (
	"context"

	"github.com/Ecook14/gocrewwai/gocrew"
)

type MyState struct {
	Topic  string
	Result string
}

func main() {
	ctx := context.Background()

	// 1. Back the flow with file persistence under a stable flow ID
	flow := gocrew.NewTypedFlow(MyState{Topic: "AI Agents"}).WithPersistence(
		"agent-run-1042",
		gocrew.NewJSONFilePersistence("./db/checkpoints"),
	)

	// 2. Add a processing node
	flow.AddNode(func(ctx context.Context, s MyState) (MyState, error) {
		s.Result = "Research Completed"
		return s, nil
	})

	// 3. Execution (state persists under the flow ID)
	out, err := flow.Kickoff(ctx)
	_, _ = out, err
}
```
```

## 🧠 State Recovery & Resume

Typed flows persist under their flow ID: re-running `Kickoff` with the same ID resumes durable state. Crew executions checkpoint to SQLite, Redis, or files and resume via `LoadLatestCheckpoint`. The `flows.Engine.Resume(ctx, f)` entrypoint restarts a flow from its last known checkpoint when a checkpointer is configured.

## 📈 Comparison with LangGraph Checkpointers

| Feature | LangGraph Checkpointer | Gocrewwai Store |
| :--- | :--- | :--- |
| **Storage Type** | Key-Value based | Strongly-typed SQL/Redis based. |
| **Concurrency** | Shared lock | Multi-reader, single-writer (WAL mode). |
| **Deployment** | Python-specific | Single-binary Go deployment. |

---

[Back to index](./index.md) | [Next: Human-in-the-Loop](./HUMAN_IN_THE_LOOP.md)
