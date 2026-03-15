# 🗺️ Gocrew Roadmap (2026)

Gocrew is currently in **Early Alpha**. My goal is to build the most performance-optimized, production-ready AI agent framework for the Go ecosystem. I welcome community collaboration to accelerate this vision.

## 🚀 Near-Term (Alpha Focus)
*Targeting v0.9.0*

- [x] **Agent-to-Agent (A2A) Communication**: Standardizing cross-crew delegation and peer-to-peer agent messaging.
- [x] **Advanced Multi-Crew Flows**: More complex combinators (`Switch`, `Map-Reduce`, `Iterate`) for the Flow engine.
- [x] **Enhanced Tool Sandboxing**: Improving Docker/E2B integration for seamless, zero-config code execution.
- [x] **Dashboard Metrics**: Real-time visualization of cost, token usage, and latency per agent/task.
- [x] **core.Agent Interface Standardization**: Seamless interoperability for local and remote agent orchestration.

## ⚡ Mid-Term (Beta Focus)
*Targeting v1.0.0-beta*

- [ ] **Native WASM Multi-Runtime**: Expanding support beyond `wazero` for varied edge compute environments.
- [ ] **Streaming Reasoning**: Real-time visibility into the "thinking" steps of complex reasoning loops.
- [ ] **Plug-and-Play Memory**: Adapters for every major vector database project (Qdrant, Milvus, Chroma, etc.).
- [ ] **CLI Scaffolding++**: Templates for specific industries (Finance, Healthcare, E-commerce).

## 🏢 Long-Term (Production Focus)
*Targeting v1.0.0 Stable*

- [ ] **Distributed Execution**: Running a single Crew across multiple physical nodes.
- [ ] **Agent Governance**: Policy-based constraints for what certain agents can and cannot do.
- [ ] **Multi-Modal Native Core**: Deep first-class support for vision, audio, and video agents beyond basic file wrappers.
- [ ] **Enterprise API Gateway**: A dedicated `gocrew serve` mode for high-throughput production workloads.

---

### 🤝 How to contribute to the Roadmap
- **Vote with Issues**: If one of these features is critical for you, upvote the corresponding issue or start a Discussion.
- **Propose New Paths**: Have an idea not listed here? Open an issue with the `proposal` tag!
