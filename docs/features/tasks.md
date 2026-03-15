# Feature Deep Dive: Tasks 📋

Tasks are the fundamental units of work in Gocrew. They define exactly what needs to be accomplished and which agent is responsible for the mission.

---

## 🏗️ The Task Builder

Use the `TaskBuilder` (via the `gocrew` facade) to construct complex tasks with precise requirements.

```go
task := gocrew.NewTaskBuilder().
    Description("Research the current price of Bitcoin.").
    ExpectedOutput("A short summary with the current price in USD.").
    Agent(researcher).
    AsyncExecution(false).
    Build()
```

### Key Parameters

- **Description (`string`)**: The clear, actionable instruction for the agent.
- **Agent (`core.Agent`)**: The autonomous agent or remote adapter responsible for this task.
- **ExpectedOutput (`string`)**: A hint to the agent about the desired format (Markdown, JSON, etc.).
- **OutputJSON (`interface{}`)**: A pointer to a Go struct. The engine will force the LLM into JSON mode and unmarshal the result directly into your struct.
- **Context (`[]*Task`)**: Links this task to previous ones. The outputs of these tasks will be injected into this task's prompt as background context.
- **HumanInput (`bool`)**: Enables **Human-in-the-Loop (HITL)**. Execution will pause for manual approval or feedback before and after the task.
- **OutputFile (`string`)**: Automatically saves the final task result to the specified path.
- **Timeout (`time.Duration`)**: Enforces a strict time limit on the task's execution.

---

## 🛠️ Task Execution Lifecycle

When a task is executed by an agent, the following steps occur:

1. **Context Injection**: Outputs from dependency tasks are injected into the prompt.
2. **HITL Review (Pre)**: If `HumanInput` is true, the user is prompted for approval/feedback.
3. **Agent Action**: The agent runs its reasoning loop to complete the task.
4. **Validation**: If `OutputJSON` is provided, the engine validates the result against the schema.
5. **Guardrails**: Any task-level guardrails are executed to ensure safety/compliance.
6. **HITL Review (Post)**: The final result is presented for human sign-off.
7. **Persistence**: The result is saved to disk (if `OutputFile` is set) and passed to subsequent tasks.

---

## 🔗 Chaining Tasks

Gocrew makes it easy to build complex multi-agent pipelines by chaining tasks together.

```go
// Task A must finish before Task B starts
taskA := gocrew.NewTaskBuilder().Description("Fetch data").Agent(agentA).Build()
taskB := gocrew.NewTaskBuilder().Description("Analyze data").Agent(agentB).Context(taskA).Build()
```

---
**Gocrew** - Structured missions for autonomous agents.
