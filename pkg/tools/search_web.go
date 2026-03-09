package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// SearchWebTool provides web search capabilities using DuckDuckGo HTML scraping.
// No API key required — uses the DuckDuckGo HTML endpoint directly.
type SearchWebTool struct {
	BaseTool
	MaxResults int
}

func NewSearchWebTool() *SearchWebTool {
	return &SearchWebTool{
		BaseTool: BaseTool{
			NameValue:        "SearchWebTool",
			DescriptionValue: "Searches the web for information on a given query. Input requires 'query' as a string. Returns relevant search result snippets.",
		},
		MaxResults: 5,
	}
}


func (t *SearchWebTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	queryRaw, ok := input["query"]
	if !ok {
		return "", fmt.Errorf("missing 'query' in input")
	}

	query, ok := queryRaw.(string)
	if !ok {
		return "", fmt.Errorf("'query' must be a string")
	}

	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create search request: %w", err)
	}
	req.Header.Set("User-Agent", "CrewGO/1.0 (Search Tool)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("search returned status code %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read search results: %w", err)
	}

	body := string(bodyBytes)

	// Basic extraction of search result snippets from DuckDuckGo HTML
	results := extractSearchResults(body, t.MaxResults)
	if results == "" {
		return "No search results found for: " + query, nil
	}

	return results, nil
}

func (t *SearchWebTool) RequiresReview() bool { return false }

// extractSearchResults does a simple extraction of result snippets from DuckDuckGo HTML.
func extractSearchResults(html string, maxResults int) string {
	var results []string
	count := 0

	// DuckDuckGo HTML results contain class="result__snippet"
	parts := strings.Split(html, "result__snippet")
	for _, part := range parts[1:] { // Skip the first split (before first result)
		if count >= maxResults {
			break
		}

		// Extract text between > and </
		start := strings.Index(part, ">")
		if start == -1 {
			continue
		}
		end := strings.Index(part[start:], "</")
		if end == -1 {
			continue
		}

		snippet := part[start+1 : start+end]
		snippet = strings.TrimSpace(snippet)
		// Remove HTML tags
		snippet = stripHTMLTags(snippet)

		if snippet != "" {
			count++
			results = append(results, fmt.Sprintf("%d. %s", count, snippet))
		}
	}

	return strings.Join(results, "\n\n")
}

// stripHTMLTags removes basic HTML tags from a string.
func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}
