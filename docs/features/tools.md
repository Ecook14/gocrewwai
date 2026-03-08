# Feature: Tool Arsenal & Sandboxing

## Overview
Crew-GO breaks the mold of minimal-tool frameworks by providing **over 34 native integrations** inside `pkg/tools`. These are completely type-safe Go implementations engineered for speed and reliability.

## Elite Multi-modal Execution Sandboxes
When an Agent decides to write and execute code (e.g. `CodeInterpreterTool`), safety is the absolute highest priority. Running `eval()` natively is catastrophic. Crew-GO provides tiered sandboxing:

1. **Local Isolate**: Subprocess isolation. (Risky, requires total trust).
2. **Docker Ephemeral Engine**: 
   - Dynamically pulls target containers (`python:3.11-slim`).
   - Bounds CPU and Memory limits heavily.
   - Executes agent-generated code seamlessly and returns stdout/stderr.
3. **WASM (wazero)**:
   - Compiles and executes WebAssembly *inside the Go runtime* with zero external dependencies.
   - Total memory isolation. Instant startup times (nanoseconds compared to Docker's milliseconds).

## Built-In Capabilities
*   **Web Scraping & RAG**: `BrowserTool` (Chromedp headful/headless SPAs), `ExaTool` (Neural Web Search), `SerperTool`.
*   **Databases**: Native query execution on `Postgres`, `MySQL`, `MongoDB`, `ElasticSearch`.
*   **Productivity**: `Slack`, `GitHub`, `Email`.
*   **Math & Logic**: `Calculator`, `WolframAlpha`.

## Self-Healing Tools
If a tool encounters a transient error (e.g., an HTML layout changed and the scraper failed, or a Postgres SQL syntax error):
1. The Go error is captured.
2. It is appended to the Agent's context as: *"Fatal Tool Error: syntax error at or near 'SELECTT'. Please correct your parameters and try again."*
3. The Agent autonomously corrects its JSON payload and fires the tool again.
