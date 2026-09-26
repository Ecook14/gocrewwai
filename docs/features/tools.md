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
|| **SearchWeb** | `gocrew.NewSearchWebTool()` | Generic web search. |
|| **Exa Search** | `gocrew.NewExaTool(apiKey)` | Vector-indexed neural search for high-quality results. |
|| **Tavily** | `gocrew.NewTavilyTool(apiKey)` | AI-oriented web search with extractive answers. |
|| **Brave** | `gocrew.NewBraveTool(apiKey)` | Brave Search API integration. |
|| **Wikipedia** | `gocrew.NewWikipediaTool()` | Extract information from Wikipedia. |
|| **Arxiv** | `gocrew.NewArxivTool()` | Search academic papers on Arxiv. |
|| **Browser** | `gocrew.NewBrowserTool()` | Automated web navigation and scraping. |
|| **Scraper** | `gocrew.NewScraperTool()` | SSRF-protected URL content fetching. |
|| **ScrapeWebsite** | `gocrew.NewScrapeWebsiteTool()` | Structured website scraping with SSRF protection. |
|| **Calculator** | `gocrew.NewCalculatorTool()` | Precise mathematical operations. |
|| **Code Interpreter** | `gocrew.NewCodeInterpreter(safe)` | Sandboxed Python/Shell/Go via Docker, WASM (wazero), or E2B. |
|| **Code Sandbox** | `gocrew.NewCodeSandboxTool()` | Isolated Docker container for Python/JS code execution. |
|| **File Read** | `gocrew.NewFileReadTool(chroot)` | Read files with extension-based auto-detection. |
|| **File Write** | `gocrew.NewFileWriteTool(chroot)` | Write content to files (requires review). |
|| **File Edit** | `gocrew.NewFileEditTool(chroot)` | Edit files with old/new string replacement. |
|| **Directory** | `gocrew.NewDirectoryTool(rootPath, maxDepth, allowAbsolute)` | List directory contents. |
|| **CSV** | `gocrew.NewCSVTool(chroot)` | Read and parse CSV files. |
|| **HTML** | `gocrew.NewHTMLTool(chroot)` | Read and parse HTML files. |
|| **XML** | `gocrew.NewXMLTool(chroot)` | Read and parse XML files. |
|| **YAML** | `gocrew.NewYamlTool(chroot)` | Read and parse YAML files. |
|| **JSON Parse** | `gocrew.NewJSONParseTool(chroot)` | Parse and query JSON data. |
|| **PDF** | `gocrew.NewPDFFile(chroot)` | Extract text from PDF documents. |
|| **Excel** | `gocrew.NewExcelTool(chroot)` | Read .xlsx files. |
|| **HTTP Client** | `gocrew.NewHTTPClientTool(opts...)` | Make HTTP requests with SSRF protection (requires review). |
|| **Shell** | `gocrew.NewShellTool(opts...)` | Execute shell commands with exact-match whitelist (default: deny all). |
|| **AskHuman** | `gocrew.NewAskHumanTool(enabled)` | Explicitly prompt for human input mid-task. |
|| **Human Review** | `gocrew.NewHumanReviewGuardrail()` | Guardrail that pauses for manual approval. |
||| **AskQuestion** | `gocrew.NewAskQuestionTool()` | Ask a question to the agent pool (delegation). |
||| **DelegateWork** | `gocrew.NewDelegateWorkTool(coworkers)` | Delegate a subtask to another agent (delegation). |
|| **GitHub** | `gocrew.NewGitHubTool(token)` | GitHub API integration. |
|| **Jira** | `gocrew.NewJiraTool(baseURL, email, token)` | Jira API integration. |
|| **Linear** | `gocrew.NewLinearTool(token)` | Linear API integration. |
|| **Slack** | `gocrew.NewSlackTool(token)` | Slack API integration. |
|| **Discord** | `gocrew.NewDiscordTool(botToken, channelID)` | Discord API integration. |
|| **SendGrid** | `gocrew.NewSendGridTool(apiKey)` | Email sending via SendGrid. |
|| **Notion** | `gocrew.NewNotionTool(token)` | Notion API integration. |
|| **HubSpot** | `gocrew.NewHubSpotTool(token)` | HubSpot CRM integration. |
|| **Supabase** | `gocrew.NewSupabaseTool(url, apiKey, table)` | Supabase database integration. |
|| **MongoDB** | `gocrew.NewMongoDBTool(endpoint, apiKey, dataSource, database)` | MongoDB database integration. |
|| **MySQL** | `gocrew.NewMySQLTool(dsn)` | MySQL database integration (requires review). |
|| **PostgreSQL** | `gocrew.NewPostgresTool(connStr)` | PostgreSQL database integration (requires review). |
|| **SQLite** | `gocrew.NewSQLiteTool(dbPath)` | SQLite database integration (requires review). |
|| **Elasticsearch** | `gocrew.NewElasticsearchTool(baseURL)` | Elasticsearch integration. |
|| **Google Sheets** | `gocrew.NewGoogleSheetsTool(token)` | Google Sheets API integration. |
|| **Twilio** | `gocrew.NewTwilioTool(accountSID, authToken)` | SMS/voice via Twilio. |
|| **S3** | `gocrew.NewS3Tool(endpoint, accessKey, secretKey, region)` | AWS S3 object storage. |
|| **WASM Sandbox** | `gocrew.NewWASMSandboxTool(ctx)` | WebAssembly-based code sandbox (wazero). |
|| **MCP Bridge** | `protocols.NewMCPClient(serverURL)` | Model Context Protocol server bridge (via `pkg/protocols`). |
|| **WebMCP** | — | Web-based MCP protocol support. |
|| **RagTool** | `gocrew.NewRagTool(mem, dir)` | Retrieval-augmented generation tool. |
|| **DateTime** | `gocrew.NewDateTimeTool()` | Date/time utilities. |
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

// 1. Embed gocrew.BaseTool for the Name/Description/ArgsSchema boilerplate.
type WeatherTool struct {
	gocrew.BaseTool
}

func NewWeatherTool() gocrew.Tool {
	return &WeatherTool{
		BaseTool: gocrew.BaseTool{
			NameValue:        "get_weather",
			DescriptionValue: "Retrieves the current weather for a given city.",
			Schema: []gocrew.ArgSchema{
				{Name: "city", Type: "string", Description: "The name of the city.", Required: true},
				{Name: "units", Type: "string", Description: "metric or imperial.", Required: false},
			},
		},
	}
}

// 2. Implement Execute — the only method you must write yourself.
func (t *WeatherTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	city, _ := input["city"].(string)
	units, _ := input["units"].(string)

	// 3. Execute your business logic
	// (In a real app, you'd make an HTTP call to a weather API here)
	return fmt.Sprintf("The weather in %s is currently 72 degrees (%s).", city, units), nil
}
```

You can now inject `NewWeatherTool()` directly into any `AgentConfig`. The engine will automatically convert your `WeatherInput` struct into a JSON schema that the LLM understands!

---

[Back to Telemetry Guide](./telemetry.md) | [Next: Files](./files.md)
