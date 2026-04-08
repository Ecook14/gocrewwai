# Self-Correction & Reflection in Gocrewwai 🛡️

In high-stakes environments, agents must be able to evaluate their own work before returning a result. Gocrewwai provides native support for **Self-Critique** and **Reflection Loops**, similar to CrewAI and LangGraph patterns.

## 🏗️ Core Reflection Principles

| Feature | Gocrewwai Implementation | Key Advantage |
| :--- | :--- | :--- |
| **Self-Critique** | `SelfCritique: true` in Config | Automatic "Double Check" before task completion. |
| **Iterative Refinement** | `MaxRetryLimit` in Task | Autonomous loops to fix validation errors. |
| **Reflective Crew** | `Process: gocrew.Reflective` | Custom multi-agent loop for deep critique. |
| **Validation Guardrails** | Strong Go Types | Compiler-enforced schema validation for JSON outputs. |

## 🚀 Enabling Basic Self-Critique

Turning on self-critique is as simple as setting a flag in your `AgentConfig`. When enabled, the agent will internally review its draft against the original goal and task description:

```go
package main

import "github.com/Ecook14/gocrewwai/gocrew"

func main() {
    expert := gocrew.NewAgent(gocrew.AgentConfig{
        Role:         "Senior Architect",
        Goal:         "Design a scalable system architecture.",
        SelfCritique: true, // Enable autonomous reflection loop
        Verbose:      true,
    })
}
```

## 🧠 Advanced Reflection Patterns

### 1. The Reflective Crew (Peer Review)
For the highest quality results, use a `Reflective` process. This pattern involves:
1.  **Drafting**: A "Primary Agent" generates the initial result.
2.  **Reviewing**: A "Critic Agent" reviews the draft and provides feedback.
3.  **Iterating**: The Primary Agent incorporates the feedback and produces the final version.

```go
reflectiveCrew := gocrew.NewCrew(gocrew.CrewConfig{
    Agents:  []gocrew.CoreAgent{primary, critic},
    Tasks:   []*gocrew.Task{researchTask},
    Process: gocrew.Reflective,
})
```

## 🛡️ Comparison with CrewAI Self-Correction

| Feature | CrewAI Self-Correction | Gocrewwai Self-Critique |
| :--- | :--- | :--- |
| **Efficiency** | Multiple sequential calls | Internalized agent reflection step. |
| **Feedback** | String-based | Structured state merging with the Critic agent. |
| **Reliability** | Python-level retries | Native Go error handling and typed validation. |

---

[Back to HITL Guide](./HUMAN_IN_THE_LOOP.md) | [Next: Observability Guide](./features/telemetry.md)
