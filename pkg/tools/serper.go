package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// SerperTool uses the Serper.dev API to perform web searches.
type SerperTool struct {
	BaseTool
	APIKey string
}

func NewSerperTool(apiKey string) *SerperTool {
	if apiKey == "" {
		apiKey = os.Getenv("SERPER_API_KEY")
	}
	return &SerperTool{
		BaseTool: BaseTool{
			NameValue:        "SerperSearch",
			DescriptionValue: "Searches the web for a given query and returns search results (titles, snippets, and links).",
		},
		APIKey: apiKey,
	}
}


func (t *SerperTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	query, ok := input["query"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'query' parameter")
	}

	if t.APIKey == "" {
		return "", fmt.Errorf("Serper API key not provided")
	}

	url := "https://google.serper.dev/search"
	payload := map[string]string{"q": query}
	payloadBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	req.Header.Set("X-API-KEY", t.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("search API returned status %d: %s", resp.StatusCode, string(body))
	}

	var results struct {
		Organic []struct {
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
			Link    string `json:"link"`
		} `json:"organic"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return "", fmt.Errorf("failed to decode search results: %w", err)
	}

	if len(results.Organic) == 0 {
		return "No results found for that query.", nil
	}

	var output bytes.Buffer
	output.WriteString(fmt.Sprintf("Search Results for '%s':\n\n", query))
	for i, res := range results.Organic {
		if i >= 5 { // Limit to top 5
			break
		}
		output.WriteString(fmt.Sprintf("%d. %s\nSnippet: %s\nLink: %s\n\n", i+1, res.Title, res.Snippet, res.Link))
	}

	return output.String(), nil
}

func (t *SerperTool) RequiresReview() bool { return false }
