# Feature Deep Dive: Crews 👥

A **Crew** represents a team of agents working together to accomplish a set of tasks. It is the top-level orchestrator that manages the flow of information and execution.

---

## 🏗️ The Crew Builder

Construct your team using the `CrewBuilder`.

```go
myCrew := gocrew.NewCrewBuilder().
    Agents(researcher, writer).
    Tasks(researchTask, writeTask).
    Process(gocrew.Sequential).
    Verbose(true).
    Build()
```

### Key Parameters

- **Agents (`[]*Agent`)**: The list of all "member" agents in the crew.
- **Tasks (`[]*Task`)**: The sequence or graph of tasks to be executed.
- **Process (`ProcessType`)**: The orchestration strategy (Sequential, Hierarchical, etc.).
- **Manager (`*Agent`)**: (Optional) A specific agent to act as the manager in Hierarchical processes.
- **StateFile (`string`)**: Persists the crew's state to disk, allowing for resumable executions.
- **StepCallback**: A global hook that fires after every agent action, perfect for real-time monitoring.

---

## 🚦 Execution Modes (Processes)

Gocrew supports several ways to organize your team's workflow:

1. **Sequential**: Tasks are executed one by one in the order they appear.
2. **Hierarchical**: Tasks run in parallel, managed by an automated (or custom) manager agent.
3. **Consensual**: Agents debate the task output until they reach a consensus.
4. **StateMachine**: Tasks are executed based on conditional logic and state transitions (DAGs with cycles).

---

## 📊 Telemetry & Observability

Crews are equipped with deep telemetry. Every action, thought, and tool call is captured as an **Event** (`pkg/events`).
- **Dashboard**: Start `dashboard.Start("8080")` to watch your crew work in real-time.
- **GlobalBus**: Subscribe to `gocrew.GlobalBus` to pipe crew events into your own monitoring stack.

---

## 💾 Saving Progress

By setting `StateFile`, you can ensure that your crew's progress is saved. If an execution is interrupted, Gocrew can reload the state and resume from the last completed task.

---
**Gocrew** - High-performance orchestration for agent fleets.
