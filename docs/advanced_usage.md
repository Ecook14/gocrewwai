# Advanced Orchestration Tutorial 🔥

Hey there! If you've already completed the Quickstart and you're comfortable with basic `Sequential` crews, you're ready for the really fun stuff. 

This guide covers the most technically complex, industrial-grade features I've built into the Crew-GO engine: Parallelism, Cyclic Graphs, Structured Go Outputs, and Multi-Crew State Machines.

Let's push this framework to its absolute limits!

---

## 1. Top-Level Process Routing 

When you organize your `Crew`, the `Process` enum dictates exactly how the graph of tasks is resolved. This is where Go's concurrency shines.

### A. Sequential (`crew.Sequential`)
Tasks execute strictly in the order they were placed in the slice.
- **Use Case**: Simple, linear data pipelines. *E.g. Scrape a website -> Summarize the text -> Save to a database.*

### B. Hierarchical (`crew.Hierarchical`)
Task definitions execute **in true parallel**. But they aren't executed blindly!
1. The Crew spins up an invisible autonomous `ManagerAgent`.
2. The Manager looks at the parallel tasks and dynamically assigns them to the worker Agents in your slice based on their skills.
3. Once all parallel Go-routines finish, the Manager synthesizes all results into a single final master output.
- **Use Case**: Running massive, concurrent research sweeps (e.g. Scrape 10 websites at once) and merging the results effortlessly.

### C. Consensual (`crew.Consensual`)
Takes a single task and launches it across **every single Agent** concurrently. The ManagerAgent then reads all of their answers and forces them to debate until a consensus is reached.
- **Use Case**: Evaluating high-risk decisions where you want 5 different specialized LLM personas to vote on the best answer.

### D. Reflective (`crew.Reflective`)
Sequential execution, but after every single task, the ManagerAgent acts like a strict editor. It executes a rigorous review prompt against the output. If it rejects the output, the worker agent is forced to retry from scratch.

---

## 2. Cyclic Graphs & State Machines (Elite Mode)

Standard legacy frameworks use direct `A -> B -> C` flows. That's boring. Crew-GO supports **DAGs with Cycles**. This means agents can get stuck in autonomous feedback loops (rewinding state) until an exact condition is met.

### Enabling Graph Mode
```go
myCrew := crew.Crew{
    Agents:  agents,
    Tasks:   tasks,
    Process: crew.Graph, // Turn on the DAG Engine!
    MaxCycles: 50,       // Global infinite-loop protection
}
```

### Defining Cycles via `OutputCondition`
You bind a function to a Task that analyzes its output. Depending on the return string, the task seamlessly routes to a new path in the graph.

```go
codeTask := &tasks.Task{
    Description: "Write a sorting algorithm in Go.",
    Agent:       coder,
}

testTask := &tasks.Task{
    Description: "Test the provided code. If it fails, output 'FAIL'. If it passes, output 'PASS'.",
    Agent:       qaAgent,
    Dependencies: []*tasks.Task{codeTask}, // testTask always waits for codeTask
}

// Map the outcomes!
testTask.NextPaths = map[string]*tasks.Task{
    "retry":   codeTask,   // Cycle backwards in time!
    "success": deployTask, // Move forwards!
}

// Evaluate the precise outcome
testTask.OutputCondition = func(result interface{}) string {
    resStr := fmt.Sprintf("%v", result)
    if strings.Contains(resStr, "FAIL") {
        return "retry"
    }
    return "success"
}
```
If `testTask` fails, the Engine natively rewinds the state, marks `codeTask` as incomplete, and kicks off the execution loop again, feeding the test errors back to the coder!

---

## 3. Structural JSON Output Extraction (Go Generics!)

Gocrew achieves this entirely natively using Go compilation structs, without messy `interface{}` casting.

### How it works
You pass a pointer of a strict Go Struct to a `Task`. Crew-GO dynamically reads your `json` tags via reflection, builds a vast JSON-Schema definition, injects it into the LLM system prompt, forces the LLM into strict `JSON Mode`, and unmarshals the response back into your pointer safely!

```go
// 1. Define your Strict Go Schema
type FinancialReport struct {
    CompanyTicker string   `json:"company_ticker"`
    BullPoints    []string `json:"bull_points"`
    BearPoints    []string `json:"bear_points"`
    RiskScore     int      `json:"risk_score_1_to_10"`
}

var finalReport FinancialReport

// 2. Bind the pointer to the Task. MAGIC HAPPENS HERE.
analystTask := &tasks.Task{
    Description:  "Analyze AAPL's latest Q3 earnings.",
    Agent:        analyst,
    OutputSchema: &finalReport, 
}

myCrew.Kickoff(ctx)

// 3. Immediately use Native Go Structs securely!
// No type assertions needed.
fmt.Printf("Risk Score: %d\n", finalReport.RiskScore)
for _, point := range finalReport.BullPoints {
    fmt.Println("+", point)
}
```

---

## 4. Multi-Crew Orchestration (`pkg/flow`)

If a `crew.Crew` is a tiny microservice of agents, `flow.Flow` is the Kubernetes that connects them all together. Flows are reactive State Machines that pipe shared state dictionaries (`map[string]interface{}`) between entirely distinct crews or raw Go functions.

### Creating a Flow
```go
import "github.com/Ecook14/gocrewwai/pkg/flow"

// Initial State Dictionary
initialState := flow.State{"company": "OpenAI", "ticker": "MSFT"}
f := flow.NewFlow(initialState)
```

### Adding Parallel Nodes
Run multiple independent AI crews concurrently, modifying the shared state safely.
```go
f.AddParallelNodes([]flow.Node{
    fetchFinancialsCrew, // An entire crew
    fetchNewsCrew,       // Another entire crew
    fetchSocialSentimentNode, // Er... just a standard Go function!
})
```

### Adding Conditional Branches (Routers)
Evaluate the state and completely branch your application execution natively in Go.
```go
f.AddRouter(&flow.RouterNode{
    Routes: []flow.Route{
        {
            Name: "buy-stock",
            Pred: func(s flow.State) bool { return s["sentiment"] == "very_bullish" },
            Node: executeTradeCrew, // Kickoff a specialized trading Crew
        },
        {
            Name: "hold-position",
            Pred: func(s flow.State) bool { return s["sentiment"] == "neutral" },
            Node: logHoldFunction,
        },
    },
})
```

---

## 5. The Creator Studio (Prototyping Elite Crews)

While writing Go code gives you the ultimate control, I built the **Creator Studio** inside the Web UI to let you prototype complex agent meshes in seconds.
Using the `--ui` flag, you can:
- **Hot-Swap Models**: Change an agent from `gpt-4o` to `claude-3-opus` live to compare reasoning quality.
- **Stage Bridges**: Establish A2A bridges between local agents and remote MCP tools with zero code.

---

## Challenge: Help Me Expand This!

These features lay the groundwork for building essentially any complex reasoning software application imaginable. 

But I know we can push it further. Are you thinking about distributed Crew orchestration across multiple physical servers using gRPC? Or implementing parallel task mapping natively inside a single Agent node? 

**I strongly invite you to help me build it.** Hop into `pkg/crew/crew.go`, test your craziest ideas, and submit a Pull Request. Let's make this framework legendary!
