# Feature: Telemetry & Enterprise Observability

## Overview
Unlike experimental scripts that output to `stdout`, Crew-GO is instrumented internally with a **Global Async Event Bus** and **OpenTelemetry (OTEL)** traces (`pkg/telemetry`). 

## Key Capabilities

1. **The Event Bus (`telemetry.GlobalBus`)**
   Every critical stage of execution emits a typed event:
   - `EventAgentStarted`, `EventAgentThinking`, `EventAgentFinished`
   - `EventToolStarted`, `EventToolFinished`
   - `EventError`

   You can subscribe natively in Go to pipe these into your own logging endpoints or Kafka streams.

2. **OpenTelemetry (OTEL) Tracing**
   Crew-GO natively builds trace spans across functions. A single `crew.Kickoff()` will trace:
   - The Crew Total Execution
   - Parallel Agent Task Executions (as sub-spans)
   - Specific Tool Executions (as leaf spans)
   
   These traces can be instantly exported to **Datadog, Jaeger, Honeycomb, or Grafana** using standard OTEL exporters.

3. **Usage Metrics & Token Cost Tracking**
   Every Agent and the Crew itself holds a `UsageMetrics` map. Every LLM call computes the exact number of Prompt Tokens and Completion Tokens consumed. At the end of a `Crew` kickoff, a summation is provided natively, allowing strict SLA monitoring and cost accounting inside enterprise infrastructure.

4. **The Live Glassmorphic Dashboard**
   Using `--ui` bridges the internal Event Bus directly to a WebSocket server, visualizing the OTEL traces and Event payloads in a beautiful, real-time React/VanillaJS dashboard out-of-the-box.
