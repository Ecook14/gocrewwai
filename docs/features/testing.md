# Feature Deep Dive: Testing & Evaluation ⚓🧪🛡️

Reliability is the greatest challenge in agentic AI. Gocrewwai addresses this with a native, high-performance testing and evaluation framework that allows you to measure agent reasoning and mission success at scale.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai testing supports **Multi-run Evaluation**, **Task Scoring**, and **Regreession Tracing** via OpenTelemetry.

---

## 🏗️ Core Testing Concepts

### 1. Multi-run Evaluation
To account for LLM non-determinism, Gocrewwai's testing engine can execute the same crew multiple times and aggregate the results. This provides a statistically significant measure of performance and reliability.

### 2. Task Scoring & Grading
Define custom "Grading Agents" whose sole job is to review and score the output of your primary agents. Gocrewwai provides a standard `pkg/testing` module for defining these evaluation criteria.

### 3. Trace Comparisons
By leveraging the native **OpenTelemetry** integration, you can compare the reasoning traces of different crew runs. This allows you to identify exactly where an agentic workflow "drifted" from the expected path.

---

## 🚀 Implementing Agent Tests (Elite Style)

Using the `gocrew` SDK, you can define and run evaluations with ease:

```go
package main

import (
    "github.com/Ecook14/gocrewwai/gocrew"
    "github.com/Ecook14/gocrewwai/pkg/testing"
)

func main() {
    // 1. Define the Evaluation Criteria
    eval := testing.NewEvaluator(testing.EvaluatorConfig{
        Agents:    []gocrew.CoreAgent{tester},
        Criteria:  "The summary must be exactly 3 bullet points long.",
        NumRuns:   10, // Run the crew 10 times
    })

    // 2. Run the Test
    results := eval.Run(myCrew)
    fmt.Printf("Average Score: %.2f%%\n", results.AverageScore)
}
```

## 🛡️ Production Safety & Regressions

### 1. CI/CD Integration
The `gocrew` CLI supports a `test` command that can be integrated into your CI/CD pipeline. These tests can fail the build if agentic performance drops below a critical threshold.

### 2. Regression Tracking
Store your evaluation results in a persistent database to track performance over time as you update your agent backstories or LLM models. Gocrewwai supports exporting these metrics directly to **Grafana** or **Datadog** via OTEL.

---

[Back to CLI Guide](./cli.md) | [Next: Training](./training.md)
