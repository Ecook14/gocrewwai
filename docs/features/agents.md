# Feature Deep Dive: Autonomous Agents 🤖

Gocrew agents are stateful, goal-oriented entities designed for reliable orchestration. They are more than just LLM wrappers; they are autonomous loops with memory and tools.

---

## 🏗️ The Agent Builder

Construct agents fluently using the `AgentBuilder`.

```go
agent := gocrew.NewAgentBuilder().
    Role("Strategic Advisor").
    Goal("Formulate high-level project goals").
    Backstory("Ex-consultant with a focus on efficiency.").
    LLM(model).
    Build()
```

### Key Parameters

- **Reasoning (`bool`)**: Enables the **Reflect -> Evaluate -> Refine** loop. The agent will internally critique its own thoughts before taking action.
- **SelfHealing (`bool`)**: Allows the agent to autonomously fix tool errors (e.g., if a Python script crashes, the agent reads the traceback and tries to fix the code).
- **Sandbox (`string`)**: Defines the execution environment for code tools. Options: `"local"`, `"docker"`, `"e2b"`, `"wasm"`.
- **InjectDate (`bool`)**: Automatically injects the current system date into the prompt to prevent "knowledge cutoff" hallucinations.
- **MaxRPM (`int`)**: Enforces rate limiting at the individual agent level to prevent API token exhaustion.
- **AllowDelegation (`bool`)**: Determines if this agent can ask coworkers for help or delegate sub-tasks.

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

## 🌐 Heterogeneous Swarms (`core.Agent`)

In Gocrew v0.9, all orchestration logic utilizes the polymorphic `core.Agent` interface rather than concrete `*agents.Agent` pointers. This decouples the engine from the physical implementation of the agent.

Why does this matter?
- **Local Agents**: Your standard `gocrew.Agent` runs queries in the same process.
- **Remote Agents**: Using the `RemoteAgentAdapter`, you can dynamically inject remote agents (running on different servers) into a local crew. To the orchestration engine, they look exactly the same!

This interface standardization enables massive distributed workloads and true Agent-to-Agent (A2A) networking.

---
**Gocrew** - Built for reliable autonomy.
