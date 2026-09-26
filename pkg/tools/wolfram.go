package tools

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

// wolframHTTPClient is a shared client with timeouts for all outbound HTTP calls.
// Using http.DefaultClient or bare http.Get/http.Head would have no connect,
// TLS handshake, or read timeouts — an attacker who controls the URL could hang
// the process indefinitely (DCR-03 / go-static-checks.md).
var wolframHTTPClient = &http.Client{
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
	},
	Timeout: 30 * time.Second,
}

// WolframAlphaTool allows agents to perform complex calculations and queries.
type WolframAlphaTool struct {
	BaseTool
	AppID string
}

func NewWolframAlphaTool(appID string) *WolframAlphaTool {
	if appID == "" {
		appID = os.Getenv("WOLFRAM_APP_ID")
	}
	return &WolframAlphaTool{
		BaseTool: BaseTool{
			NameValue:        "WolframAlphaTool",
			DescriptionValue: "Performs complex calculations and scientific queries via WolframAlpha.",
		},
		AppID: appID,
	}
}

func (t *WolframAlphaTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	query, ok := input["query"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'query'")
	}

	apiURL := fmt.Sprintf("http://api.wolframalpha.com/v1/result?appid=%s&i=%s", t.AppID, url.QueryEscape(query))
	resp, err := wolframHTTPClient.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

func (t *WolframAlphaTool) Name() string        { return t.BaseTool.NameValue }
func (t *WolframAlphaTool) Description() string { return t.BaseTool.DescriptionValue }

// RequiresReview gates outbound queries carrying agent-influenced input.
func (t *WolframAlphaTool) RequiresReview() bool { return true }

var _ Tool = (*WolframAlphaTool)(nil)
