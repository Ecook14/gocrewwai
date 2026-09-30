package tools

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	maxArxivDecompressedBytes = 2 << 20 // 2MiB cap on expanded response
	maxArxivResults           = 3
	maxArxivOutputBytes       = 64 * 1024
)

// arxivHTTPClient is a shared client with timeouts for all outbound HTTP calls.
// Using http.DefaultClient or bare http.Get/http.Head would have no connect,
// TLS handshake, or read timeouts — an attacker who controls the URL could hang
// the process indefinitely (DCR-03 / go-static-checks.md).
var arxivHTTPClient = &http.Client{
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
	},
	Timeout: 30 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return fmt.Errorf("too many redirects")
		}
		// Stay within the arXiv HTTPS service authority; response-selected
		// redirects must never send the request elsewhere.
		if req.URL.Scheme != "https" || req.URL.Hostname() != "export.arxiv.org" {
			return fmt.Errorf("arxiv redirect outside export.arxiv.org blocked: %s", req.URL.String())
		}
		return nil
	},
}

// ArxivTool allows agents to search for academic papers.
type ArxivTool struct {
	BaseTool
}

func NewArxivTool() *ArxivTool {
	return &ArxivTool{
		BaseTool: BaseTool{
			NameValue:        "ArxivTool",
			DescriptionValue: "Searches arXiv.org for academic papers. Action: search.",
		},
	}
}

func (t *ArxivTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	query, ok := input["query"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'query'")
	}

	apiURL := fmt.Sprintf("https://export.arxiv.org/api/query?search_query=all:%s&start=0&max_results=3", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := arxivHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxArxivDecompressedBytes+1))
	if err != nil {
		return "", fmt.Errorf("failed to read arxiv response: %w", err)
	}
	if len(bodyBytes) > maxArxivDecompressedBytes {
		return "", fmt.Errorf("arxiv response exceeds %d byte budget", maxArxivDecompressedBytes)
	}
	body := string(bodyBytes)

	// Simple extraction of titles and summaries from arXiv XML
	var results []string
	entries := strings.Split(body, "<entry>")
	for _, entry := range entries[1:] {
		if len(results) >= maxArxivResults {
			break
		}
		title := extractTag(entry, "title")
		summary := extractTag(entry, "summary")
		results = append(results, fmt.Sprintf("Title: %s\nSummary: %s", title, summary))
	}

	if len(results) == 0 {
		return "No academic papers found for: " + query, nil
	}

	out := strings.Join(results, "\n---\n")
	if len(out) > maxArxivOutputBytes {
		out = out[:maxArxivOutputBytes] + "\n... [Output Truncated]"
	}
	return out, nil
}

func extractTag(content, tag string) string {
	startTag := "<" + tag + ">"
	endTag := "</" + tag + ">"
	start := strings.Index(content, startTag)
	if start == -1 {
		return ""
	}
	start += len(startTag)
	end := strings.Index(content[start:], endTag)
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(content[start : start+end])
}

func (t *ArxivTool) RequiresReview() bool { return true } // Outbound fetch of agent-influenced URLs

var _ Tool = (*ArxivTool)(nil)
