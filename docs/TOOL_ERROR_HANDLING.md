# Tool Error Handling in Gocrewwai 🛡️

In production, tools often fail due to network timeouts, API rate limits, or invalid input from the LLM. Gocrewwai provides native, high-performance error handling and retry strategies to ensure your agentic workflows are resilient.

## 🏗️ Error Handling Principles

| Feature | Gocrewwai Implementation | Key Advantage |
| :--- | :--- | :--- |
| **Native Go Errors** | Standard `error` interface | Idiomatic, high-performance error propagation. |
| **Retry Logic** | `MaxRetryLimit` in Task | Autonomous loops to fix tool errors. |
| **Review Strategy** | `RequiresReview()` on Tool | Pauses for human sign-off on risky tool actions. |
| **Error Feedback** | Direct feedback loop to LLM | The agent sees the tool error and attempts to self-correct. |

## 🚀 Implementing Robust Tools

By implementing the `RequiresReview()` method on your custom tools, you can ensure that destructive or high-cost actions are never taken without human oversight:

```go
type DeleteFileTool struct {
    gocrew.BaseTool
}

func (t *DeleteFileTool) RequiresReview() bool {
    return true // Always pause for approval before deleting files!
}

func (t *DeleteFileTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
    // 1. Gocrewwai will automatically pause here.
    // 2. Once the user approves, this code will run.
    return "File deleted successfully.", nil
}
```

## 🧠 Handling Common Tool Failures

### 1. API Timeouts
Gocrewwai's engine respects context timeouts across all tool executions. If a tool exceeds its allocated time, the engine will trigger a retry (if configured) with an updated prompt to the agent:

```go
// Task configuration with retry tolerance
task := gocrew.NewTask(gocrew.TaskConfig{
    Description:    "Search the web for top stock prices.",
    MaxRetryLimit:  3, // Automatically retry tool failures 3 times
})
```

### 2. Invalid Input Correction
If an agent passes invalid arguments to a tool, Gocrewwai will return the exact error message to the agent. The agent will then analyze the error, adjust its reasoning, and try again with the corrected input.

## 📈 Comparison with LangChain Tool Error Handlers

| Feature | LangChain Error Handlers | Gocrewwai Strategy |
| :--- | :--- | :--- |
| **Definition** | Complex `on_tool_error` callbacks | Idiomatic, native Go error handling. |
| **Logic** | Scripted logic | Agent-driven self-correction. |
| **Execution** | Python-based | Highly concurrent, non-blocking Go. |

---

[Back to Observability Guide](./OBSERVABILITY.md) | [Next: Migration Guide](./MIGRATION.md)
