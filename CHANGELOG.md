# Changelog 📜

All notable changes to the **Gocrew** project will be documented in this file.

---

## [Unreleased]
### Added
- **Mesh gRPC server** (`pkg/api/mesh`) — language-agnostic delegation over the wire with three RPCs: `DelegateTask`, `SearchKnowledge`, `IndexKnowledge`.
- **AgentDiscovery** (`pkg/protocols/discovery.go`) — background scanning and `AgentRegistry` for A2A peer discovery.
- **AgentManager** (`pkg/agents/manager.go`) — first-class hierarchical crew manager driver.
- **Agent cloning** (`pkg/agents/clone.go`) — duplicate an agent with overrides and optional isolated memory.
- **Crew visualization** (`pkg/crew/visualization.go`) — DOT export for crew graphs.
- **TypedFlow[T]** (`pkg/flow/typed_flow.go`) — generic typed flow with `WithPersistence`.
- **8 memory backends** — SQLite, Redis, Chroma, Qdrant, Pinecone, Weaviate, InMemCosine, InMemEntity.
- **6 process types** — Sequential, Hierarchical, Consensual, Graph, Reflective, StateMachine.
- **8 guardrail types** — MaxToken, ContentFilter, Schema, PIIRedaction, Toxicity, JSON, Sanitizer, HumanReview.
- **3 sandbox providers** — Wasm (wazero), Docker, E2B Cloud Sandbox.
- **WASM shell tools** (`pkg/tools/wasm_sandbox.go`).
- **Wolfram Alpha tool** (`pkg/tools/wolfram.go`).
- **WASM file tools** (`pkg/tools/file_{read,write,edit}.go`) with chroot isolation.
- **MongoDB tool** (`pkg/tools/mongodb.go`).

### Changed
- **Dashboard** — moved to `web/` (React/Vite). `web-ui/embed.go` remains for backward-compatible embeds.

### Documentation
- **Examples** — 30 ready-to-run examples in `examples/` (was 13 documented).
- **Architecture** — added `pkg/protocols/discovery.go`, `pkg/crew/visualization.go`, `pkg/i18n/translations/`.

---

## [0.9.0] - 2026-03-10
### Added
- **core.Agent Interface Migration**: Structural decoupling enable heterogeneous local and remote agent orchestration.
- **Distributed Resilience**: Standardized `RemoteAgentAdapter` with circuit breaking and OTel tracing.
- **Stability**: Resolved all 22 execution-layer compilation errors across core and examples.
- **Dynamic Dashboard**: Enhanced creator mode with `core.Agent` interface support for live injection.

## [0.8.0] - 2026-03-09
### Added
- **Initial Public Release**: The Gocrew framework is now open-source!
- **Unified SDK Facade**: Optimized `gocrew` package for a high-performance library experience.
- **Documentation**: New comprehensive guides for Agents, Tasks, and Memory.
- **Build**: 100% verified build success across all examples.

## [0.1.0 - 0.7.0] - Internal Development
- Initial foundation, iteration on core engine packages, and internal stabilizer builds.
- Development of the Glassmorphic Dashboard and tool ecosystem.
- Refinement of the ReAct reasoning loop and memory scoring logic.
