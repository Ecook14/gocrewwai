# Feature Deep Dive: Autonomous Agents ⚓🤖

Gocrew agents are stateful, goal-oriented entities designed for reliable orchestration. They are more than just LLM wrappers; they are autonomous loops with memory and tools.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai agents utilize a declarative, strictly-to-typed configuration pattern that ensures compile-time safety and predictable orchestration.

---

## 🏗️ The Agent Config (Elite Style)

In Gocrewwai v1.0, agents are constructed using the `AgentConfig` struct, providing a clean, declarative interface.

```go
agent := gocrew.NewAgent(gocrew.AgentConfig{
    Role:            "Strategic Advisor",
    Goal:            "Formulate high-level project goals",
    Backstory:       "Ex-consultant with a focus on efficiency.",
    LLM:             model,
    Verbose:         true,
    AllowDelegation: true,
})
```

### Key Parameters

| Parameter | Type | Description |
| :--- | :--- | :--- |
| **Role** | `string` | The functional persona of the agent. |
| **Goal** | `string` | The objective the agent is trying to achieve. |
| **Backstory** | `string` | Provides context and personality to the agent. |
| **LLM** | `LLMClient` | The model that powers the agent (e.g., OpenAI, Claude). |
| **Tools** | `[]Tool` | A slice of tools available to the agent. |
| **Memory** | `MemoryStore` | Enables the agent to store and recall past context. |
| **SelfCritique** | `bool` | Enables the **Reflect -> Evaluate -> Refine** loop. |
| **Reasoning** | `bool` | Activates deep reasoning mode for complex problem solving. |
| **SelfHealing** | `bool` | Allows the agent to autonomously fix tool & code errors. |
| **Sandbox** | `string` | Isolation environment for code tools (`"local"`, `"docker"`, `"e2b"`, `"wasm"`). |
| **InjectDate** | `bool` | Injects current system date into prompts. |
| **MaxRPM** | `int` | Agent-level rate limiting for API safety. |
| **MaxIterations** | `int` | Global cap on the number of steps an agent can take (Default: 15). |
| **StepReview** | `func` | Human-in-the-loop hook for tool approval. |
| **MCPS** | `[]string` | URLs or stdio commands for MCP server connection. |
| **A2APort** | `int` | Port for inter-agent communication (if > 0). |
| **Language** | `string` | Preferred language for prompts/formatting (e.g., `"es"`, `"fr"`). |

---

## 🔄 The ReAct Reasoning Loop

Gocrew agents follow a Go-native implementation of the **Reason-Act-Observe** pattern:

1. **Thought**: The agent analyzes the task and decides on an action.
2. **Action**: The agent generates a tool call (JSON).
3. **Execution**: The Go engine executes the tool (locally or in a sandbox).
4. **Observation**: The tool output is fed back to the agent.
5. **Final Answer**: Once the agent believes it has enough info, it generates the final result.

---

## 🛡️ Guardrails

Guardrails are strictly-typed rules that agent output MUST pass before being accepted.
- **PII Redactor**: Masks sensitive data.
- **Human Review**: Pauses execution for manual approval through the Dashboard.
- **LLM Review**: Uses a second "Critic" agent to grade the output of the first.

---

## 🌐 Heterogeneous Swarms (`gocrew.CoreAgent`)

In Gocrew v1.0, all orchestration logic utilizes the polymorphic `gocrew.CoreAgent` interface (aliased to `core.Agent`). This decouples the engine from the physical implementation of the agent.

Why does this matter?
- **Local Agents**: Your standard `gocrew.Agent` runs queries in the same process.
- **Remote Agents**: Using the `RemoteAgentAdapter`, you can dynamically inject remote agents (running on different servers) into a local crew. To the orchestration engine, they look exactly the same!

This interface standardization enables massive distributed workloads and true Agent-to-Agent (A2A) networking.

---

[Back to index](../index.md) | [Next: Tasks](./tasks.md)
