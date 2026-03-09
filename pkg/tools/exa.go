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

// ExaTool uses the Exa.ai API (formerly Metaphor) to perform semantic, AI-native searches.
type ExaTool struct {
	BaseTool
	APIKey string
}

func NewExaTool(apiKey string) *ExaTool {
	if apiKey == "" {
		apiKey = os.Getenv("EXA_API_KEY")
	}
	return &ExaTool{
		BaseTool: BaseTool{
			NameValue:        "ExaSearch",
			DescriptionValue: "Performs a semantic search using Exa.ai. Returns high-quality, AI-indexed results including titles, URLs, and text snippets/highlights.",
		},
		APIKey: apiKey,
	}
}


func (t *ExaTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	query, ok := input["query"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'query' parameter")
	}

	if t.APIKey == "" {
		return "", fmt.Errorf("Exa API key not provided")
	}

	url := "https://api.exa.ai/search"
	
	// Exa supports various parameters like useAutoprompt, type, etc.
	payload := map[string]interface{}{
		"query":          query,
		"useAutoprompt":  true,
		"numResults":     5,
		"contents": map[string]interface{}{
			"text": true, // Request text content
		},
	}
	
	payloadBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	req.Header.Set("x-api-key", t.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("exa search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("exa API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Score   float64 `json:"score"`
			Text    string `json:"text"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode exa results: %w", err)
	}

	if len(response.Results) == 0 {
		return "No results found on Exa for that query.", nil
	}

	var output bytes.Buffer
	output.WriteString(fmt.Sprintf("Exa AI Search Results for '%s':\n\n", query))
	for i, res := range response.Results {
		textSnippet := res.Text
		if len(textSnippet) > 300 {
			textSnippet = textSnippet[:300] + "..."
		}
		output.WriteString(fmt.Sprintf("%d. %s\nURL: %s\nRelevance: %.4f\nContent: %s\n\n", i+1, res.Title, res.URL, res.Score, textSnippet))
	}

	return output.String(), nil
}

func (t *ExaTool) RequiresReview() bool { return false }
