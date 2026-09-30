# Changelog

All notable changes to Gocrewwai will be documented in this file.

## [1.0.0-beta.5] - 2026-09-30

### 🔒 Security hardening (67 agent-findings + 20 review findings, no public API removed)

Fail-closed behavior changes below reject previously-accepted malicious or
malformed inputs only; legitimate clients are unaffected.

**Crash / DoS closed**
- `JSONTool`: array indices require `0 <= idx < len` (strict `Atoi`, was `Sscanf`
  lower-bound-only panic); JSON nesting/input/output budgets; `RegexTool`
  replacement budget.
- Nil-request panics fixed across 10 integrations (HubSpot, Discord, Sheets,
  Jira, +24 sibling sites now check `NewRequest` errors) with identifier
  validation and `PathEscape` per path segment.
- Bounded host-side output for Docker sandbox, CLI executors (`docker`,
  code-interpreter, code-sandbox), arXiv/Ollama decodes, DOCX/xlsx members,
  JSON formatting, regex replacement, and the file matcher (normalize-once +
  input caps). Sandbox cleanup uses a fresh context and reports failures.
- `EventBus`/`Bus.Publish` holds the read lock through non-blocking sends —
  no more send-on-closed panics on SSE disconnect races.

**SSRF closed**
- Shared redirect-revalidating + resolving-dial transport for both scrapers;
  ingestion/files/arXiv reject or re-validate redirects; arXiv moved to HTTPS
  with same-authority redirects; query params are URL-encoded.

**Traversal closed**
- Checkpoint `CrewID` allowlist + `BaseDir` containment (`ValidCheckpointID`);
  symlink validation fails closed on existing dangling links; MCP stdio
  resolves the full chain and executes the validated target; SQLite tool and
  dashboard reject `file:` URIs and percent-escapes.
- File read/write/edit and doc loaders (csv/html/json/xml/yaml/xlsx) open
  through `os.OpenRoot` confinement — no validate-then-open race.
- Memory scopes match on segment boundaries (`/tenant/a` ≠ `/tenant/ab`);
  read-only slices return detached metadata copies.

**Secrets closed**
- Email header-injection guard (single-line subjects/addresses, `ParseAddress`,
  MIME-encoded subjects); Gemini credential moved to auth header with error
  sanitization; audit metadata redacted via JSON-tree normalization;
  telemetry uses constant span names (no prompt text) with record-safe export.
- `UnifiedMemory.Forget` uses the new `ScopeDeleter` primitive (in-mem +
  SQLite) with failure propagation instead of capped-search best-effort.

**AuthZ closed**
- MCP bridge rejects review-required tools; headless REST crews skip MCP
  auto-injection (`WithHeadless`); `AskHuman` sampling denies without an
  approval callback; session IDs get atomic owner/lifecycle gates (409 on
  conflict), owners persist into terminal checkpoints, fallback reads fail
  closed; mesh TLS fails closed on client-CA configs; loopback exception
  requires literal loopback IPs; dashboard honors JWT/read tokens, enforces
  same-origin + `application/json` CSRF guards, authenticates WS subscribers,
  and splits public-metrics vs private-review feeds.

**Integrity**
- Budget admission + heuristic usage accounting on all paid providers;
  `MaxTokens` enforced on OpenAI paths; failover honors `failover_enabled`;
  config parsing split from policy install (kickoff discovery can't clobber
  `CREW_CONFIG_PATH` budgets); mesh delegations serialized per role;
  agent metrics synchronized with snapshot reads; ADK temp tools restored
  per-invocation; guardrails enforced on structured results; SQLite expiry
  compares instants; `gocrew.Recall` forwards scope/source options.

**Dependencies**
- `go-redis` v9.21.0 → v9.22.0 (v9.21.0 is broken upstream — missing internal
  package, blocked all builds). `docker +incompatible` verified expected.

## [1.0.0-beta.4] - 2026-09-29

### ✨ Multi-agent crews over HTTP

- **`POST /api/v1/crews/kickoff` accepts `agents[]`/`tasks[]`** — the engine always ran
  N agents and M tasks, but the endpoint only exposed one of each. The flat
  single-agent fields are unchanged and the two shapes are never mixed (400).
  Up to 10 agents and 32 tasks per request; tasks wire to agents by role
  (omittable only with a single agent); duplicate roles rejected.
- **Per-agent model keys** — each agent resolves its key from its own `api_key`
  first, then the server's `OPENAI_API_KEY`; a requested model with no key
  anywhere is a 503. Keys are construction-only: never logged, never persisted.
- **`crew_process` is honored** in the multi-agent path (it was validated but
  silently ignored before; the flat path keeps its sequential default).
- **Session, idempotency, semaphore, SSE, and owner scoping are shared** with the
  flat path via a common dispatch function, so the guarantees cannot drift.
- **Visual Builder sends whole canvases** — token auth, per-kickoff
  idempotency keys, authenticated SSE streaming, and live node statuses driven
  by real backend event types. The release pipeline compiles the React app and
  embeds it in the server binary (`-tags webdist`); the release job boots the
  binary and fails unless it serves the app.
- **Static frontend paths bypass API auth** (same model as the dashboard: assets
  carry no secrets and navigations cannot send headers); every `/api/*`
  endpoint stays gated.

### 🐛 Fixes

- **`/api/v1/health` reported a hardcoded `0.9.0`** — now reads the link-time
  `pkg/version`, so health, CLI, and banner agree with the release tag.

## [1.0.0-beta.3] - 2026-09-28

### 🐛 Fixes

- **`RedisCheckpointStore.ListCheckpoints` always returned empty** — the SCAN glob pattern
  ended at `"<prefix><crewID>:"` with no trailing `*`, so it matched only a key of exactly
  that name and never the versioned checkpoint keys. Callers saw zero checkpoints on every
  Redis-protocol backend while `Save`/`LoadLatest` kept working, which is why it went
  unnoticed. Caught by the CI compat matrix against real service containers.
- **Idempotency reservation was not a test-and-set** — `reserveIdem` returned nothing, so a
  concurrent request that lost the race after both requests missed `checkIdem` silently
  no-opped and continued, persisting a second session and executing a duplicate crew under
  one `Idempotency-Key`. The loser now receives 409.
- **Kickoff semaphore released tokens to the wrong channel** — `releaseSem` re-read the
  package global instead of returning the token to the channel `acquireSem` took it from,
  permanently draining the acquired channel if the variable was reassigned in between.
- **Kickoff tests dialed the real provider** — two tests set a fake `OPENAI_API_KEY` and
  still made a live `api.openai.com` request (401 + retry), adding ~150ms, flakiness, and
  outbound traffic to CI. They now use an in-memory stub via a `newLLMClient` seam.

### 🔧 Build & release

- **Single source of truth for version** — new `pkg/version` package, injected at link time
  via `-ldflags -X`. Previously the version was hardcoded in three places and the release
  workflow's `-ldflags` was just `-s -w`, so a tagged release shipped binaries that
  self-reported the old version. Unset builds now report `dev` rather than a stale number.
- **Release workflow** now stamps the tag and commit SHA into every binary, verifies the
  embedded version matches the tag (failing the release if not), publishes `SHA256SUMS`,
  and generates release notes. macOS and Windows builds previously got no `-ldflags` at all.
- **Go matrix trimmed to 1.25** — `go.mod` requires `go 1.25.0`, so the 1.23/1.24 legs were
  auto-upgrading via `GOTOOLCHAIN` and testing 1.25 anyway.

### 🧹 Housekeeping

- Removed remaining "Elite" branding from shipped binaries and source, including the
  user-visible `Initiating Elite Graph Execution` log line. Docs were already cleaned; the
  binary and comments were missed.

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
- Dependency upgrades (go 1.25.0 held): mysql 1.10.1, websocket 1.5.3, lib/pq 1.12.3, sqlite3 1.14.52, grpc 1.84.0, go-openai 1.42.1, redis 9.21.0, docker v28, wazero 1.12, slack 0.29, pdf 2026-09 (redis 9.22.0 skipped: its packaging trips the go1.25 module loader; verified clean on go1.27)
- SQLite `latest_pointers` table; file checkpoint locking; real CPU metric; HTML single-pass entities

### ✅ Quality

- Test coverage: every hand-written package now has tests (31 suites green) — dashboard auth/origin, events bus, training store, testutil mocks (aligned to real `llm.Client`), `pkg/testing` harness, facade smoke, sandbox isolation
- `examples/` builds again (`NewFileCache`, `NewCodeInterpreterTool` facade wrappers)
- Version strings unified at 0.9.0; package count corrected to 30; tree gofmt-clean

### ✨ Parity close-out (this session)

- **Kickoff inputs** — `Crew.KickoffWithInputs(ctx, inputs)` with fail-closed `{{var}}` interpolation (`pkg/crew/inputs.go`)
- **TypedFlow real resume** — JSON-envelope checkpoint round-trip; `flows.md` fabrications (`AddInterrupt/AddEdge`) replaced with real APIs
- **YAML providers** — all 6 + recursive `failover` + `temperature/max_tokens/stop` via `llm.DefaultsClient`
- **Facade coverage** — `gocrew/facade_extra.go`: telemetry, MCP/A2A, config, crew checkpoints/viz, LLM middleware/usage/tracing, 6 memory backends, knowledge/guardrail stragglers, flows DAG engine (pinned by coverage test)
- **CLI execution** — `train/test/replay/chat` delegate to the project; subcommand-aware scaffold template; direct SQLite `reset-memories`
- **Docs truth** — tool count 57→51, 6 process types, 6+1 providers, E2B-compatible (custom HTTP, not SDK), CLI command list corrected
- **WASM hardening** — ctx deadline, compile cache, 16MB memory cap, 10MB binary cap, bounded env/output
- **i18n Spanish** — `es.json` full parity + coverage test; **OTel TLS** for non-loopback collectors, loopback-bound Prometheus

### ⚡ Reliability

- **Task race safety** — `sync.RWMutex` + locked setters/getters/`Snapshot()`; hierarchical writes converted (proven with `-race`)
- **Telemetry bus snapshot** — publish no longer holds the lock; bridge supports ctx-cancel via `BridgeEventsToMetricsWithContext`
- **Limiter bounds** — per-IP table cap + idle sweeper (server), windowed sweep + cap (api), TTL-bounded idempotency ledger
- **Memory retry** — `WithRetry` decorator with clamped attempts/backoff, ctx-aware (never retries cancel)
- **Code interpreter** — host fallbacks removed (fail closed), `RequiresReview` derived from sandbox config, `SafeMode` deprecated
- **Config fail-soft** — `Get()` returns empty defaults with warning instead of panicking

### ✨ Features

- **Crew.Test()** — LLM-judged evaluation (`pkg/testing` Evaluator); CLI `-m` forwards judge model; scaffold template judges when `-m` set
- **Output parsers** — XML/CSV/Regex validators + extractors (`pkg/guardrails/parsers.go`) + facade wrappers
- **TLS listeners** — `WithTLSFiles`/env for `pkg/server` + dashboard (`DASHBOARD_TLS_CERT/KEY`)
- **API validation** — binding tags + CrewAI process allowlist on kickoff payload
- **Mesh fail-closed** — insecure default token refuses `Run()` unless `ALLOW_INSECURE_DEV=1`
- **Tavily key hygiene** — api_key moved from URL query to POST JSON body

### 🌍 i18n

- **French (`fr.json`)** — 66/66 key parity with placeholders verified; `Kinds/Keys/Has` API; parity test now exhaustive

### 📊 Benchmarks

- **Opt-in live benchmarks** — `GOCREW_BENCH_LIVE=1` runs OpenAI/Ollama end-to-end latency (skip without keys/daemon); synthetics remain default

### 🔑 Auth

- **JWT Bearer option** — stdlib-only HS256 validation (`pkg/auth`), `JWT_SECRET`-gated on `pkg/server`, `pkg/api` (subject-scoped tenancy), `pkg/dashboard` (admin); `gocrew.NewJWTValidator`

### 📣 Webhooks + deploy

- **Completion webhooks** — `pkg/webhook` HMAC-signed delivery (`X-Gocrew-Signature/Timestamp/Event`), SSRF-gated, failures never fatal; `CrewConfig.WebhookURL/Secret`
- **`gocrew deploy`** — builds release binaries into `--out DIR` (repo-root validated)

### 🗄️ KV layer (Dragonfly-ready)

- **`pkg/kv`** — shared Redis-protocol dial layer (`REDIS_BACKEND=redis|dragonfly|valkey`, fail-closed); memory/checkpoint/LLM-cache constructors rewired, backend logged at server boot with fail-fast validation
- **Compat matrix** — env-switchable suites (`KV_TEST_ADDR`) across all three consumers; CI `kv-compat.yml` runs Dragonfly + Redis service containers
- **Deploy assets** — `deploy/dragonfly/` (compose + k8s via operator), fixed `railway.toml`/`render.yaml` KV wiring, `docs/ops/dragonfly.md` (topology, keyspaces, failover, BSL note)

### ✅ Truth (this session)

- Tool count corrected to 52 constructors; 6 process types; 6+1 providers; E2B marked compatible (custom HTTP)
- Roadmap refreshed (done items struck, JWT/OIDC added); pending_features dashboard section rewritten; UI docs + static search index gained adk/config-errors/flow-vs-flows
- Dockerfile non-root + server entrypoint; CI gofmt gate

### 🧪 Fuzzing

- `FuzzParseCSV`, `FuzzXMLValidator`, `FuzzRegexValidator`, `FuzzInterpolateInputs` — 15s each, zero crashes

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
