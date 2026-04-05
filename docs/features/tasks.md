# Feature Deep Dive: Tasks ⚓📋

Tasks are the specific units of work that your agents must perform. In Gocrewwai, tasks are more than just string descriptions; they are highly structured objects that manage context, schemas, and output validation.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai tasks support advanced **Context Chaining** and **Strict JSON Output** validation out-of-the-box.

---

## 🏗️ The Task Config (Elite Style)

In Gocrewwai v1.0, tasks are constructed using the `TaskConfig` struct, providing a clean, declarative interface.

```go
summaryTask := gocrew.NewTask(gocrew.TaskConfig{
    Description:    "Identify the top 5 trends in AI agents for 2025.",
    ExpectedOutput: "A high-fidelity markdown report with specific bullet points.",
    Agent:          researcher,
    OutputJSON:     &TrendReport{}, // Gocrew will unmarshal and validate the LLM's raw output!
    MaxRetryLimit:  3,
})
```

### Key Parameters

| Parameter | Type | Description |
| :--- | :--- | :--- |
| **Description** | `string` | The detailed prompt/instructions for the task. |
| **Agent** | `CoreAgent` | The agent assigned to perform this task. |
| **Context** | `[]*Task` | A slice of prior tasks whose results provide required context for this task. |
| **OutputJSON** | `interface{}` | An optional Go pointer (e.g., `&MyStruct{}`) that the agent will populate as structured JSON. |
| **OutputFile** | `string` | Path where the result will be saved (supports automatic directory creation). |
| **Markdown** | `bool` | If true, ensures the output is formatted as valid Markdown. |
| **MaxRetryLimit** | `int` | Number of autonomous retries if the task fails or validation fails. |

---

## 🧠 Context Chaining & Piping

Gocrew tasks excel at building complex, multi-stage "Logic Chains." By providing a slice of tasks to the `Context` field, the framework will:

1. **Inject History**: Append the results of the context tasks into the current task's prompt.
2. **Handle Dependencies**: Ensure that dependencies are resolved before the current task begins (handled by the Crew/Flow engine).

---

## 🛡️ Strict JSON Outputs

Gocrewwai solves the "Unreliable LLM" problem by enforcing strictly-typed JSON outputs. If you provide a struct pointer to `OutputJSON`:

1. **Schema Injection**: Gocrew informs the LLM of the exact JSON schema required.
2. **Validation**: The engine attempts to unmarshal the response into your Go struct.
3. **Self-Correction**: If unmarshalling fails, the engine automatically triggers a retry, providing the LLM with the specific error and asking for a fix.

---

## 🌅 Task Callbacks

You can register custom hooks to trigger logic based on task status:
- **`OnSuccess`**: Run logic after a task successfully completes.
- **`OnError`**: Handle failures (e.g., alert Slack or log to a database).

---

[Back to index](../index.md) | [Next: Crews](./crews.md)
