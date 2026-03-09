# Feature Deep Dive: Processes 🚦

Processes define the "orchestration logic" of your crew. They determine how tasks are distributed among agents and how the execution flow is managed.

---

## 🏗️ Supported Process Types

Gocrew supports several orchestration strategies to handle everything from simple linear pipelines to complex, autonomous debacles.

### 1. Sequential (`gocrew.Sequential`)
The simplest and most common process. Tasks are executed in the exact order they are defined in the `Tasks` slice.
- **Context Flow**: Output from Task N is automatically injected as context for Task N+1.
- **Best For**: Data processing pipelines, research-then-write workflows.

### 2. Hierarchical (`gocrew.Hierarchical`)
A manager-led approach where tasks can be executed in parallel.
- **The Manager**: You can provide a `ManagerAgent` or let Gocrew spin up an automated one.
- **Delegation**: The manager analyzes the tasks and delegates them to the workers best suited for the role.
- **Systematic Merging**: The manager merges parallel results into a final cohesive output.
- **Best For**: Large-scale research sweeps, complex project planning.

### 3. Consensual (`gocrew.Consensual`)
A democratic approach to problem-solving.
- **Voting**: Every agent in the crew works on the *same* task.
- **Debate**: The manager gathers all responses and facilitates a debate between the agents until a consensus is reached.
- **Best For**: High-stakes decision making, creative brainstorming.

### 4. StateMachine (`gocrew.StateMachine`) / Graph
The most advanced orchestration mode, allowing for non-linear flows and cycles.
- **Conditional Routing**: Tasks can route to different "Next" tasks based on their output.
- **Cycles**: Agents can loop back to previous tasks if criteria aren't met (e.g., "Code failed tests, try again").
- **Best For**: Software development (Write-Test-Fix loop), iterative design.

---

## 🛠️ Configuring the Process

You set the process type during crew construction.

```go
myCrew := gocrew.NewCrewBuilder().
    Agents(a, b).
    Tasks(t1, t2).
    Process(gocrew.Hierarchical). // Set the strategy here
    Build()
```

---

## 🔄 Custom Orchestration

Because Gocrew is built in Go, you can also build custom orchestration logic by using **Flows** (`pkg/flow`). Flows allow you to treat entire Crews as nodes in a larger state machine, enabling multi-crew coordination.

---
**Gocrew** - Intelligent orchestration for any workflow.
