package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// BraveSearchTool provides private, privacy-first web search using Brave Search API.
type BraveSearchTool struct {
	BaseTool
	APIKey string
}

// NewBraveSearchTool creates a Brave Search tool.
func NewBraveSearchTool(apiKey string) *BraveSearchTool {
	if apiKey == "" {
		apiKey = os.Getenv("BRAVE_SEARCH_API_KEY")
	}
	return &BraveSearchTool{
		BaseTool: BaseTool{
			NameValue:        "BraveSearchTool",
			DescriptionValue: "Private AI-powered web search using Brave Search. No tracking, no ads.",
		},
		APIKey: apiKey,
	}
}

type braveResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Snippet string  `json:"description"`
	Score   float64 `json:"score"`
}

// Execute implements the Tool interface for Brave Search.
func (b *BraveSearchTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	query, ok := input["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("missing 'query' parameter")
	}

	count := 10
	if c, ok := input["count"].(int); ok && c > 0 {
		count = c
	}

	url := fmt.Sprintf("https://api.search.brave.com/res/search?q=%s&count=%d",
		strings.ReplaceAll(query, " ", "+"), count)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", b.APIKey)

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil { return "", fmt.Errorf("Brave search failed: %w", err) }
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Brave returned status %d", resp.StatusCode)
	}

	var br struct {
		Query   string         `json:"query"`
		Results []braveResult  `json:"results"`
		Total int            `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&br)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Brave Search Results for: %s\n\n", query))
	sb.WriteString(fmt.Sprintf("Total Results: %d\n\n", br.Total))

	for i, r := range br.Results {
		sb.WriteString(fmt.Sprintf("-- [%d] %s --\n", i+1, r.Title))
		sb.WriteString(fmt.Sprintf("URL: %s\n", r.URL))
		sb.WriteString(fmt.Sprintf("Snippet: %s\n\n", r.Snippet))
	}

	return sb.String(), nil
}

func (b *BraveSearchTool) RequiresReview() bool { return false }
func (b *BraveSearchTool) Name() string { return b.BaseTool.NameValue }
func (b *BraveSearchTool) Description() string { return b.BaseTool.DescriptionValue }
