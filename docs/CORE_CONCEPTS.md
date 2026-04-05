# Gocrewwai: Core Concepts ⚓🏆🚀

Gocrewwai is a high-performance, asynchronous orchestration framework for AI agents, written in idiomatic Go. It is designed to be the "Elite Tier" alternative to Python-based agentic frameworks, providing superior speed, type safety, and production-ready architecture.

## ⚓ The Four Pillars of Gocrewwai

To build effective AI workflows, you need to understand the four primary abstractions in the ecosystem:

### 1. Agents: The "Who"
Agents are specialized autonomous units programmed with specific **Roles**, **Goals**, and **Backstories**. Unlike simple LLM calls, Gocrewwai agents have a sense of identity and can use **Tools** to interact with the world.
- **Role**: Their job title (e.g., "Senior Researcher").
- **Goal**: What they are trying to achieve (e.g., "Find top 5 AI papers").
- **Backstory**: Context that shapes their personality and prompts.

### 2. Tasks: The "What"
Tasks are specific work units assigned to agents. They define the **Description** of the work, the **Expected Output**, and the necessary **Context** from other tasks. Gocrewwai tasks support sophisticated **Structured Outputs** (JSON) and **Validation Guardrails**.

### 3. Crews: The "How"
A Crew is a collaborative group of agents working together to solve a sequence of tasks. The **Process** defines how the work flows:
- **Sequential**: Tasks are done one after another.
- **Hierarchical**: A "Manager Agent" orchestrates the team, delegating and reviewing work.
- **Graph**: Multi-directional, non-linear workflows based on dynamic state logic.

### 4. Flows: The "When"
Flows are the highest level of orchestration, allowing you to build complex state machines and multi-step workflows that can span across multiple crews. Flows support **State Persistence**, **Human-in-the-loop**, and **Conditional Branching**.

## 🛠️ Feature Highlights

- **⚡ Blazing Fast**: Native Go concurrency (Goroutines) for agent parallelization.
- **🛡️ Strictly Typed**: Full compile-time safety across all agents, tasks, and configurations.
- **🧠 Advanced Memory**: Vector-indexed memory stores (SQLite, Redis, Chroma) for long-term recall.
- **🌊 Native Streaming**: Real-time token streaming for responsive UI/UX.
- **🌐 Unified SDK**: A single ergonomic import (`gocrew`) for all your needs.

---

[Next: Getting Started](./GETTING_STARTED.md) | [Agents Guide](./AGENTS.md) | [Crews Guide](./CREWS.md)
