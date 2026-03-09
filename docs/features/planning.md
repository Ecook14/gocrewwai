# Feature Deep Dive: Task Planning 🗺️

Gocrew agents don't just jump into tasks; they can be configured to **Plan** their approach first. This significantly improves accuracy for complex, multi-step missions.

---

## 🏗️ What is Task Planning?

Planning enables the **Planning Engine** (`pkg/crew/planning.go`). When a task is kicked off:

1. **Strategic Analysis**: A specialized `PlanningLLM` analyzes the task description and the agent's tools.
2. **Plan Generation**: The LLM generates a structured, step-by-step "Plan" for how the agent should proceed.
3. **Mission Briefing**: This plan is injected at the very top of the agent's prompt, acting as a mental roadmap for the reasoning loop.

---

## 🛠️ Enabling Planning

Planning can be enabled globally for a crew or overridden for specific configurations.

```go
myCrew := gocrew.NewCrewBuilder().
    Planning(true). // Turn on the planning engine
    PlanningLLM(smartModel). // Optional: Use a more powerful model for planning
    Build()
```

---

## 📊 The Benefits of Planning

1. **Tool Optimization**: The planner identifies *which* tools are most likely to yield the best results early on.
2. **Reduced Hallucination**: By defining steps before execution, agents are less likely to drift off-target.
3. **Complex Reasoning**: For tasks requiring multiple hops (e.g., "Research A, then calculate B, then summarize for C"), the plan ensures no step is missed.

---

## 🧩 Plan Components

A typical Gocrew task plan includes:
- **Major Steps**: The logic flow (e.g., "Step 1: Scrape financial data").
- **Tool Selection**: Specific tool mappings for each step.
- **Stop Criteria**: Defining when each sub-step is considered "Done."

---
**Gocrew** - Precision thinking through structured planning.
