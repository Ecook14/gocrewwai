package protocols

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// ---------------------------------------------------------------------------
// WebMCP (Web Model Context Protocol)
// ---------------------------------------------------------------------------
//
// WebMCP extends MCP by allowing standard websites to declare their tools and
// capabilities to AI agents natively in HTML via <script type="application/mcp+json">
// or <meta> tags.
//
// This enables autonomous agents to scrape, discover, and natively execute
// web-forms and endpoints as structured Tools.

// WebMCPDeclaration represents the raw JSON schema pulled from an application/mcp+json tag.
type WebMCPDeclaration struct {
	Version      string                  `json:"version"`
	Capabilities map[string]interface{}  `json:"capabilities"`
	Tools        []WebMCPToolDeclaration `json:"tools"`
}

// WebMCPToolDeclaration is a single tool defined by the remote web page.
type WebMCPToolDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Endpoint    string                 `json:"endpoint"`
	Method      string                 `json:"method"` // GET, POST
	InputSchema map[string]interface{} `json:"inputSchema"` // Standard JSON Schema
}

// WebMCPClient is a lightweight scraper and executor.
type WebMCPClient struct {
	httpClient *http.Client
}

func NewWebMCPClient() *WebMCPClient {
	return &WebMCPClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// DiscoverTools attempts to pull WebMCP declarations from the target URL.
func (c *WebMCPClient) DiscoverTools(ctx context.Context, targetURL string) ([]WebMCPToolDeclaration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("webmcp: invalid url: %w", err)
	}

	req.Header.Set("User-Agent", "Gocrew/1.0 WebMCP Discoverer")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webmcp: failed to fetch document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("webmcp: target returned status %d", resp.StatusCode)
	}

	return c.parseHTMLForMCP(resp.Body, targetURL)
}

// parseHTMLForMCP scans an HTML stream specifically for <script type="application/mcp+json">.
func (c *WebMCPClient) parseHTMLForMCP(r io.Reader, base string) ([]WebMCPToolDeclaration, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("webmcp: failed to parse html body: %w", err)
	}

	var declarations []WebMCPToolDeclaration

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			isMCP := false
			for _, a := range n.Attr {
				if a.Key == "type" && strings.Contains(strings.ToLower(a.Val), "application/mcp+json") {
					isMCP = true
					break
				}
			}

			if isMCP && n.FirstChild != nil {
				var decl WebMCPDeclaration
				if err := json.Unmarshal([]byte(n.FirstChild.Data), &decl); err == nil {
					// Normalize endpoints against the base URL if they are relative
					for i, tool := range decl.Tools {
						if !strings.HasPrefix(tool.Endpoint, "http") {
							// Simple normalizer. In production, use net/url.ResolveReference
							decl.Tools[i].Endpoint = strings.TrimRight(base, "/") + "/" + strings.TrimLeft(tool.Endpoint, "/")
						}
					}
					declarations = append(declarations, decl.Tools...)
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			f(child)
		}
	}
	f(doc)

	return declarations, nil
}

// ExecuteTool attempts to fire the REST/HTTP request constructed by the schema back to the remote endpoint.
func (c *WebMCPClient) ExecuteTool(ctx context.Context, tool WebMCPToolDeclaration, params map[string]interface{}) ([]byte, error) {
	var bodyReader io.Reader
	method := strings.ToUpper(tool.Method)
	
	if method == "" {
		method = http.MethodPost // Default to POST if unspecified
	}

	if method == http.MethodPost || method == http.MethodPut {
		jsonBytes, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("webmcp: failed to marshal params: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, tool.Endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("webmcp: failed to construct tool request: %w", err)
	}

	req.Header.Set("User-Agent", "Gocrew/1.0 WebMCP Executor")
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// For GET requests, append params as query string
	if method == http.MethodGet && len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Add(k, fmt.Sprintf("%v", v))
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webmcp: failed to execute tool endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("webmcp: failed to read tool response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("webmcp: tool endpoint returned status %d. Body: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
