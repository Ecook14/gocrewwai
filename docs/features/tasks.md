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
| **ExpectedOutput** | `string` | Clear definition of what the final result should be. |
| **Agent** | `CoreAgent` | The agent assigned to perform this task. |
| **Context** | `[]*Task` | Prior tasks whose results provide required context. |
| **OutputJSON** | `interface{}` | Struct pointer for structured JSON extraction. |
| **OutputSchema** | `string` | Raw JSON schema for validation (Alternative to OutputJSON). |
| **OutputFile** | `string` | Path where the result will be auto-saved. |
| **CreateDirectory** | `bool` | Auto-create parent directories for the output file. |
| **HumanInput** | `bool` | Enables **HITL** (Human-in-the-Loop) approval/feedback loops. |
| **Guardrails** | `[]Guardrail` | Custom validation logic for task outputs. |
| **MaxRetries** | `int` | Retries for schema validation failures. |
| **Timeout** | `Duration` | Max execution time for the single task. |
| **NextPaths** | `map[string]*Task` | State machine transitions for cyclic/graph logic. |

---

## 🧠 Context Chaining & Piping

Gocrew tasks excel at building complex, multi-stage "Logic Chains." By providing a slice of tasks to the `Context` field, the framework will:

1. **Wait for Dependencies**: Ensure that dependencies are resolved before the current task begins.
2. **Inject History**: Append the results of the context tasks into the current task's prompt.

**Visualizing the Chain:**
```text
Task A (Research) ──┐
                    ▼
Task B (Scrape) ────┼──► Task D (Write Final Report)
                    ▲
Task C (Analyze) ───┘
```
In Gocrewwai, you simply set `Context: []*Task{TaskA, TaskB, TaskC}` inside `TaskD`.

---

## 🛡️ Strict JSON Outputs (`GetOutput[T]`)

Gocrewwai solves the "Unreliable LLM" problem by enforcing strictly-typed JSON outputs. If you provide a struct pointer to `OutputJSON`, the engine will inject the required JSON schema, validate the LLM's response, and autonomously retry if the schema is malformed.

**Extracting the Data:**
Once the crew has finished, you can extract the populated data using the generic `GetOutput[T]()` helper.

```go
// 1. Define your struct
type Employee struct {
    Name     string `json:"name"`
    Age      int    `json:"age"`
    IsRemote bool   `json:"is_remote"`
}

// 2. Configure the task
profileTask := gocrew.NewTask(gocrew.TaskConfig{
    Description: "Extract the employee's details from the provided bio.",
    Agent:       analyst,
    OutputJSON:  &Employee{},
})

// ... run the crew ...

// 3. Extract Safely (Type Assertion handled internally)
result, _ := myCrew.Kickoff(ctx)
emp := gocrew.GetOutput[Employee](result)

fmt.Printf("Parsed Name: %s\n", emp.Name)
```

---

## 🌅 Task Callbacks

You can register custom hooks to trigger logic based on task status:
- **`OnSuccess`**: Run logic after a task successfully completes.
- **`OnError`**: Handle failures (e.g., alert Slack or log to a database).

---

[Back to index](../index.md) | [Next: Crews](./crews.md)
