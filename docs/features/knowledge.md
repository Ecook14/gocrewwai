# Feature Deep Dive: Knowledge (RAG) ⚓📚

Knowledge represents the structured and unstructured data your agents can access to perform their tasks. Unlike memory, which stores personal experiences, knowledge stores your domain-specific data (PDFs, URLs, internal docs) for **Retrieval-Augmented Generation (RAG)**.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai knowledge systems support automatic **Chunking & Vectorization** for PDF, Web, and Text sources with native storage.

---

## 🏗️ Knowledge Sources

Gocrewwai allows you to import data from a wide range of sources via the `knowledge.Source` interface.

| Source | SDK Constructor | Description |
| :--- | :--- | :--- |
| **PDF** | `gocrew.NewPDFSource` | Extracts text and tables from local or remote PDF files. |
| **URL** | `gocrew.NewURLSource` | Scrapes and processes content from any web page. |
| **Text** | `gocrew.NewTextSource` | Imports raw string data. |
| **Directory** | `gocrew.NewDirectorySource` | Batch processes all files within a local directory. |

## 🚀 Adding Knowledge to a Crew (Elite Style)

In Gocrewwai v1.0, knowledge is typically added at the **Crew** level, making it available to all participating agents:

```go
package main

import "github.com/Ecook14/gocrewwai/gocrew"

func main() {
    // 1. Define Knowledge Sources
    pdfSource := gocrew.NewPDFSource("./docs/annual_report.pdf")
    webSource := gocrew.NewURLSource("https://docs.gocrew.ai")

    // 2. Assemble Crew with Knowledge
    myCrew := gocrew.NewCrew(gocrew.CrewConfig{
        Agents:    []gocrew.CoreAgent{researcher},
        Tasks:     []*gocrew.Task{task},
        Knowledge: []gocrew.KnowledgeSource{pdfSource, webSource},
    })
}
```

## 🧠 Advanced Knowledge Retrieval

### 1. Vector Embeddings
Gocrewwai will automatically chunk, embed, and store your knowledge sources using your configured LLM's embedding model.

### 2. Multi-Modal Knowledge
If using a vision-capable model (like `gpt-4o`), Gocrewwai supports extracting information from images and charts within PDF files for the retrieval phase.

### 3. Knowledge Refreshing
You can configure knowledge sources to periodically refresh their data from their original source, ensuring your agents always have access to the latest information.

---

[Back to Memory Guide](./memory.md) | [Next: Planning](./planning.md)
