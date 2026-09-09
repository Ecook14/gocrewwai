# Changelog

All notable changes to Gocrewwai will be documented in this file.

## [v0.9.0] - 2026-09-09

### ✨ Added

- **Ollama native client** — Full local LLM inference with streaming, structured output, embeddings
- **8 new SaaS tools** — Google Sheets, Linear, Twilio, Brave Search, Notion, HubSpot, Jira, Supabase, SendGrid, Discord
- **AAMARVA network integration** — `pkg/aamarva/client.go` with Register, Login, Search, Post, Reply, Connect
- **GitHub Actions CI/CD** — Build, test, and release automation for Linux/Mac/Windows
- **51 built-in tools** — Comprehensive tool ecosystem covering CRM, search, messaging, databases
- **19 memory backends** — SQLite, Redis, Chroma, Pinecone, Qdrant, Weaviate, and more
- **8 LLM providers** — OpenAI, Anthropic, Google Gemini, DeepSeek, Groq, OpenRouter, Ollama
- **Go 1.25 compatibility** — Full support for latest Go toolchain
- **Documentation overhaul** — PROGRESS.md, Gap.md, AAMARVA_INTEGRATION.md, architecture docs
- **Test coverage** — 17/17 packages passing, 100+ tests

### 🔧 Fixed

- go.mod version format fixed to match Go toolchain
- Agent interface compatibility across all test mocks
- Tool registration and discovery
- Config singleton testability
- Broken Windows paths in source files

## [v0.8.0] - 2026-08-01

### ✨ Added

- Core agent framework with role-based agents
- Crew orchestration (Sequential, Hierarchical, Graph)
- Flow persistence with checkpoints
- MCP server support
- A2A protocol implementation
- HITL (Human-in-the-Loop) support
- OpenTelemetry tracing
- CLI scaffolding (`gocrew init`, `gocrew new`, `gocrew run`)
- Dashboard server with real-time metrics

### 🔧 Fixed

- Memory manager initialization
- Task execution error handling
- Agent delegation logic

## [v0.7.0] - 2026-07-01

### ✨ Added

- Self-correction and reflective crews
- Knowledge base with RAG
- Training data pipelines
- Event bus for async communication
- Guardrails framework
- Multi-provider LLM routing

## [v0.1.0] - 2026-06-01

### ✨ Added

- Initial project structure
- Core agent interface
- Basic crew execution
- Tool registration system
- CLI entry point
- Configuration management
