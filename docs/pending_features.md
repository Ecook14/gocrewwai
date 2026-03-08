# Crew-GO Pending Features & Technical Debt ⚠️

While Crew-GO is exceptionally feature-rich for an enterprise deployment, there are several modules, stubs, and incomplete features that require attention before a `v1.0.0` general availability release.

## ~~1. LLM Client Stubs (Multi-modal & Audio)~~ ✅ COMPLETED

~~The `llm.Client` interface defines methods for `GenerateEmbedding`, `GenerateSpeech`, and `TranscribeSpeech`...~~

*Note: The `llm.Client` interface has been successfully segregated using pure Go interface patterns (`llm.Embedder`, `llm.AudioGenerator`). Empty stubs removed from Anthropic and Gemini. Downstream logic utilizes safe type-assertions.*

## ~~2. Advanced Graph Replanning (Manager Synthesis)~~ ✅ COMPLETED

~~In `pkg/crew/crew.go`, under the `executeHierarchical` function, there is a **Dynamic Re-Planning Stage**...~~

*Note: Natively implemented using an unbounded execution loop. Managers now seamlessly append `tasks.Task{}` structs that are picked up by the execution goroutines natively.*

## 3. Web UI Dashboard (`web-ui/`)

The real-time telemetry Glassmorphism UI is functional for streaming events from `pkg/telemetry`. However, it currently lives as a static `index.html` and `app.js`.

**Action Required**:
*   The UI doesn't allow for starting/stopping the Crew execution dynamically; it acts purely as a read-only observability layer.
*   HITL (Human-in-the-loop) reviews currently print to the console. The Web UI needs a websocket endpoint `POST /review` to allow users to click [Approve/Reject] on a tool call directly from the browser.

## ~~4. Wasm Sandboxing Limitations~~ ✅ COMPLETED

~~The `WASMSandboxTool` is implemented via `wazero`. While incredibly secure, WASM running in go lacks network access...~~

*Note: WASMSandboxTool enhanced with explicit `MountedDirs` for full OSFS/MemFS control and exposes an `env.http_proxy_get` function for memory-safe outgoing network execution.*

## ~~5. File System Extraction & RAG Storage~~ ✅ COMPLETED

~~While Memory stores (Chroma, Redis, Weaviate) are implemented in `pkg/memory`, the automatic document ingestion pipeline...~~

*Note: Integrated a dependency-free docx `archive/zip` XML text unroller and the `ledongthuc/pdf` library to natively support complex enterprise document formats within `IngestionEngine`.*

---
*Note: This document should be updated iteratively as sprints and pull requests cover these stubs.*
