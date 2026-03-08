# Feature Deep Dive: Tool Arsenal & Sandboxing 🧰

Hey there! Let's talk about how Gocrew agents actually interact with the real world. 

An LLM that can only generate text is just a fancy chatbot. To build autonomous agents that hold real value, we have to give them `Tools`. 

Gocrew breaks the mold of minimal-tool frameworks by providing **over 24 native integrations** right out of the box inside `pkg/tools`. These are completely type-safe Go implementations engineered for speed, safety, and reliability.

---

## 🛡️ Elite Multi-modal Execution Sandboxes

The most powerful capability you can give an AI is the ability to write and execute its own code (e.g., using the `CodeInterpreterTool`). However, running untrusted, AI-generated scripts using Go's `os/exec` natively on your production server is an enormous security risk.

Gocrew solves this by providing tiered, highly-controlled sandboxing environments.

### 1. The Docker Ephemeral Engine
This is the gold standard for heavy code execution.
- Dynamically pulls target isolated containers (e.g., `python:3.11-slim` or `node:18`).
- Allows you to strictly bound CPU limits and Memory limits (`tools.WithLimits(512, 1024)`).
- Executes the agent-generated code seamlessly inside the container and securely pipes `stdout/stderr` back to the Go engine.

### 2. WASM (wazero) local-isolation
If you don't want the heavy overhead of maintaining Docker daemons, we built something incredible: WebAssembly isolation.
- Compiles and executes WASM *directly inside the Go runtime* using `wazero`.
- Zero external dependencies.
- Total memory isolation from the host system, with instant startup times (nanoseconds compared to Docker's milliseconds).
- You explicitly pass virtual filesystems to the agent: it can only read the `memfs` or directories you explicitly mount!

---

## 🎒 The Built-In Tool Capabilities

Want your agents to do more than write code? Here is a taste of the native tools you can hand them:

### 🕸️ Web Browsing & RAG Document Extraction
- **`BrowserTool`**: Need to scrape a complex React Single Page Application? This tool spins up a full `Chromedp` headless browser to natively execute Javascript and click buttons on behalf of the agent!
- **`SearchWebTool` & `SerperTool`**: For lightning-fast Google/DuckDuckGo searches.
- **Native Document Extractor**: Point the agent at a PDF or a `.docx` Microsoft Word file, and the Go engine parses it natively without needing external API credits!

### 💾 Databases & File Logic
- **`PostgresTool` / `MySQLTool` / `MongoDBTool` / `ElasticSearchTool`**: Give the agent a Database connection string, and it will autonomously query your live data to answer questions.
- **`FileOpsTool`**: Allow the agent to safely read, write, and manipulate local text files.

### 💬 Productivity & Communications
- **`SlackTool`**: Allow the agent to broadcast its final reports to a Slack channel.
- **`GitHubTool`**: Allow the agent to read issues, review pull requests, and commit code autonomously.

---

## 🏥 Self-Healing Tools

APIs fail. HTML layouts change. SQL syntax gets mistyped. In legacy scripts, this crashes the program.

In Gocrew, tools are **Self-Healing**. 
If a tool encounters an error (e.g., the agent writes a bad SQL query with a syntax error):
1. The Go error is securely captured by the engine.
2. It is appended to the Agent's context explicitly: *"Fatal Tool Error: syntax error at or near 'SELECTT'. Please correct your parameters and try again."*
3. The Agent autonomously reads the error, corrects its JSON payload, and fires the tool again successfully!

---

## 🤝 Let's Build More Tools Together!

The best part about Gocrew is how easy it is to add new capabilities. A tool just needs to satisfy the `tools.Tool` Go interface (provide a Name, Description, and an `Execute` method).

Are you integrating with a specific SaaS product at work like Jira, Salesforce, or Datadog? 
**Please consider writing a Tool for it and submitting a Pull Request!** 

Let's build the largest, safest, most expansive native tool library in the Go ecosystem together!
