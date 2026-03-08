# Crew-GO Deep Usage Guide 📖

This document provides extensive, technical details on utilizing the powerful modules embedded within the Crew-GO Elite framework.

---

## 1. The Tool Arsenal (24+ Native Tools)

Crew-GO agents use `Tools` to interact with the world. Tools are struct types that implement the `tools.Tool` interface.

### A. Code Execution & Sandboxing (Elite)

**`CodeInterpreterTool`**
Executes Python or Go code dynamically. In the Elite tier, you MUST use sandboxes to run untrusted agent code safely.

*   **Docker Configuration:**
    Requires a local Docker daemon. Spins up ephemeral pods for execution.
    ```go
    // Limit to 512MB RAM and use Python 3.11
    tool := tools.NewCodeInterpreterTool(
        tools.WithDocker("python:3.11-slim"),
        tools.WithLimits(512, 1024), 
    )
    ```

	```

*   **Local Process Configuration:**
    Executes code natively on the host machine. (Use only in completely trusted environments!)
    ```go
    tool := tools.NewCodeInterpreterTool()
    ```

**`WASMSandboxTool`**
Executes WebAssembly modules locally using `wazero`. Zero-dependency and extremely secure.
```go
tool := tools.NewWASMSandboxTool()
```

### B. Web Interaction & Scraping

**`BrowserTool` (Chromedp)**
Full browser automation. Useful for SPAs, React/Vue sites, and bypassing basic bot protections.
```go
tool := tools.NewBrowserTool() // Operates headlessly by default
```

**`ScrapeWebsiteTool` & `SearchWebTool`**
For fast, text-only extraction and Google search.
```go
searchDetails := tools.NewSearchWebTool() // Uses DuckDuckGo by default
serperSearch := tools.NewSerperTool(os.Getenv("SERPER_API_KEY")) // Google Maps/News
exaSearch := tools.NewExaTool(os.Getenv("EXA_API_KEY")) // Semantic neural search
```

### C. Developer & Social Integrations

**`GitHubTool`**
Native integration to read issues, pull requests, and file contents.
```go
github := tools.NewGitHubTool(os.Getenv("GITHUB_TOKEN"))
```

**`SlackTool`**
Allows agents to broadcast messages to channels or read history.
```go
slack := tools.NewSlackTool(os.Getenv("SLACK_BOT_TOKEN"))
```

---

## 2. Dynamic YAML Configuration (`pkg/config`)

Instead of hardcoding agents in `main.go`, define them dynamically. This separates Prompt Engineering from Go backend logic.

### Define `config/agents.yaml`
```yaml
DataAnalyst:
  role: "Lead SQL Data Analyst"
  goal: "Extract insights from the Postgres database."
  backstory: "A rigorous statistician who double-checks every JOIN statement."
  verbose: true
  sandbox: "docker" # Dictates how CodeInterpreter runs
  tools:
    - name: "PostgresTool"
      params:
        dsn: "postgres://user:pass@localhost:5432/db"
    - name: "CodeInterpreterTool"
      params:
        use_docker: true
        docker_image: "python:3.11-slim"
```

### Define `config/tasks.yaml`
```yaml
analyze_retention:
  description: "Calculate the 30-day user retention rate."
  agent: "DataAnalyst"
```

### Load in Go
```go
agentsMap, err := config.LoadAgents("config/agents.yaml")
tasksMap, err := config.LoadTasks("config/tasks.yaml", agentsMap)

myCrew := crew.Crew{
    Agents: []*agents.Agent{agentsMap["DataAnalyst"]},
    Tasks:  []*tasks.Task{tasksMap["analyze_retention"]},
}
```

---

## 3. Production Guardrails (`pkg/guardrails`)

Guardrails are imperative criteria that **must** pass before a Task is considered officially "Finished". If a guardrail fails, the Agent is fed the error and forced to try again.

### PII Redaction
Strips sensitive data from the final output payload. Great for compliance logging.
```go
redactor := guardrails.NewPIIRedactor()
agent.Guardrails = append(agent.Guardrails, redactor)
// Output: "Contact me at [EMAIL REDACTED] from [IP REDACTED]"
```

### Toxicity Filtering
Rejects LLM responses that contain abusive or unsafe language.
```go
toxicFilter := guardrails.NewToxicityFilter()
agent.Guardrails = append(agent.Guardrails, toxicFilter)
```

### LLM-in-the-Loop Review (HitL Simulation)
Forces an output to be judged by an entirely separate "Critic" Agent before passing.
```go
critic := agents.NewAgent("Reviewer", "Be extremely harsh on code quality", "...", llm)
reviewGuardrail := guardrails.NewLLMReview(critic)

coderAgent.Guardrails = append(coderAgent.Guardrails, reviewGuardrail)
```

---

## 4. Elite Orchestration & Reliability

### A. Strongly-Typed Output Extraction (Generics)
Avoid `interface{}` type assertions. Use the generic `tasks.GetOutput[T]` helper to safely extract structured results from a task.

```go
type AnalysisResult struct {
    Trends []string `json:"trends"`
    Score  int      `json:"score"`
}

// ... execute crew ...

// Securely extract the result into your struct pointer
result, err := tasks.GetOutput[AnalysisResult](marketTask)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Score: %d\n", result.Score)
```

### B. Concurrency & Rate Limit Resiliency
Crew-GO is built for production volume. It handles API rate limits (`429`) and server errors (`5xx`) automatically with an **Exponential Backoff** retry strategy.

You can also gate the maximum parallelism of your crew to avoid hitting provider limits:

```go
myCrew := crew.NewCrew(
    agents, 
    tasks, 
    crew.WithMaxConcurrency(10), // Limit to 10 simultaneous LLM calls
)
```

---

## 5. Agent Memory Architectures (`pkg/memory`)

Agents with Memory are significantly smarter. Crew-GO utilizes an advanced RAG (Retrieval-Augmented Generation) loop inside the agent.

1.  **Short-Term Memory**: Agents remember their previous tool invocations within the same task.
2.  **Long-Term Memory**: Persistent storage across application restarts.

### Backends Available:
*   **In-Memory**: `memory.NewInMemCosineStore()` (Great for testing)
*   **SQLite**: `memory.NewSQLiteStore("memory.db")` (Great for local single-binary apps)
*   **Redis**: `memory.NewRedisStore("localhost:6379", "")` (Required for distributed scale)
*   **ChromaDB**: `memory.NewChromaStore("http://localhost:8000")` (Best-in-class vector retrieval)

**Usage:**
```go
// Connect to Redis
store, _ := memory.NewRedisStore("redis:6379", "password")

agent := agents.NewAgent(...)
agent.Memory = store // The Agent now has infinite, persistent recall.
```

---

## 5. The Real-time Telemetry Event Bus

At the core of the Elite Dashboard is `pkg/telemetry`. Everything the engine does emits an `Event`.

You can tap into this Event pipeline natively to pipe monitoring to Datadog, Splunk, or custom loggers.

```go
subID, eventChannel := telemetry.GlobalBus.Subscribe()
defer telemetry.GlobalBus.Unsubscribe(subID)

go func() {
    for event := range eventChannel {
        switch event.Type {
        case telemetry.EventAgentThinking:
            fmt.Printf("Deep thought loop #%d...\n", event.Payload["iteration"])
        case telemetry.EventToolFinished:
            fmt.Printf("Tool %s finished in %s\n", event.Payload["tool"], event.Payload["duration"])
        }
    }
}()
```
