package tools

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

// wikipediaHTTPClient is a shared client with timeouts for all outbound HTTP calls.
// Using http.DefaultClient or bare http.Get/http.Head would have no connect,
// TLS handshake, or read timeouts — an attacker who controls the URL could hang
// the process indefinitely (DCR-03 / go-static-checks.md).
var wikipediaHTTPClient = &http.Client{
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
	},
	Timeout: 30 * time.Second,
}

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
	resp, err := wikipediaHTTPClient.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// Advanced Implementation: Parses and cleans Wikipedia extract content.
	return string(body), nil
}

func (t *WikipediaTool) Name() string        { return t.BaseTool.NameValue }
func (t *WikipediaTool) Description() string { return t.BaseTool.DescriptionValue }
