# Feature Deep Dive: Files ⚓📄🖼️

Gocrewwai features advanced, multi-modal file handling capabilities. Agents can read, write, and process a wide range of file formats, including PDF, images, CSV, and structured data, with native support for **Sandboxed File Systems**.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai agents support **Vision-based Processing** for images and PDF tables, with strictly-to-typed extraction logic.

---

## 🏗️ Core File Operations (Elite Style)

Gocrewwai provides specialized tools for common file operations, ensuring that agents can interact with your data safely and efficiently.

| Tool | SDK Constructor | Description |
| :--- | :--- | :--- |
| **File Read** | `gocrew.NewFileReadTool` | Safe, chrooted reading of local text and data files. |
| **PDF Parser** | `gocrew.NewPDFTool` | High-fidelity extraction of text, tables, and metadata. |
| **Directory Scan** | `gocrew.NewDirectoryTool` | Batch processing of entire folder structures. |
| **Vision Browser** | `gocrew.NewBrowserTool` | Captures and analyzes screenshots of web pages. |

## 🚀 Implementing File Handling

Pass the file-related tools to your agent and define a task that requires file interaction:

```go
agent := gocrew.NewAgent(gocrew.AgentConfig{
    Tools: []gocrew.Tool{gocrew.NewFileReadTool(), gocrew.NewPDFTool()},
})

task := gocrew.NewTask(gocrew.TaskConfig{
    Description: "Read the 'report.pdf' and extract all table data into a CSV file.",
    Agent:       agent,
    OutputFile:  "output/results.csv", // Automatic file writing!
})
```

## 🧠 Multi-Modal Vision Processing

If you are using a vision-capable model (like `gpt-4o` or `claude-3.5-sonnet`), Gocrewwai can automatically process image-based data:

1. **Image Injection**: Pass image URLs or base64 data into the agent's context.
2. **Vision Reasoning**: The agent will "see" the image and incorporate it into its reasoning loop.
3. **Table Extraction**: Gocrewwai's PDF tool can leverage vision models to extract complex tables from scanned PDF documents that traditional parsers might miss.

---

[Back to Tools Guide](./tools.md) | [Next: Production](./production.md)
