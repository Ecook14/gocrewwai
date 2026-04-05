# Feature Deep Dive: Telemetry ⚓📊⛓️

Gocrewwai is built with production-grade observability from the ground up. Our framework features native **OpenTelemetry (OTEL)** instrumentation, allowing you to trace every aspect of your agentic orchestration with high fidelity and standardized tools.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai telemetry follows **W3C Trace Context** standards and supports automatic propagation across **A2A (Agent-to-Agent)** mesh networks.

---

## 🏗️ The Telemetry Architecture

Gocrewwai's telemetry system is decentralized and vendor-neutral.

| Component | Implementation | Key Advantage |
| :--- | :--- | :--- |
| **Tracer** | `pkg/telemetry` | Standardized, low-overhead Go instrumentation. |
| **Collector** | HTTP/gRPC/OTLP | Export traces to Jaeger, Honeycomb, Zipkin, or OpenSearch. |
| **Auto-Spans** | Native in Engine/SDK | Zero-code setup for tracing entire multi-agent lifecycles. |
| **Attribute Set** | `semconv` (Semantic Conventions) | Industry-standard metadata for agent role, task, and tool usage. |

---

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

## 🧠 Monitoring Agentic Behaviors

### 1. The Reasoning Trace
Trace exactly how an agent moves from a task assignment to a final result. Every internal "Thought," "Action," and "Observation" is captured as a distinct, nested span, allowing you to debug complex reasoning failures.

### 2. Tool Performance Metrics
Monitor the latency and success rate of your tools. Identify slow-performing search APIs or failing code execution sandboxes by analyzing the spans in your tracing backend.

### 3. Distributed A2A Tracing
If you are running a distributed crew across multiple servers, Gocrewwai automatically propagates the trace context through its native A2A protocols, providing you with a single, unified view of the entire global orchestration.

---

[Back to Events Guide](./events.md) | [Next: Tools](./tools.md)
