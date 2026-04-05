# Human-in-the-Loop (HITL) in Gocrewwai 👤

In critical AI workflows, human oversight is essential. Gocrewwai provides native support for **Manual Interrupts**, allowing you to pause an agent's execution for review and approval.

## 🏗️ HITL Core Primitives

| Feature | Gocrewwai Implementation | Key Advantage |
| :--- | :--- | :--- |
| **Interrupt Nodes** | `HumanInterrupt` node type | Pauses the flow automatically for user input. |
| **Task Approvals** | `WithApproval` option in Task | Requires explicit manual sign-off before completion. |
| **State Injection** | Manual state updates | Humans can modify the agent's work before it continues. |
| **Dashboard Sync** | Real-time TUI/Web UI | Approve or reject actions directly from the dashboard. |

## 🚀 Implementing a Manual Approval Flow

Using the `gocrew` SDK, you can define "Checkpoints" in your flow where the system will pause for human review:

```go
package main

import "github.com/Ecook14/gocrewwai/gocrew"

func main() {
    flow := gocrew.NewTypedFlow(MyState{Topic: "AI Agents"})

    // 1. Add a normal processing node
    flow.AddNode("generate", generateDraft)

    // 2. Add a Human Interrupt node for review
    flow.AddInterrupt("review", func(ctx context.Context, s *MyState) (bool, error) {
        // This node will wait for an external 'Signal' from the UI
        return true, nil
    })
    
    // 3. Execution (Flow will pause at 'review')
    flow.Start(ctx)
}
```

## 🧠 Approving via the Dashboard

While the flow is paused, the Gocrewwai **Dashboard** will display an "Awaiting Approval" status. To continue:
1.  **Review**: Inspect the current state and agent logs.
2.  **Approve**: Send a signal to resume the flow.
3.  **Reject**: (Optional) Modify the state and signal a "retry" of the previous node.

## 🛡️ Comparison with LangGraph Interrupts

| Feature | LangGraph `interrupt_before` | Gocrewwai `HumanInterrupt` |
| :--- | :--- | :--- |
| **Placement** | Pre-node hook | Explicit, logic-driven node. |
| **Input** | Manual state update | Direct Signal injection from UI/CLI. |
| **UX** | CLI-centric | Real-time Dashboard integration. |

---

[Back to Persistence Guide](./PERSISTENCE.md) | [Next: Self-Correction Guide](./SELF_CORRECTION.md)
