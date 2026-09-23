# Advanced Orchestration Tutorial 🔥

Ready to push **Gocrew** to its absolute limits? This guide covers industrial-grade features: parallel execution, cyclic graphs, structured outputs, and multi-crew flows.

---

## 1. Process Routing

The `Process` type on a `Crew` determines how tasks are orchestrated.

### Sequential (`gocrew.Sequential`)
Tasks run one after another. Context from previous tasks is automatically injected into the next.

### Hierarchical (`gocrew.Hierarchical`)
Tasks run in **parallel**. A hidden `ManagerAgent` dynamically assigns tasks to worker agents based on their goals and tools, then merges the final results.

### Consensual (`gocrew.Consensual`)
Every agent in the crew executes the *same* task. The manager then gathers all answers and forces the agents to debate until a consensus is reached.

---

## 2. Cyclic Graphs & State Machines

Gocrew supports **Directed Acyclic Graphs (DAGs)** that can contain **cycles**. This allows agents to loop back and retry work until a specific quality threshold is met.

```go
// 1. Define the Tasks
codeTask := gocrew.NewTask(gocrew.TaskConfig{
    Description: "Write a Go function for quicksort.",
    Build:       true,
})

testTask := gocrew.NewTask(gocrew.TaskConfig{
    Description: "Test the code. Output 'FAIL' if it breaks, 'PASS' otherwise.",
    Context:     []*gocrew.Task{codeTask},
    Build:       true,
})

// 1. Define the Cycle
testTask.NextPaths = map[string]*gocrew.Task{
    "retry":   codeTask,   // Loop back!
    "success": deployTask, // Move forward!
}

testTask.OutputCondition = func(result interface{}) string {
    if strings.Contains(fmt.Sprintf("%v", result), "FAIL") {
        return "retry"
    }
    return "success"
}
```

---

## 3. Structured Output (Go Generics)

Gocrew can unmarshal LLM responses directly into your Go structs with zero boilerplate.

```go
type Analysis struct {
    Ticker string `json:"ticker"`
    Score  int    `json:"score"`
}

var result Analysis

task := gocrew.NewTask(gocrew.TaskConfig{
    Description: "Analyze AAPL earnings.",
    OutputJSON:  &result,
})

// After Kickoff:
fmt.Println(result.Score)
```

---

## 4. Multi-Crew Flows (`pkg/flow`)

Use `Flow` to connect multiple independent crews together into a stateful workflow.

```go
f := gocrew.NewFlow(initialState)

// Run crews in parallel
f.AddParallelNodes([]gocrew.Node{
    researchCrew,
    financialCrew,
})

// Route based on state
f.AddRouter(&gocrew.RouterNode{
    Routes: []gocrew.Route{
        {
            Name: "alert",
            Pred: func(s gocrew.State) bool { return s["risk"].(float64) > 0.8 },
            Node: alertCrew,
        },
    },
})
```

---

## 5. Creator Studio

Launch the dashboard with: `go run cmd/server/main.go --api-port 8080 --web`

---
**Gocrew** - Mastery in agentic orchestration.
