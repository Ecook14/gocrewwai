# Tasks in Gocrewwai ⚓

Tasks set the specific work units for your agents. They define exactly what needs to be done, who will do it, and what the final output should look like.

## 🏗️ Task Anatomy

A task in Gocrewwai is defined by its description, agent assignment, and its output configuration.

| Field | Type | Description |
| :--- | :--- | :--- |
| **Description** | `string` | Detailed instructions for the task. |
| **ExpectedOutput** | `string` | What the final result should look like (e.g., "A 3-paragraph summary"). |
| **Agent** | `CoreAgent` | The agent assigned to perform the task. |
| **Context** | `[]*Task` | Other tasks whose results provide required context for this task. |
| **OutputJSON** | `interface{}` | An optional Go pointer (e.g., `&MyStruct{}`) that the agent will populate as structured JSON. |
| **OutputFile** | `string` | Path where the result will be saved (supports automatic directory creation). |
| **Markdown** | `bool` | If true, ensures the output is formatted as valid Markdown. |
| **MaxRetryLimit** | `int` | Number of autonomous retries if the task fails or validation fails. |

## 🚀 Creating a Task (Elite Style)

Using the `gocrew` SDK, you can create tasks using a declarative configuration:

```go
package main

import "github.com/Ecook14/gocrewwai/gocrew"

type SummaryResult struct {
    Topic   string `json:"topic"`
    Findings []string `json:"findings"`
}

func main() {
    researchTask := gocrew.NewTask(gocrew.TaskConfig{
        Description:    "Research the top 5 emerging AI trends in 2024.",
        ExpectedOutput: "A structured list of findings with specific topics.",
        Agent:          researcher,
        OutputJSON:     &SummaryResult{},
        MaxRetryLimit:  3,
    })
}
```

## 🧠 Advanced Task Features

### 1. Task Context & Chaining
Tasks can depend on the results of other tasks. By adding a task to the `Context` field, the agent will receive the results of that task as part of its prompt, enabling sophisticated multi-step workflows.

### 2. Structured Outputs
Gocrewwai excels at converting unstructured LLM responses into structured Go data. By providing a pointer to a struct in the `OutputJSON` field, the framework will automatically parse, validate, and retry until the output matches your schema.

### 3. Output Handlers
You can register custom callbacks like `OnSuccess` or `OnError` to trigger secondary logic once a task is complete.

---

[Back to Agents Guide](./AGENTS.md) | [Next: Crews Guide](./CREWS.md)
