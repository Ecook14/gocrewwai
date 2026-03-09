# Feature Deep Dive: Gocrew Flows 🌊

Flows allow you to orchestrate complex, multi-crew workflows with ease. While a `Crew` manages agents, a **Flow** manages crews, routing, and state.

---

## 🏗️ The Flow Structure

A Flow is a state machine where each node can be a simple function, a task, or an entire `Crew`.

```go
f := gocrew.NewFlow(initialState)

// Define steps
f.AddNode("research", researchCrew)
f.AddNode("writing", writingCrew)

// Define connections
f.SetEntryPoint("research")
f.AddEdge("research", "writing")
```

### Key Components

- **State (`flow.State`)**: A thread-safe, generic map that holds the "truth" of the workflow as it progresses.
- **Routers**: Dynamic nodes that decide the next path based on the current state (e.g., "If risk score > 0.8, go to AlertNode, otherwise go to ProcessNode").
- **Persistence**: Flows can automatically save their state to a database (PostgreSQL, SQLite, etc.), allowing for long-running workflows that span days or weeks.

---

## 🚦 Typed Flows (`flow.TypedFlow`)

For maximum Go safety, you can use `TypedFlow`, which uses Go generics to ensure your state always matches your custom struct.

```go
type MyState struct {
    Query  string
    Result string
}

f := flow.NewTypedFlow[MyState](initialState)
```

---

## 🛡️ Human-in-the-Loop (HITL)

Flows natively support human feedback loops. You can insert "WaitNodes" that pause execution and wait for an external signal (from the Dashboard or an API call) before proceeding.

---

## 🔄 Parallel & Branching

Flows support complex topologies:
- **Parallel Nodes**: Run multiple crews simultaneously and wait for all to finish.
- **Conditional Branching**: Use `AddRouter` to create intelligent forks in your workflow.
- **Cycles**: Loop back to previous nodes for iterative refinement.

---
**Gocrew** - Multi-crew orchestration with industrial-grade reliability.
