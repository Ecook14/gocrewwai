package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

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
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

func (t *WolframAlphaTool) RequiresReview() bool { return false }
