package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

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

	apiURL := fmt.Sprintf("http://export.arxiv.org/api/query?search_query=all:%s&start=0&max_results=3", url.QueryEscape(query))
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)

	// Simple extraction of titles and summaries from arXiv XML
	var results []string
	entries := strings.Split(body, "<entry>")
	for _, entry := range entries[1:] {
		title := extractTag(entry, "title")
		summary := extractTag(entry, "summary")
		results = append(results, fmt.Sprintf("Title: %s\nSummary: %s", title, summary))
	}

	if len(results) == 0 {
		return "No academic papers found for: " + query, nil
	}

	return strings.Join(results, "\n---\n"), nil
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

func (t *ArxivTool) RequiresReview() bool { return false }
