# Feature Deep Dive: File Handling 📂

Gocrew provides a sophisticated, multi-modal file system designed to bridge the gap between local storage and LLM context windows.

---

## 🏗️ Typed File Objects

In Gocrew, files are not just strings. They are structured objects that understand their own content and constraints.

### Supported File Types
- **TextFile**: Standard `.txt`, `.md`, `.go`, `.py` files.
- **PDFFile**: Handled via local parsers or LLM vision capabilities.
- **ImageFile**: Supports JPEG, PNG (with automatic Base64 encoding for vision models).
- **VideoFile / AudioFile**: Managed via provider-specific upload URI systems (e.g., Gemini's File API).

---

## 🛠️ Attaching Files to Tasks

You can provide files directly to a task context. Gocrew handles the heavy lifting of reading, chunking, or uploading based on the target LLM.

```go
task := gocrew.NewTaskBuilder().
    Description("Analyze this invoice.").
    Files(gocrew.NewPDFFile("./invoice.pdf")).
    Build()
```

---

## 🛡️ Provider Constraints

Different LLMs have different rules for files. Gocrew handles these automatically:
- **OpenAI**: Automatically converts images to Base64 data URIs.
- **Gemini**: Uses the specialized Gemini File API for large videos/PDFs to prevent context window bloat.
- **Anthropic**: Manages document blocks according to the latest Claude specifications.

---

## 🧹 Auto-Cleanup

By default, temporary files or remote uploads are managed by Gocrew's `pkg/files` lifecycle controller. You can configure retention policies or manual cleanup if needed.

---
**Gocrew** - Seamless multi-modal data orchestration.
