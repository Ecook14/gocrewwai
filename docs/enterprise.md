# Gocrewwai Enterprise Edition

High-performance, enterprise-grade agentic orchestration for the Go ecosystem. Designed for organizations that demand security, scalability, and production reliability.

---

## Enterprise Features

### Security & Compliance
- **MIT License** — freely usable in commercial products
- **WASM Sandbox** — isolate untrusted code execution
- **Audit Logging** — track every agent action and tool call
- **Guardrails** — 8 built-in guardrail types (PII redaction, toxicity, schema validation, etc.)
- **Human-in-the-Loop** — manual approval gates for sensitive operations
- **API Key Rotation** — support for multiple API keys with automatic failover

### Scalability & Performance
- **Go-native concurrency** — hundreds of agents running truly in parallel (no GIL)
- **Sub-15ms startup** — compiled binary, no interpreter warmup
- **10-50MB memory** — orders of magnitude less than Python frameworks
- **Single binary deployment** — compile once, run anywhere (Linux, macOS, Windows)
- **Horizontal scaling** — stateless agent execution, easy to cluster

### Integration & Interoperability
- **7 LLM providers** — OpenAI, Anthropic, Gemini, Groq, OpenRouter, Ollama, Failover
- **40+ built-in tools** — spanning SaaS, databases, code execution, web, search
- **MCP Protocol** — connect to any MCP-compatible tool server
- **A2A Protocol** — agent-to-agent communication with discovery
- **OpenTelemetry** — vendor-neutral observability, export to any OTEL backend

### Production Readiness
- **Checkpointing** — pause, resume, and time-travel through agent workflows
- **Rate limiting** — per-key and global rate limiting with token bucket algorithm
- **Retry with backoff** — automatic retry on transient failures
- **Multi-provider failover** — automatically switch LLM providers on failure
- **Structured logging** — JSON-formatted logs for log aggregation systems
- **Metrics** — Prometheus-compatible metrics endpoint

---

## Quick Start (Enterprise)

```bash
# Install
go get github.com/Ecook14/gocrewwai/gocrew

# Initialize a crew
cat > crew.yaml << EOF
agents:
  - role: Researcher
    goal: Find information about X
    llm:
      provider: openai
      model: gpt-4o
tasks:
  - description: Research X and report findings
    agent: Researcher
EOF

# Run
gocrew run --config crew.yaml
```

---

## Security Best Practices

1. **Never hardcode secrets** — use environment variables or secret managers
2. **Enable guardrails** — at minimum, enable PII redaction and content filtering
3. **Use HITL for mutations** — require human approval for tools that write to external systems
4. **Sandbox untrusted code** — use the WASM sandbox for any code execution tool
5. **Monitor and audit** — enable OpenTelemetry and review logs regularly
6. **Rotate credentials** — rotate API keys every 90 days
7. **Limit tool access** — only give agents the tools they need for their specific task

---

## Support

- **Documentation:** [docs/](docs/)
- **GitHub Issues:** https://github.com/Ecook14/gocrewwai/issues
- **Security:** See [SECURITY.md](SECURITY.md)
- **License:** MIT — see [LICENSE](LICENSE)
