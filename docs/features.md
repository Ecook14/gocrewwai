# Feature Documentation

## Complete Feature Matrix

### Agent Framework
- Role-based agents with goal-driven execution
- Multi-agent crews (Sequential, Hierarchical, Consensual, Graph, Reflective, StateMachine — 6 process types, see `pkg/crew/crew.go:33-38`)
- Agent delegation and sub-agent creation
- Agent memory (short-term, long-term, entity)
- Recursive self-correction
- Human-in-the-Loop (HITL) interrupts
- Rate limiting (MaxRPM, MaxTokens)

### Tool Ecosystem (52 tools)
Verifiable: `grep -h "^func New.*Tool" pkg/tools/*.go` (minus `NewToolRegistry`).
- Search: Tavily, Brave, Exa, Serper, Wikipedia, Arxiv
- Databases: Supabase, PostgreSQL, MySQL, MongoDB, Elasticsearch, SQLite
- CRM: HubSpot
- Project Management: Jira, Linear
- Communication: Discord, Slack, SendGrid, Twilio, Email
- Spreadsheets: Google Sheets, Excel, CSV
- Documents: PDF, HTML, XML, YAML, JSON
- Local: File Read/Write/Edit, Directory, Code Interpreter, Shell, DateTime, Calculator
- Web: Browser, Scraper, HTTP, Notion, GitHub, S3, Wolfram
- AI/Human: Ask Human, Ask Question, Native RAG, WASM Sandbox, Code Sandbox
- Not yet implemented (roadmap): Snowflake, Salesforce, Asana, Trello, Redis-tool, Telegram/Teams/WhatsApp, GitLab CI, HuggingFace, Airtable

### LLM Providers (6+1)
- **6 native clients + 1 composite** — OpenAI, Anthropic, Gemini, Groq, OpenRouter, Ollama + Failover wrapper. Note: `pkg/config` registry `GetClient` currently wires openai/anthropic/google/openrouter only (groq/ollama via direct constructors).

### Memory & RAG (12 backends)
- SQLite, Redis, Chroma, Pinecone, Qdrant, Weaviate, InMemCosine, Conversation, Entity, ShortTerm, LongTerm, Unified

### Protocols
- MCP, A2A, gRPC, WebSocket

### Infrastructure
- Sandbox: Docker, WASM (wazero)
- Server: Production HTTP API, Dashboard
- CLI: gocrew with create, run, train, test, replay, chat, reset-memories, kickoff, version
- CI/CD: GitHub Actions
- Docker: Multi-stage build
- Observability: OpenTelemetry
