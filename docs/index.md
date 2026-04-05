# Gocrewwai Documentation ⚓🏆🚀

Welcome to the official documentation for **Gocrewwai**, the high-performance, asynchronous orchestration framework for AI agents, written in idiomatic Go.

---

## ⚓ The Documentation Suite

Explore the core components and features of the Gocrewwai framework through our detailed guides:

### 🚀 Getting Started
- **[Introduction](./CORE_CONCEPTS.md)**: High-level overview and the "Four Pillars".
- **[Installation & Quickstart](./GETTING_STARTED.md)**: Set up your environment and run your first crew.
- **[Migration Guide](./MIGRATION.md)**: Moving from CrewAI, LangChain, or LangGraph.

### 🧩 Core Components
- **[Agents](./AGENTS.md)**: Roles, Goals, Backstories, and autonomous capabilities.
- **[Tasks](./TASKS.md)**: Description, Expected Output, and Structured JSON.
- **[Crews](./CREWS.md)**: Sequential, Hierarchical, and Graph orchestration.

### 🧠 Intelligence & Memory
- **[LLM Providers](./LLM_PROVIDERS.md)**: Multi-provider integration (OpenAI, Anthropic, Gemini, etc.).
- **[Caching](./LLM_PROVIDERS.md#high-performance-caching)**: File and Redis-backed LLM response caching.
- **[Memory Stores](./MEMORY.md)**: Persistent vector-indexed memory (SQLite, Redis, Chroma).
- **[Persistence & Time-Travel](./PERSISTENCE.md)**: Durable execution, checkpoints, and state recovery.

### 🛠️ Extensibility & Resilience
- **[Built-in Tools](./TOOLS.md)**: Search, Browser, Calculator, Code Interpreter, and more.
- **[Custom Tools](./TOOLS.md#creating-custom-tools)**: Wrap any Go function as an agent capability.
- **[Tool Error Handling](./TOOL_ERROR_HANDLING.md)**: Robust tool usage and autonomous retries.
- **[Knowledge (RAG)](./KNOWLEDGE.md)**: Retrieval-Augmented Generation with PDF and Web sources.

### 🌊 Advanced Orchestration
- **[Flows](./FLOWS.md)**: State management, Multi-step workflows, and Human-in-the-loop.
- **[Human-in-the-Loop](./HUMAN_IN_THE_LOOP.md)**: Manual approvals, interrupts, and dashboard integration.
- **[Self-Correction](./SELF_CORRECTION.md)**: Reflective agent loops and autonomous reasoning patterns.

### 📊 Operations & Production
- **[Observability](./OBSERVABILITY.md)**: Native OpenTelemetry tracing and performance monitoring.

---

## 💎 Why Gocrewwai?

- **⚡ Blazing Fast**: Native Goroutines for parallel agent execution.
- **🛡️ Strictly Typed**: Full compile-time safety across all agents and tasks.
- **⚓ Unified SDK**: A single ergonomic import (`gocrew`) for all your needs.

Join the high-performance AI revolution with Gocrewwai! 🚀⚓🛡️
