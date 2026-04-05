# Feature Deep Dive: Processes ⚓⚖️

Processes in Gocrewwai define the "Rules of Engagement" for your crew. They determine how tasks are distributed and how agents collaborate to achieve their goals.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai supports **Sequential**, **Hierarchical**, and **Consensus-based** processes with native **Human-in-the-Loop** interrupts.

---

## 🏗️ Sequential (Default)

The simplest and most common orchestration strategy.

1. **Order matters**: Tasks are executed in the exact order they appear in the `TaskConfig` slice.
2. **Result Piping**: The output of Task 1 is automatically provided as context to Task 2.
3. **Efficiency**: Best for linear workflows where each step depends on the previous one.

```go
process := gocrew.Sequential
```

---

## 👑 Hierarchical

Emulates a real-world organizational structure. In this mode, Gocrewwai automatically creates a **Manager Agent** to orchestrate the team.

1. **Delegation**: The manager analyzes the `Tasks` and `Agents` and decides who is best suited for each assignment.
2. **Review**: Once an agent completes a task, the manager reviews the result. If unsatisfactory, the manager asks the agent to refine it.
3. **Final Synthesis**: After all tasks are complete, the manager synthesizes the final result.

```go
process := gocrew.Hierarchical
managerLLM := gocrew.NewOpenAI(apiKey, "gpt-4o")
```

---

## 📈 Graph (State Machine)

The most advanced orchestration strategy, inspired by LangGraph. It allows for non-linear, dynamic workflows.

1.  **Nodes & Edges**: Define your workflow as a directed graph where each node is an agent/task and edges are the logic for determining the "Next" step.
2.  **Conditional Branching**: Move through the graph based on real-time task results (e.g., if "Validation" fails, go back to "Researcher").
3.  **Cyclic Loops**: Perfect for iterative refinement and recursive problem-solving.

---

## 📊 Process Observability

All orchestration processes in Gocrewwai are fully instrumented with **OpenTelemetry**. Every "Handoff" and "Review" is captured as a distinct span in your trace, allowing you to see exactly how your crew collaborated.

---

[Back to Crews Guide](./crews.md) | [Next: Flows](./flows.md)
