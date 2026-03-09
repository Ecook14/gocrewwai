# Feature Deep Dive: Knowledge Ingestion 📚

Gocrew allows you to ground your agents in real-world data by feeding them external **Knowledge Sources**. This is the foundation of high-precision RAG (Retrieval-Augmented Generation).

---

## 🏗️ The Knowledge Interface

Gocrew uses a typed source system (`pkg/knowledge`) to parse and vectorize data from various formats.

### Supported Sources
- **StringSource**: For raw text snippets.
- **TextFileSource**: For `.txt` and `.md` files.
- **PDFSource**: For document ingestion.
- **CSVSource / JSONSource**: For structured data.
- **URLSource**: For scraping and parsing websites.

---

## 🛠️ Configuring Knowledge

You can attach knowledge sources to individual agents or the entire crew.

```go
agent := gocrew.NewAgentBuilder().
    KnowledgeSources(
        gocrew.NewURLSource("https://docs.gocrewwai.com"),
        gocrew.NewPDFSource("./manual.pdf"),
    ).
    Build()
```

---

## 🔄 The Ingestion Pipeline

When you add a knowledge source, Gocrew performs the following:

1. **Extraction**: The raw data is parsed (e.g., PDF text extraction).
2. **Chunking**: Large documents are broken into atomic facts or semantic chunks.
3. **Embedding**: Each chunk is converted into a vector embedding using your configured `Embedder`.
4. **Storage**: Chunks are stored in the agent's `Memory` (the vector store).

---

## 🧠 Semantic Retrieval

During task execution, the agent doesn't read the entire knowledge base. Instead:

1. The agent's current task is vectorized.
2. The engine performs a **Similarity Search** against the knowledge store.
3. The most relevant chunks are injected into the agent's context window as "Reference Material."

---

## 📊 Knowledge Events

The ingestion process is observable via `KnowledgeEvent` types (`pkg/events`). You can monitor:
- **IngestionStarted**: When a source begins parsing.
- **IngestionCompleted**: When embeddings are finalized.
- **IngestionError**: If a source fails to parse or vectorize.

---
**Gocrew** - Building agents with real-world expertise.
