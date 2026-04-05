# Observability & Tracing in Gocrewwai 📊

Gocrewwai is built with production-grade observability from the ground up. Unlike other frameworks that require complex third-party wrappers, Gocrewwai includes native **OpenTelemetry (OTEL)** integration for transparent, high-fidelity tracing of every agentic action.

## 🏗️ Observability Architecture

Gocrewwai's observability system is designed for both local development and high-scale production monitoring.

| Component | Gocrewwai Implementation | Key Advantage |
| :--- | :--- | :--- |
| **Tracer** | `pkg/telemetry` (OTEL) | Vendor-neutral, standards-compliant tracing. |
| **Exporter** | HTTP/gRPC/Honeycomb/Zipkin | Export logs and traces to any OTEL-compatible backend. |
| **Auto-Instrumentation** | Native in Engine/Crews/Tasks | Zero-code setup for tracing entire multi-agent lifecycles. |
| **Dashboard** | Built-in TUI & Web Engine | Visual, real-time tracking of logs and token usage. |

## 🚀 Enabling Tracing (Elite Style)

Using the `gocrew` SDK, you can enable global tracing with a single line of code. Gocrewwai will automatically generate spans for every agent thought, tool execution, and LLM call:

```go
package main

import (
    "github.com/Ecook14/gocrewwai/gocrew"
    "github.com/Ecook14/gocrewwai/pkg/telemetry"
)

func main() {
    // 1. Initialize OpenTelemetry Exporter (e.g., Honeycomb or Jaeger)
    telemetry.InitTracer("gocrew-app", "http://localhost:4318")

    // 2. Run your Crew (Spans are automatically generated!)
    myCrew.Kickoff(ctx)
}
```

## 🧠 Monitoring Agentic Workflows

### 🏎️ Performance Metrics
Track the execution time and token consumption of every task and agent. This data is essential for optimizing costs and identifying bottlenecks in your orchestration.

### 🛡️ Error Tracing & Retries
When a task fails or a tool errors out, the entire stack trace is captured within the OTEL context, allowing for precise debugging of complex multi-agent failures.

### 📊 Token Usage & Cost Management
Gocrewwai's engine precisely tracks token usage for every single LLM call, providing granular cost data for high-scale production usage.

## 📈 Comparison with LangSmith

| Feature | LangSmith | Gocrewwai Telemetry |
| :--- | :--- | :--- |
| **Hosting** | Proprietary SaaS | Fully self-hostable (OTEL Standard). |
| **Instrumentation** | Python Wrappers | Native, performance-optimized Go. |
| **Trace Format** | Custom JSON | Standardized OTEL Spans & Attributes. |

---

[Back to Self-Correction Guide](./SELF_CORRECTION.md) | [Next: Tool Error Handling](./TOOL_ERROR_HANDLING.md)
