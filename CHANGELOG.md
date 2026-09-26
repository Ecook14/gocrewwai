# Changelog

All notable changes to Gocrewwai will be documented in this file.

## [Unreleased]

### 🔒 Security

- **API tenancy-lite** — `API_AUTH_TOKENS` multi-token set; sessions owner-scoped per token fingerprint, cross-owner reads return not-found
- **Kickoff idempotency** — `Idempotency-Key` header (409 while running, cached replay when done) + duplicate-session 409
- **SSE session validation** — `/stream/:id` requires a live caller-owned session; crew events carry `session_id`
- **A2A header-only auth** — `?token=` query fallback removed; server timeouts added
- **Dashboard hardening** — bearer token on `/api/*`, same-origin `CheckOrigin` default, server timeouts
- **Shell metachar blocking** — `;|&$\`()<>` rejected even for whitelisted commands
- **Mesh TLS dials** — `RemoteAgent`/`RemoteKnowledgeSource` use TLS with `MESH_TLS_CA`, warn otherwise
- **Tool review defaults** — Arxiv, Brave Search now require review; file/cache keys per path
- **Training path traversal** — role names sanitized in checkpoint store
- **Mesh TLS by default** — servers mint an ephemeral self-signed cert unless `MESH_INSECURE=1`; clients fail closed on remote plaintext without `MESH_TLS_CA`; loopback dev stays encrypted
- **Mesh graceful shutdown** — `GracefulStop` wired into SIGTERM path
- **Session-scoped SSE** — `/stream/:id` carries only the caller's session lifecycle events + global metrics
- **Config fail-soft** — `TryGet()` + duration warnings; server exits cleanly on bad config

### ⚡ Resilience & deps

- **LLM circuit breaker** — `WithCircuitBreaker(threshold, cooldown)` on `MiddlewareClient`, fail-fast `ErrCircuitOpen`
- Dependency upgrades (go 1.25.0 held): mysql 1.10.1, websocket 1.5.3, lib/pq 1.12.3, sqlite3 1.14.52, grpc 1.84.0, go-openai 1.42.1, redis 9.22.0, docker v28, wazero 1.12, slack 0.29, pdf 2026-09
- SQLite `latest_pointers` table; file checkpoint locking; real CPU metric; HTML single-pass entities

### ✅ Quality

- Test coverage: every hand-written package now has tests (31 suites green) — dashboard auth/origin, events bus, training store, testutil mocks (aligned to real `llm.Client`), `pkg/testing` harness, facade smoke, sandbox isolation
- `examples/` builds again (`NewFileCache`, `NewCodeInterpreterTool` facade wrappers)
- Version strings unified at 0.9.0; package count corrected to 30; tree gofmt-clean

## [v0.9.0] - 2026-09-09

### ✨ Added

- **Ollama native client** — Full local LLM inference with streaming, structured output, embeddings
- **8 new SaaS tools** — Google Sheets, Linear, Twilio, Brave Search, Notion, HubSpot, Jira, Supabase, SendGrid, Discord
- **AAMARVA network integration** — `pkg/aamarva/client.go` with Register, Login, Search, Post, Reply, Connect
- **GitHub Actions CI/CD** — Build, test, and release automation for Linux/Mac/Windows
- **57 built-in tools** — Comprehensive tool ecosystem covering CRM, search, messaging, databases
- **12 memory backends** — SQLite, Redis, Chroma, Pinecone, Qdrant, Weaviate, and more
- **7 LLM providers** — OpenAI, Anthropic, Gemini, Groq, OpenRouter, Ollama, Failover
- **Go 1.25 compatibility** — Full support for latest Go toolchain
- **Documentation overhaul** — PROGRESS.md, Gap.md, AAMARVA_INTEGRATION.md, architecture docs
- **Test coverage** — 24/24 packages passing, 100+ tests

### 🔧 Fixed

- go.mod version format fixed to match Go toolchain
- Agent interface compatibility across all test mocks
- Tool registration and discovery
- Config singleton testability
- Broken Windows paths in source files

## [v0.8.0] - 2026-08-01

### ✨ Added

- Core agent framework with role-based agents
- Crew orchestration (Sequential, Hierarchical, Graph, Consensual, Reflective, StateMachine)
- Flow persistence with checkpoints
- MCP server support
- A2A protocol implementation
- HITL (Human-in-the-Loop) support
- OpenTelemetry tracing
- CLI scaffolding (`gocrew create`, `gocrew run`, `gocrew kickoff`)
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
