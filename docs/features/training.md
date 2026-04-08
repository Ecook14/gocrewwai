# Feature Deep Dive: Training ⚓🎓🛡️

Training in Gocrewwai goes beyond simple prompt engineering. It is the process of improving agent performance over time by capturing human feedback and "injecting" it into the agent's long-term memory as persistent knowledge.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai training mode supports **Feedback Capture**, **Advice Injection**, and **Long-term Reasoning Refinement**.

---

## 🏗️ The Training Workflow

Gocrewwai training is integrated directly into the **Human-in-the-Loop** (HITL) cycle of the crew mission.

1. **Pause**: The crew execution reaches a task that requires human approval.
2. **Interact**: Through the **Dashboard**, you review the agent's draft and provide corrective feedback (e.g., "The tone is too formal, make it more conversational").
3. **Learn**: Gocrewwai's engine captures this feedback and stores it in the agent's **Entity Memory**.
4. **Persist**: In future runs, the agent will recall this advice when faced with a similar task, autonomously applying the learned preference.

---

## 🚀 Implementing Training Mode (Elite Style)

Using the `gocrew` SDK, you can enable training mode with a single boolean flag:

```go
myCrew := gocrew.NewCrew(gocrew.CrewConfig{
    Agents:   []gocrew.CoreAgent{researcher, writer},
    Tasks:    []*gocrew.Task{articleTask},
    Training: true, // Enable feedback capture and memory injection
})

// Trigger a Training Pause and apply human feedback
func acceptTaskFeedback(taskID string, humanFeedback string) {
    err := myCrew.ProvideTrainingFeedback(taskID, gocrew.TrainingFeedback{
        Rating:  -1, // Negative rating triggers Agent learning loop
        Comment: humanFeedback,
    })
    
    // The agent will capture this into Entity Memory and recall it for all future runs
}
```

## 🧠 Memory-Based Refinement

### 1. Advice Recall
During the planning phase of the mission, agents perform a vector search in their memory for any training feedback related to the current task. This advice is then injected directly into the agent's reasoning prompt.

### 2. Multi-Agent Training
If you train a "Lead Researcher," any knowledge or advice captured during the training session can be shared with other researcher agents in the same crew, creating a "Collective Intelligence" effect.

---

## 🛡️ Training Safety & Reset

- **Feedback Auditing**: Review and edit captured training data through the Gocrewwai CLI or Dashboard.
- **Memory Reset**: If an agent's reasoning develops a negative bias, you can clear its training memory without affecting its core backstory or tools.
- **Human-only Training**: Only designated "Humans" can provide training feedback, ensuring that the agent's knowledge base remains trustworthy and verified.

---

[Back to Testing Guide](./testing.md) | [Back to index](../index.md)
