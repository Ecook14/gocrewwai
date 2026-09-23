# Feature Documentation

## Complete Feature Matrix

### Agent Framework
- Role-based agents with goal-driven execution
- Multi-agent crews (Sequential, Hierarchical, Graph, Reflective)
- Agent delegation and sub-agent creation
- Agent memory (short-term, long-term, entity)
- Recursive self-correction
- Human-in-the-Loop (HITL) interrupts
- Rate limiting (MaxRPM, MaxTokens)

### Tool Ecosystem (57 tools)
- Search: Tavily, Brave, Exa, Serper
- Databases: Supabase, Snowflake, PostgreSQL, MySQL, Redis
- CRM: HubSpot, Salesforce
- Project Management: Jira, Linear, Asana, Trello
- Communication: Discord, Slack, SendGrid, Twilio
- Spreadsheets: Google Sheets, Excel
- Local: Ollama, File Read/Write/Edit, Code Interpreter

### LLM Providers (7)
- **7 LLM providers** — OpenAI, Anthropic, Gemini, Groq, OpenRouter, Ollama, Failover

### Memory & RAG (7 backends)
- SQLite, Redis, Chroma, Pinecone, Qdrant, Weaviate

### Protocols
- MCP, A2A, gRPC, WebSocket

### Infrastructure
- Sandbox: Docker, WASM (wazero)
- Server: Production HTTP API, Dashboard
- CLI: gocrew with init, new, run, serve
- CI/CD: GitHub Actions
- Docker: Multi-stage build
- Observability: OpenTelemetry
