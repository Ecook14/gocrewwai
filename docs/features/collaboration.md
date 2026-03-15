# Feature Deep Dive: Collaboration & Delegation 🤝

Gocrew agents are not isolated entities. They are designed to work together, sharing information and delegating sub-tasks to coworkers to achieve complex goals.

---

## 🏗️ How Collaboration Works

Collaboration is managed by the `Crew` engine and the agents' internal `delegation` tools.

### 1. Coworker Awareness
When a crew is initialized, every agent is aware of its "coworkers" (the other agents in the crew). If `AllowDelegation` is enabled, the agent gains access to two special tools:
- **DelegateWorkTool**: Allows the agent to hand off a specific sub-task to a coworker.
- **AskQuestionTool**: Allows the agent to query a coworker for information without delegating the entire task.

### 2. Information Sharing
Agents share a unified event bus and can reference the outputs of previous tasks via the `Context` parameter. This ensures that every agent has the latest "truth" about the project state.

---

## 🎯 Task Delegation Logic

Delegation follows the **Request -> Execution -> Report** pattern:

1. **Strategic Choice**: An agent (or a Manager in Hierarchical mode) decides that a specific sub-task is better suited for a coworker's role and tools.
2. **The Handoff**: The agent calls the `DelegateWorkTool` with the coworker's name and the sub-task description.
3. **Execution**: The coworker executes the sub-task as if it were a primary mission.
4. **Integration**: The result is returned to the original agent, who integrates the findings into their own reasoning loop.

---

## 🚦 Constraints & Controls

To prevent infinite loops and runaway costs, delegation is strictly controlled:

- **AllowDelegation (`bool`)**: A simple flag on the `Agent` to enable/disable this feature.
- **MaxIterations**: Limits the total number of reasoning steps (including delegations).
- **Hierarchical Processes**: In this mode, delegation is managed by a centralized `ManagerAgent` for maximum efficiency and parallel execution.

---

## 🛠️ Code Example

Enabling delegation is a single flag in the `AgentBuilder`.

```go
agent := gocrew.NewAgentBuilder().
    Role("Lead Architect").
    AllowDelegation(true). // Enable collaboration!
    Build()
```

---

## ☁️ Distributed Collaboration (A2A Protocol)

As of v0.9, collaboration is no longer restricted to a single machine. The **A2A Protocol** allows Gocrew instances to dynamically discover and communicate with each other over the network.

### Hardening Features:
- **Zero-Trust Security**: Every inter-agent request is authenticated using Bearer tokens (`X-A2A-Auth`).
- **Distributed Resilience**: Exponential backoffs and **Circuit Breakers** prevent cascading failures. If a remote agent goes down, the orchestrator handles it gracefully.
- **mDNS Auto-Discovery**: Agents can advertise their capabilities and discover peers on the local network automatically.
- **Observability Hub**: OpenTelemetry trace propagation ensures that a task seamlessly tracked across multiple physical servers.

To use remote agents, simply wrap the connection in a `RemoteAgentAdapter` and inject it into your crew alongside your local agents. The `core.Agent` interface ensures they all play by the same rules.

---
**Gocrew** - Scaling intelligence through collaboration.
