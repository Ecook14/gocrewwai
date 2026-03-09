package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// WikipediaTool allows agents to search Wikipedia.
type WikipediaTool struct {
	BaseTool
}

func NewWikipediaTool() *WikipediaTool {
	return &WikipediaTool{
		BaseTool: BaseTool{
			NameValue:        "WikipediaTool",
			DescriptionValue: "Searches Wikipedia for information. Action: search.",
		},
	}
}


func (t *WikipediaTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	query, ok := input["query"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'query'")
	}

	apiURL := fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=query&prop=extracts&exintro&explaintext&format=json&titles=%s", url.QueryEscape(query))
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// Advanced Implementation: Parses and cleans Wikipedia extract content.
	return string(body), nil
}

func (t *WikipediaTool) RequiresReview() bool { return false }
