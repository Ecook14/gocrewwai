# Feature Deep Dive: Agent Training 🎓

Gocrew includes a specialized **Training System** (`pkg/training`) that allows you to improve agent performance through iterative, human-in-the-loop feedback.

---

## 🏗️ How Training Works

Training isn't about fine-tuning the base LLM weights. Instead, it's about building a **Persistent Advice Layer** that guides the agent's behavior based on past corrections.

1. **Training Run**: You run a crew in "Training Mode" using the CLI: `gocrew train --n 5`.
2. **Execution**: The agent performs the tasks.
3. **Human Feedback**: After each task, the agent presents its result. You can then provide "Advice" or "Corrections."
4. **Advice Persistence**: This feedback is saved to a local JSON/SQLite store, indexed by the agent's role and the specific task type.
5. **Guided Reasoning**: In future production runs, the agent automatically retrieves relevant "Advice" from the training store and injects it into its reasoning loop.

---

## 🛠️ The Training Loop

```go
// Training logic is built directly into the Crew engine
crew := gocrew.NewCrewBuilder().
    TrainingDir("./training_data").
    Build()

// Advice is injected during Agent.Execute
if a.TrainingMode {
    advice := a.TrainingStore.GetAdvice(a.Role, task.Description)
    prompt = fmt.Sprintf("%s\n\nPast Human Advice: %s", prompt, advice)
}
```

---

## 📊 Why Train Your Agents?

- **Edge Case Correction**: Fix persistent logic errors without changing the code.
- **Style Alignment**: Teach the agent to match your specific tone or formatting preferences.
- **Tool Mastery**: Guide the agent on *when* and *how* to use specific complex tools more effectively.

---

## 🔄 Managing Training Data

Use the `gocrew` CLI to manage your training data:
- `gocrew reset-training`: Wipes all stored advice.
- `gocrew export-training`: Dumps advice to a shareable JSON file for other developers.

---
**Gocrew** - Continuous improvement through human expertise.
