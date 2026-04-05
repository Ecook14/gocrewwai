# Agents in Gocrewwai ⚓

Agents are the autonomous core of Gocrewwai. Each agent is a specialized unit designed to perform specific tasks within a team. 

## 🏗️ Agent Anatomy

In Gocrewwai, an agent is defined by its role, goal, backstory, and its capabilities (LLM, Tools, Memory).

### 🛠️ Key Parameters

| Field | Type | Description |
| :--- | :--- | :--- |
| **Role** | `string` | The job title or persona of the agent (e.g., "Researcher"). |
| **Goal** | `string` | The objective the agent is trying to achieve. |
| **Backstory** | `string` | Provides context and personality to the agent's prompts. |
| **LLM** | `LLMClient` | The language model that powers the agent (e.g., OpenAI, Claude). |
| **Tools** | `[]Tool` | A slice of tools the agent can use to interact with the world. |
| **Memory** | `MemoryStore` | Enables the agent to store and recall past experiences. |
| **Verbose** | `bool` | Enables detailed logging of the agent's internal thought process. |
| **AllowDelegation** | `bool` | Allows the agent to delegate sub-tasks to other agents in the crew. |

## 🚀 Creating an Agent (Elite Style)

Using the `gocrew` SDK, you can create agents using a declarative configuration:

```go
package main

import "github.com/Ecook14/gocrewwai/gocrew"

func main() {
    gpt4 := gocrew.NewOpenAI("api-key", "gpt-4o")

    researcher := gocrew.NewAgent(gocrew.AgentConfig{
        Role:            "Senior Researcher",
        Goal:            "Perform deep analysis on emerging AI trends.",
        Backstory:       "With a decade of experience in ML, you excel at trend detection.",
        LLM:             gpt4,
        Verbose:         true,
        AllowDelegation: true,
    })
}
```

## 🧠 Advanced Agent Capabilities

### 1. Self-Critique & Reflection
Enable `SelfCritique: true` in the `AgentConfig` to make the agent reflect on its own answers before returning them.

### 2. Multi-Modal Vision
If using a compatible model (like `gpt-4o`), enable `Multimodal: true` to allow the agent to process images passed via its execution context.

### 3. Step Callbacks & Streaming
- **`StepCallback`**: Hook into every thought/action cycle of the agent for UI updates.
- **`StepStreamCallback`**: Receive real-time token streams for high-performance messaging.

---

[Back to Core Concepts](./CORE_CONCEPTS.md) | [Next: Tasks Guide](./TASKS.md)
