# Crew-GO Advanced Usage 🔥

While the Quickstart covers the basics, `Crew-GO` is built specifically to take advantage of Go's unique features compared to Python: specifically **Concurrency** and **Struct Typed Outputs**.

## 1. Structured JSON Output Extraction (Pydantic Equivalent)
In Python's CrewAI, you use `Pydantic BaseModels` to force the LLM to output specific schemas. In `Crew-GO`, this is elegantly handled using native Go structs mapped to `json` tags.

### Defining Your Schema
Define a Go struct for the data you want back:

```go
type ReviewSchema struct {
	Score  int      `json:"score"`
	Pros   []string `json:"pros"`
	Cons   []string `json:"cons"`
}
```

### Binding to Tasks
Pass a pointer of your schema struct into `Task.OutputPydan`. When the task completes, `Crew-GO` instructs the LLM network to respond purely in JSON format and natively unmarshals it into your memory pointer!

```go
validationStruct := &ReviewSchema{}

reviewTask := &tasks.Task{
    Description: "Review the provided code and provide a score, pros, and cons.",
    Agent:       reviewerAgent,
    OutputPydan: validationStruct, // << Force schema here
}

// ... Run your crew Kickoff()

// The result natively translates!
fmt.Printf("Final Score is: %d\n", validationStruct.Score)
```

## 2. Parallel Concurency (Hierarchical Mode / Async Tasks)

One of Go's massive benefits is lightweight Goroutines. `Crew-GO` leverages `sync.WaitGroup` to easily execute massive amounts of parallel tasks natively without the headaches of Python's `asyncio` loop.

### Parallel Process Engine
If you set the Crew Process to `crew.Hierarchical`, the engine will blast all mapped tasks out in parallel using WaitGroup worker pools.

```go
techCrew := crew.Crew{
    Agents:  []*agents.Agent{agentA, agentB, agentC},
    Tasks:   []*tasks.Task{task1, task2, task3},
    Process: crew.Hierarchical, // << All tasks launch in parallel!
}

// Kickoff returns an `[]interface{}` slice when in Hierarchical Mode
results, _ := techCrew.Kickoff(context.Background())
```

### Async Task Overrides
Even inside a `crew.Sequential` execution block, you can designate specific tasks to immediately return and detach into the background using `AsyncExecution: true`.

```go
backgroundTask := &tasks.Task{
    Description:    "Scrape massive amounts of web pages for an hour.",
    Agent:          scraperAgent,
    AsyncExecution: true, // Does not block the sequential pipeline!
}
```

## 3. Context Timeout Propagation

Because `Crew-GO` agents and crews take a native `context.Context` payload, you can prevent LLM hangs easily out-of-the-box. Both Tasks and the Crew Orchestrators constantly listen for `<-ctx.Done()`.

```go
```go
// Allow the crew 5 minutes MAXIMUM to finish all tasks
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

result, err := techCrew.Kickoff(ctx)
if err == context.Canceled {
    fmt.Println("Crew took too long! Aborting cleanly.")
}
```

## 4. YAML Configuration Loaders (`pkg/config`)

Instead of writing verbose Go structs, non-engineers can define Prompts natively in YAML exactly like the Python equivalent.

**config/agents.yaml**:
```yaml
designer:
  role: "Lead Software Designer"
  goal: "Architect scalable Go solutions based on requirements."
  backstory: "You are a senior engineer who favors interfaces."
```

**main.go**:
```go
agentsMap, err := config.LoadAgents("config/agents.yaml")
tasksMap, err := config.LoadTasks("config/tasks.yaml", agentsMap)

myCrew := crew.Crew{
    Agents: []*agents.Agent{agentsMap["designer"]},
    Tasks:  []*tasks.Task{tasksMap["design_task"]},
}
```

## 5. Event-Driven Flow State Machines (`pkg/flow`)

To connect multiple standalone `Crews` together using reactive state-machines, use the built in Go-Channel Router.

```go
f := flow.NewFlow(nil)

// Build a pipeline of multiple Crews
f.AddNode(func(ctx context.Context, state flow.State) (flow.State, error) {
    // Run Crew A...
    state["research_done"] = true
    return state, nil
})

f.AddNode(func(ctx context.Context, state flow.State) (flow.State, error) {
    if state["research_done"] == true {
        // Run Crew B!
    }
    return state, nil
})

finalState, _ := f.Kickoff(context.Background())
```
