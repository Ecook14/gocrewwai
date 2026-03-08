# Feature: Autonomous Agents

## Overview
The `Agent` in Crew-GO is the foundational intelligent unit. Unlike simple wrappers that only send a prompt to an API, Crew-GO Agents operate on an **Autonomous ReAct (Reason + Act) Loop**.

## Key Capabilities

1. **Role, Goal, Backstory**: The core unchangeable persona of the agent. This guarantees consistent behavior across hundreds of iterations.
2. **Dynamic Tool Use**: Agents are given slices of `tools.Tool`. They natively parse available tools to generate JSON requests that the Go engine intercepts, executes securely, and returns as "Observations".
3. **Guardrails**: Agents support strict output sanitization. Before an Agent is "Allowed" to complete its task, its final output passes through `Agent.Guardrails`. If it fails (e.g., PII detected), the Go engine throws an error *directly back to the LLM*, forcing the agent to auto-correct and retry.
4. **Self-Critique**: An optional strict parameter `SelfCritique`. Before returning an answer, the Agent evaluates its own response. If the response violates its core persona, it rewrites it automatically.

## Memory Integration
Agents natively integrate to three tiers of memory:
- **Short-Term**: Recalling context of the current task iteration.
- **Long-Term**: Recalling past interactions via vector embeddings (RAG).
- **Entity Memory**: Extracting specific facts about subjects and storing them in distinct KV pairs for high-precision recall.

## Usage
```go
researcher := agents.NewAgentBuilder().
    Role("Data Scientist").
    Goal("Find accurate statistics").
    Backstory("You are incredibly pedantic about citations.").
    LLM(client).
    Tools(searchTool, calculatorTool).
    SelfCritique(true).
    Build()
```
