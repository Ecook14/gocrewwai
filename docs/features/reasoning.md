# Feature Deep Dive: Advanced Reasoning 🧠

Gocrew agents can be upgraded from simple execution loops to sophisticated **Reasoning Engines**. By enabling the Reasoning flag, agents gain the ability to critique, evaluate, and refine their own thoughts before acting.

---

## 🏗️ The Reasoning Loop

When `Reasoning: true` is set on an agent, the standard ReAct loop is wrapped in a **Reflective Loop**:

1. **Generation**: The agent generates a draft thought or action.
2. **Critique**: A specialized "Reflection Prompt" is used to analyze the draft for logical fallacies, tool misuse, or goal misalignment.
3. **Evaluation**: The agent assigns a "Confidence Score" to its own thought.
4. **Refinement**: If the score is below the threshold, the agent re-generates the thought based on the critique.
5. **Execution**: Only high-confidence thoughts proceed to actual tool calls.

---

## 🛠️ Enabling Reasoning

Reasoning is a high-level flag available in the `AgentBuilder`.

```go
agent := gocrew.NewAgentBuilder()
    .Role("Strategic Auditor")
    .Reasoning(true) // Enable advanced iterative reasoning
    .Build()
```

---

## 🛡️ Self-Healing Capabilities

Reasoning agents are also capable of **Self-Healing**. If a tool call fails (e.g., a Python syntax error or a 404 URL):
- The agent analyzes the error message.
- It identifies the cause of the failure.
- It generates a *new* action designed to fix the problem (e.g., "The previous script failed due to a missing library; I will rewrite it using standard library functions").

---

## 📊 Performance Trade-offs

Advanced Reasoning is powerful but comes with considerations:
- **Latency**: Multiple internal reflection steps increase the time to final answer.
- **Cost**: More tokens are consumed due to the iterative nature of the loop.
- **Accuracy**: Significantly higher for complex tasks where "first-thought" hallucinations are common.

---
**Gocrew** - Mastery through self-reflection.
