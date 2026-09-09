package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// TavilyTool provides AI-powered web search using Tavily API.
type TavilyTool struct {
	BaseTool
	APIKey string
}

// NewTavilyTool creates a Tavily search tool.
func NewTavilyTool(apiKey string) *TavilyTool {
	if apiKey == "" {
		apiKey = os.Getenv("TAVILY_API_KEY")
	}
	return &TavilyTool{
		BaseTool: BaseTool{
			NameValue:        "TavilySearch",
			DescriptionValue: "AI-powered web search with structured results including summaries, scores, and links.",
		},
		APIKey: apiKey,
	}
}

// tavilyResponse represents the Tavily search API response.
type tavilyResponse struct {
	Query     string              `json:"query"`
	Results   []tavilyResult      `json:"results"`
	TotalResults int             `json:"total_results"`
	ExecutionTime float64         `json:"execution_time"`
}

type tavilyResult struct {
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Snippet     string  `json:"snippet"`
	Summary     string  `json:"content"`
	Score       float64 `json:"score"`
	PublishedAt string  `json:"published_date"`
	Site        string  `json:"site"`
}

// Execute implements the Tool interface for Tavily search.
func (t *TavilyTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	query, ok := input["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("missing 'query' parameter")
	}

	maxResults := 10
	if mr, ok := input["max_results"].(int); ok && mr > 0 {
		maxResults = mr
	}
	searchType := "research"
	if st, ok := input["search_type"].(string); ok {
		searchType = st
	}
	includeDomains := ""
	if d, ok := input["include_domains"].(string); ok {
		includeDomains = d
	}

	url := fmt.Sprintf("https://api.tavily.com/search?query=%s&api_key=%s&max_results=%d&search_depth=%s",
		strings.ReplaceAll(query, " ", "+"), t.APIKey, maxResults, searchType)

	if includeDomains != "" {
		url += "&include_domains=" + includeDomains
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("tavily search failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tavily returned status %d", resp.StatusCode)
	}

	var tavResp tavilyResponse
	if err := json.NewDecoder(resp.Body).Decode(&tavResp); err != nil {
		return "", fmt.Errorf("failed to parse tavily response: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Tavily Search Results for: %s\n\n", query))
	sb.WriteString(fmt.Sprintf("Total Results: %d\n\n", tavResp.TotalResults))

	for i, r := range tavResp.Results {
		sb.WriteString(fmt.Sprintf("-- [%d] %s (Score: %.2f) --\n", i+1, r.Title, r.Score))
		sb.WriteString(fmt.Sprintf("URL: %s\n", r.URL))
		sb.WriteString(fmt.Sprintf("Site: %s\n", r.Site))
		if r.PublishedAt != "" {
			sb.WriteString(fmt.Sprintf("Published: %s\n", r.PublishedAt))
		}
		sb.WriteString(fmt.Sprintf("Summary: %s\n\n", r.Summary))
	}

	return sb.String(), nil
}

// RequiresReview returns false - search is safe.
func (t *TavilyTool) RequiresReview() bool { return false }

// Name returns the tool name.
func (t *TavilyTool) Name() string { return t.BaseTool.NameValue }

// Description returns the tool description.
func (t *TavilyTool) Description() string { return t.BaseTool.DescriptionValue }
