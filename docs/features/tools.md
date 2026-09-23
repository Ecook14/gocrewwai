# Feature Deep Dive: Tools ⚓🧰🛠️

Tools are the interface between your agents and the outside world. Gocrewwai agents can use tools to search the web, execute code, browse websites, and interact with external APIs.

---

> [!IMPORTANT]
> **Status: v0.9.0 (Alpha → Beta).** Gocrewwai includes 57 built-in tools with native support for **Docker Sandbox**, **WASM (wazero)**, and **E2B** sandboxing for code execution, plus **Browser Automation** and **MCP** integration.

---

## 🏗️ Built-in Tools (Elite Style)

Gocrewwai includes a rich set of production-ready tools available directly via the `gocrew` SDK.

| Tool | SDK Constructor | Description |
| :--- | :--- | :--- |
|| **SearchWeb** | `gocrew.NewSearchWebTool` | Generic web search (Serper, Google, etc.). |
|| **Exa Search** | `gocrew.NewExaTool` | Vector-indexed neural search for high-quality results. |
|| **Tavily** | `gocrew.NewTavilyTool` | AI-oriented web search with extractive answers. |
|| **Brave** | `gocrew.NewBraveTool` | Brave Search API integration. |
|| **Wikipedia** | `gocrew.NewWikipediaTool` | Extract information from Wikipedia. |
|| **Arxiv** | `gocrew.NewArxivTool` | Search academic papers on Arxiv. |
|| **Browser** | `gocrew.NewBrowserTool` | Automated web navigation and scraping. |
|| **Scraper** | `gocrew.NewScraperTool` | SSRF-protected URL content fetching. |
|| **ScrapeWebsite** | `gocrew.NewScrapeWebsiteTool` | Structured website scraping with SSRF protection. |
|| **Calculator** | `gocrew.NewCalculatorTool` | Precise mathematical operations. |
|| **Code Interpreter** | `gocrew.NewCodeInterpreter(opts...)` | Sandboxed Python/Shell/Go via Docker, WASM (wazero), or E2B. |
|| **Code Sandbox** | `gocrew.NewCodeSandboxTool()` | Isolated Docker container for Python/JS code execution. |
|| **File Read** | `gocrew.NewFileReadTool()` | Read files with extension-based auto-detection. |
|| **File Write** | `gocrew.NewFileWriteTool()` | Write content to files (requires review). |
|| **File Edit** | `gocrew.NewFileEditTool()` | Edit files with old/new string replacement. |
|| **Directory** | `gocrew.NewDirectoryTool()` | List directory contents. |
|| **CSV** | `gocrew.NewCSVTool()` | Read and parse CSV files. |
|| **HTML** | `gocrew.NewHTMLTool()` | Read and parse HTML files. |
|| **XML** | `gocrew.NewXMLTool()` | Read and parse XML files. |
|| **YAML** | `gocrew.NewYamlTool()` | Read and parse YAML files. |
|| **JSON Parse** | `gocrew.NewJSONParseTool()` | Parse and query JSON data. |
|| **PDF** | `gocrew.NewPDFFile(source)` | Extract text from PDF documents. |
|| **Excel** | `gocrew.NewExcelTool()` | Read .xlsx files. |
|| **HTTP Client** | `gocrew.NewHTTPClientTool()` | Make HTTP requests with SSRF protection (requires review). |
|| **Shell** | `gocrew.NewShellTool()` | Execute shell commands with exact-match whitelist (default: deny all). |
|| **AskHuman** | `gocrew.NewAskHumanTool()` | Explicitly prompt for human input mid-task. |
|| **Human Review** | `gocrew.NewHumanReviewGuardrail()` | Guardrail that pauses for manual approval. |
||| **AskQuestion** | `gocrew.NewAskQuestionTool()` | Ask a question to the agent pool (delegation). |
||| **DelegateWork** | `gocrew.NewDelegateWorkTool()` | Delegate a subtask to another agent (delegation). |
|| **GitHub** | `gocrew.NewGitHubTool()` | GitHub API integration. |
|| **GitLab** | `gocrew.NewGitLabTool()` | GitLab API integration. |
|| **Jira** | `gocrew.NewJiraTool()` | Jira API integration. |
|| **Linear** | `gocrew.NewLinearTool()` | Linear API integration. |
|| **Slack** | `gocrew.NewSlackTool()` | Slack API integration. |
|| **Discord** | `gocrew.NewDiscordTool()` | Discord API integration. |
|| **SendGrid** | `gocrew.NewSendGridTool()` | Email sending via SendGrid. |
|| **Notion** | `gocrew.NewNotionTool()` | Notion API integration. |
|| **HubSpot** | `gocrew.NewHubSpotTool()` | HubSpot CRM integration. |
|| **Supabase** | `gocrew.NewSupabaseTool()` | Supabase database integration. |
|| **MongoDB** | `gocrew.NewMongoDBTool()` | MongoDB database integration. |
|| **MySQL** | `gocrew.NewMySQLTool()` | MySQL database integration (requires review). |
|| **PostgreSQL** | `gocrew.NewPostgresTool()` | PostgreSQL database integration (requires review). |
|| **SQLite** | `gocrew.NewSQLiteTool()` | SQLite database integration (requires review). |
|| **Elasticsearch** | `gocrew.NewElasticsearchTool()` | Elasticsearch integration. |
|| **Google Sheets** | `gocrew.NewGoogleSheetsTool()` | Google Sheets API integration. |
|| **Twilio** | `gocrew.NewTwilioTool()` | SMS/voice via Twilio. |
|| **S3** | `gocrew.NewS3Tool()` | AWS S3 object storage. |
|| **WASM Sandbox** | `gocrew.NewWASMSandboxTool()` | WebAssembly-based code sandbox (wazero). |
|| **MCP Bridge** | `gocrew.NewMCPTool()` | Model Context Protocol server bridge. |
|| **WebMCP** | — | Web-based MCP protocol support. |
|| **RagTool** | `gocrew.NewRagTool()` | Retrieval-augmented generation tool. |
|| **DateTime** | `gocrew.NewDateTimeTool()` | Date/time utilities. |
|| **Developer** | `gocrew.NewDeveloperTool()` | Developer utilities. |
|| **Regex** | `gocrew.NewRegexTool()` | Regular expression operations. |
|| **JSON Tool** | `gocrew.NewJSONTool()` | JSON manipulation utilities. |

## 🚀 Using a Tool

Simply pass the tool instances to your agent's configuration:

```go
package main

import "github.com/Ecook14/gocrewwai/gocrew"

func main() {
    searchTool := gocrew.NewSearchWebTool()
    browserTool := gocrew.NewBrowserTool()

    researcher := gocrew.NewAgent(gocrew.AgentConfig{
        Tools: []gocrew.Tool{searchTool, browserTool},
    })
}
```

## 🛡️ Tool Guardrails & Security

### 1. Manual Approval (HITL)
Configure specific tools (e.g., `github.delete_issue`) to require manual approval before execution. The engine will pause and wait for a signal from the **Dashboard**.

### 2. Execution Sandboxing
All code-execution tools (Python, Shell) are isolated from the host system. Gocrewwai supports **Docker**, **WASM**, and **E2B** sandboxing out-of-the-box.

### 3. Tool Error Feedback
If a tool fails, the error message is automatically fed back to the agent's reasoning loop. The agent can then analyze the error, adjust its parameters, and attempt a retry.

---

## 🛠️ Creating Custom Tools

Building a custom tool in Gocrewwai is straightforward. You only need to implement the `tools.Tool` interface. The `gocrew.BaseTool` makes this easy by handling the boilerplate.

**Example: A Custom Weather Tool**

```go
package main

import (
	"context"
	"fmt"
	"github.com/Ecook14/gocrewwai/gocrew"
)

// 1. Define the Expected Input Schema
type WeatherInput struct {
	City  string `json:"city" jsonschema:"description=The name of the city"`
	Units string `json:"units" jsonschema:"enum=metric,enum=imperial"`
}

func NewWeatherTool() gocrew.Tool {
	return gocrew.NewBaseTool(
		"get_weather",
		"Retrieves the current weather for a given city.",
		WeatherInput{}, // Pass an empty struct so the engine can reflect the JSON schema
		func(ctx context.Context, rawInput interface{}) (string, error) {
			
            // 2. Cast the raw input to your struct
			input, ok := rawInput.(*WeatherInput)
			if !ok {
				return "", fmt.Errorf("invalid input type")
			}

			// 3. Execute your business logic
			// (In a real app, you'd make an HTTP call to a weather API here)
			result := fmt.Sprintf("The weather in %s is currently 72 degrees (%s).", 
                                  input.City, input.Units)
            
			return result, nil
		},
	)
}
```

You can now inject `NewWeatherTool()` directly into any `AgentConfig`. The engine will automatically convert your `WeatherInput` struct into a JSON schema that the LLM understands!

---

[Back to Telemetry Guide](./telemetry.md) | [Next: Files](./files.md)
