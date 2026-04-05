# Feature Deep Dive: Reasoning ⚓⚡🤖

Gocrewwai agents are not just linear task executors; they are sophisticated reasoning engines. By combining **Self-Critique**, **Step-by-Step Thought**, and **Reflective Loops**, Gocrewwai ensures the highest level of reliability in AI-generated output.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai reasoning utilizes **Recursive Self-Critique** and **Consensus-based Reflection** to ensure 100% adherence to mission goals.

---

## 🏗️ Core Reasoning Principles

### 1. Step-by-Step Thought (Reasoning)
Gocrewwai agents utilize a Go-native implementation of the **Think -> Act -> Observe** loop. This ensures that every tool call and decision is backed by a logical reasoning step, which is visible in real-time on the **Dashboard**.

### 2. Recursive Self-Critique
By enabling `SelfCritique: true` in the `AgentConfig`, you empower the agent to review its own generated answer against the original goal and task description. If the agent identifies a gap, it will internally refine the answer before presenting it.

### 3. Reflective Loops (Peer Review)
The most advanced reasoning pattern in Gocrewwai. Using the `Reflective` process, a "Primary Agent" generates a draft, and a "Critic Agent" reviews it. This peer-review cycle continues until a consensus is reached or the retry limit is hit.

---

## 🚀 Implementing Advanced Reasoning (Elite Style)

Using the `gocrew` SDK, you can enable these advanced patterns with simple declarative configuration:

```go
expert := gocrew.NewAgent(gocrew.AgentConfig{
    Role:         "Senior Architect",
    Goal:         "Design a scalable system architecture.",
    SelfCritique: true, // Enable autonomous reflection loop
    Verbose:      true,
})
```

## 🛡️ Reasoning Guardrails

- **Typed Validation**: Gocrewwai enforces strictly-typed JSON schemas for agent output.
- **Human-in-the-Loop**: Pause the reasoning loop at any point for manual expert sign-off.
- **Context Injection**: Automatically inject current date, location, and metadata to prevent reasoning drift.

---

[Back to Planning Guide](./planning.md) | [Next: Collaboration](./collaboration.md)
