# Feature Deep Dive: Agent Delegation ⚓🤝🏹

In Gocrewwai, delegation is a core mechanic that allows a lead agent to distribute its workload among its coworkers. This ensures that every sub-task is addressed by the agent with the most relevant expertise.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai agents support **Recursive Delegation** with built-in loop protection and task synthesis.

---

## 🏗️ Enabling Delegation (Elite Style)

In Gocrewwai v1.0, delegation is controlled by the `AllowDelegation` flag in your `AgentConfig`.

```go
leader := gocrew.NewAgent(gocrew.AgentConfig{
    Role:            "Project Manager",
    Goal:            "Oversee the entire AI project.",
    AllowDelegation: true, // Enable implicit coworker requests
})
```

### Explicit Task Delegation (A2A Network)
You can optionally configure explicit task assignment through the A2A sub-protocols. This is highly useful when coworkers live on remote cloud servers:

```go
subTask := leader.DelegateTo(
    "Security Analyst", 
    "Run a vulnerability scan on port 8080 and return the CVE report.",
)

// The engine blocks until the remote agent replies
result := subTask.Await()
```

## 🧠 The Delegation Lifecycle

1. **Identification**: The agent encounters a sub-task or information gap it cannot solve using its own tools.
2. **Co-worker Matching**: The agent scans the `Agents` list of the crew and finds the person whose `Goal` and `Backstory` most closely match the requirement.
3. **Task Issuance**: The lead agent issues a formal "Sub-task" to the coworker.
4. **Result Synthesis**: Once the coworker completes the work, the lead agent reviews the result and integrates it into its overall strategy.

## 📈 Benefits of Delegation

### 🏎️ 1. Parallel Task Execution
By delegating sub-tasks, multiple agents can work at the same time, significantly reducing the total time required to achieve a crew's mission.

### 🛡️ 2. Domain Expertise
Delegation ensures that specialized tasks (e.g., "Code Refactoring") are performed by specialized agents, leading to higher quality and more reliable output.

### 📊 3. Simplified Orchestration
As a developer, you only need to define high-level tasks for the lead agents; the agents will autonomously handle the "Who" and "How" of sub-delegation.

---

[Back to Collaboration Guide](./collaboration.md) | [Next: MCP Hub](./mcp.md)
