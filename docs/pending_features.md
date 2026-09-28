# Crew-GO Pending Features & Technical Debt ⚠️

While Crew-GO is exceptionally feature-rich for an enterprise deployment, there are several modules, stubs, and incomplete features that require attention before a `v0.9.0` general availability release.

## ~~1. LLM Client Stubs (Multi-modal & Audio)~~ ✅ COMPLETED

~~The `llm.Client` interface defines methods for `GenerateEmbedding`, `GenerateSpeech`, and `TranscribeSpeech`...~~

*Note: The `llm.Client` interface has been successfully segregated using pure Go interface patterns (`llm.Embedder`, `llm.AudioGenerator`). Empty stubs removed from Anthropic and Gemini. Downstream logic utilizes safe type-assertions.*

## ~~2. Advanced Graph Replanning (Manager Synthesis)~~ ✅ COMPLETED

~~In `pkg/crew/crew.go`, under the `executeHierarchical` function, there is a **Dynamic Re-Planning Stage**...~~

*Note: Natively implemented using an unbounded execution loop. Managers now seamlessly append `tasks.Task{}` structs that are picked up by the execution goroutines natively.*

## 3. Web UI Dashboard (`web-ui/`)

The real-time telemetry dashboard streams events from `pkg/telemetry` and serves
entity management + HITL review from `pkg/dashboard` (`POST /api/review`,
approve/reject from the browser; `/ws` telemetry stream; Bearer + read-only
tokens; same-origin WebSocket check). Static `index.html`/`app.js` era is over —
see `docs/features/production.md` and the `dashboard_demo`/`mission_control` examples.

**Remaining:**
*   `gocrew deploy` + webhook triggers for managed hosting (see `docs/Gap.md` roadmap).

## ~~4. Wasm Sandboxing Limitations~~ ✅ COMPLETED

~~The `WASMSandboxTool` is implemented via `wazero`. While incredibly secure, WASM running in go lacks network access...~~

*Note: WASMSandboxTool enhanced with explicit `MountedDirs` for full OSFS/MemFS control and exposes an `env.http_proxy_get` function for memory-safe outgoing network execution.*

## ~~5. File System Extraction & RAG Storage~~ ✅ COMPLETED

~~While Memory stores (Chroma, Redis, Weaviate) are implemented in `pkg/memory`, the automatic document ingestion pipeline...~~

*Note: Integrated a dependency-free docx `archive/zip` XML text unroller and the `ledongthuc/pdf` library to natively support complex enterprise document formats within `IngestionEngine`.*

---
*Note: This document should be updated iteratively as sprints and pull requests cover these stubs.*
