# Feature Deep Dive: Automated Testing 🧪

Gocrew provides a specialized **Testing Framework** (`pkg/testing`) designed for evaluating agentic workflows, where output is non-deterministic and performance must be measured over multiple runs.

---

## 🏗️ Multi-Run Evaluation

Traditional unit tests are often insufficient for LLMs. Gocrew's testing system performs **Multi-Run Benchmarking**:

1. **Iteration**: The test runner executes the same crew/task `N` times (e.g., 10 iterations).
2. **Metrics Collection**: It captures latency, token usage, and tool success rates for every run.
3. **Objective Evaluation**: A specialized `TestLLM` (usually a powerful model like GPT-4o) acts as a "Grader," scoring the output based on your specific criteria.
4. **Statistical Aggregation**: The framework calculates the average score, variance, and failure rate across all iterations.

---

## 🛠️ Running Tests via CLI

The easiest way to test your crew is via the CLI.

```bash
# Run 10 iterations of the crew and evaluate performance
gocrew test --n 10 --model gpt-4o
```

---

## 📊 Performance Metrics

Gocrew tests output a detailed JSON report including:
- **Score (0-10)**: The LLM-graded quality of the final result.
- **Consistency**: How much the results varied across runs.
- **RPM/TPM Usage**: Real-world resource consumption.
- **Tool Reliability**: Frequency of tool call failures or retries.

---

## 🧩 Programmatic Testing

You can also integrate Gocrew tests into your existing Go test suites.

```go
tester := testing.NewTester(myCrew)
report, err := tester.Run(ctx, 5) // Run 5 iterations
if report.AverageScore < 8.0 {
    t.Errorf("Agent performance below threshold: %f", report.AverageScore)
}
```

---
**Gocrew** - Data-driven confidence in your agentic workflows.
