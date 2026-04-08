# Feature Deep Dive: Crews ⚓⛴️

A Crew is the core orchestration unit in Gocrewwai. It represents a collaborative group of agents working together to solve a sequence of tasks using a specific execution strategy.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai crews support **Sequential**, **Hierarchical**, and **Graph-based** orchestration with native **OpenTelemetry** tracing.

---

## 🏗️ The Crew Config (Elite Style)

In Gocrewwai v1.0, crews are constructed using the `CrewConfig` struct, providing a clean, declarative interface.

```go
myCrew := gocrew.NewCrew(gocrew.CrewConfig{
    Agents:      []gocrew.CoreAgent{researcher, writer},
    Tasks:       []*gocrew.Task{researchTask, writingTask},
    Process:     gocrew.Sequential,
    ManagerLLM:  gpt4o, // Required only for Hierarchical orchestration
    Verbose:     true,
    Planning:    true,  // Enable pre-execution task planning
})

result, err := myCrew.Kickoff(ctx)
```

### Key Parameters

| Parameter | Type | Description |
| :--- | :--- | :--- |
| **Agents** | `[]CoreAgent` | The slice of agents participating in the crew. |
| **Tasks** | `[]*Task` | The sequence of tasks to be performed. |
| **Process** | `ProcessType` | Orchestration strategy (Sequential, Hierarchical, etc.). |
| **ManagerLLM** | `LLMClient` | LLM used for hierarchical delegation. |
| **ManagerAgent** | `CoreAgent` | Optional: Provide a custom agent to act as the manager. |
| **Verbose** | `bool` | Detailed logging of orchestration steps. |
| **MaxRPM** | `int` | Global rate limit across all agents. |
| **Planning** | `bool` | pre-execution task planning phase. |
| **PlanningLLM** | `LLMClient` | LLM specifically for the planning phase. |
| **StateFile** | `string` | Path for auto-checkpointing (Persistence). |
| **TrainingDir** | `string` | Directory for training iteration feedback. |
| **TestLLM** | `LLMClient` | Internal evaluation LLM for elite tier verification. |
| **TaskCooldown** | `Duration` | Delay between tasks to prevent bursty rate limits. |

### 🧠 The Planning Phase (`Planning: true`)

When `Planning` is enabled, Gocrewwai injects a preliminary stage before any tasks are executed. The `PlanningLLM` (or `ManagerLLM`) receives the entire `CrewConfig` and dynamically constructs an execution **Directed Acyclic Graph (DAG)**.

1. **Analysis**: The LLM evaluates the tasks and agent capabilities.
2. **Strategy**: It determines the optimal order of execution (potentially overriding your sequential array) to maximize parallelization if tasks aren't dependent.
3. **Execution**: The crew then executes according to this newly minted plan.

---

## 🧩 Orchestration Processes

### ⚖️ Sequential (Default)
Agents work in a pre-defined order, passing their results to the next task's assigned agent. This is best for linear workflows.

### 👑 Hierarchical
A **Manager Agent** is automatically created to orchestrate the team. The manager delegates tasks dynamically based on agent goals and reviews the output of each agent before moving on.

### 🤖 State Machine (Graph)
Utilizes a dynamic state machine to determine the next step based on real-time task results. This enables non-linear workflows and cyclic loops.

**Example Implementation:**
```go
// 1. Define nodes (Tasks)
generator := gocrew.NewTask(...)
reviewer := gocrew.NewTask(...)

// 2. Define the Router logic on the reviewer task
reviewer.OutputCondition = func(result interface{}) string {
    r := result.(*ReviewResult)
    if r.Passed {
        return "end"
    }
    return "retry"
}

// 3. Map the conditions to paths
reviewer.NextPaths = map[string]*gocrew.Task{
    "retry": generator, // Cycle back!
}

// 4. Run as Graph Process
myCrew := gocrew.NewCrew(gocrew.CrewConfig{
    Process: gocrew.Graph,
    Tasks:   []*gocrew.Task{generator, reviewer},
})
```

## 📊 Crew Observability

Every crew execution in Gocrewwai is automatically instrumented with **OpenTelemetry**. This includes:
1. **Trace Spans**: Every agent thought, tool execution, and task handoff is captured as a span.
2. **Token Metrics**: Total token usage and cost are aggregated across the entire crew run.
3. **Execution Metadata**: Detailed logs of inputs, outputs, and internal reasoning.

---

[Back to index](../index.md) | [Next: Processes](./processes.md)
