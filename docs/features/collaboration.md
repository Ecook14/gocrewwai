# Feature Deep Dive: Collaboration ⚓🤝

Gocrewwai is built on the principle that multi-agent collaboration is superior to a single monolithic agent. Our framework provides several native mechanisms for agents to work together, share information, and review each other's output.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai collaboration features include **Autonomous Delegation**, **Peer-Review Loops**, and **Shared Memory State**.

---

## 🏗️ Core Collaboration Patterns

### 1. Unified Delegation
By enabling `AllowDelegation: true` in an agent's config, you empower it to ask its coworkers for help. If an agent identifies a task it cannot solve alone, it will:
- **Analyze**: Determine which coworker has the required goal/backstory.
- **Request**: Send a sub-task to that coworker.
- **Synthesize**: Incorporate the coworker's result into its own final answer.

### 2. The Hierarchical Process
In **Hierarchical** mode, collaboration is managed by an automated "Manager Agent." The manager is responsible for:
- **Strategy**: Assigning tasks to the right agents.
- **Quality Control**: Reviewing agent output and asking for refinements if necessary.
- **Handoffs**: Ensuring data flows correctly between agents.

### 3. Shared Memory & Knowledge
All agents in a crew share a common **Knowledge Base** (RAG) and can access the crew's **Shared Memory**. This ensures that even if two agents work on different tasks, they have a consistent understanding of the project's state.

### 4. Agent-to-Agent (A2A) Networking
In Gocrewwai v1.0, agents can collaborate across the network using the A2A protocol.
- **Discovery**: Agents can advertise their capabilities and find coworkers on the local network.
- **Dynamic Tasking**: An agent can "reach out" to a remote agent to delegate a task, even if that agent is part of a different crew or running on a different server.
- **Strict Auth**: All inter-agent communication is secured via Bearer tokens and mutual TLS option.

---

## 🚀 Implementing Collaborative Crews (Elite Style)

### Standard Hierarchical Coordination
```go
myCrew := gocrew.NewCrew(gocrew.CrewConfig{
    Agents:      []gocrew.CoreAgent{researcher, analyst, writer},
    Process:     gocrew.Hierarchical,
    ManagerLLM:  gpt4o,
})
```

### Advanced A2A Networking (Decentralized Swarms)

To spin up a remote agent that can receive tasks over the network:

```go
// 1. Configure the Remote Agent Node
specialist := gocrew.NewAgent(gocrew.AgentConfig{
    Role:    "Remote CyberSec Analyst",
    A2APort: 9090, // Bind to port 9090
    A2AAuth: gocrew.A2AAuthConfig{
        RequireMTLS: true,
        CertDir:     "/etc/gocrew/certs",
        Token:       os.Getenv("A2A_BEARER_TOKEN"),
    },
})

// 2. Start the listener (Blocks)
go specialist.StartA2AListener()
```

Once running, another agent on a completely different server can connect to it using the Remote API via the `POST /api/create/a2a` bridge.

## 🛡️ Collaboration Guardrails

- **Max Delegation Depth**: Prevent infinite "ask-your-coworker" loops by setting a limit.
- **Human-in-the-Loop**: Require manual approval for delegation requests or coworker handoffs.
- **Strict Output Validation**: Ensure that data passed between agents adheres to your defined JSON schemas.

---

[Back to Reasoning Guide](./reasoning.md) | [Next: Delegation](./agent_delegation.md)
