package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ElasticsearchTool allows agents to search and index documents in Elasticsearch.
//
// Input examples:
//
//	{"action": "search", "index": "logs", "query": {"match": {"message": "error"}}}
//	{"action": "index", "index": "logs", "id": "1", "document": {"message": "test", "level": "info"}}
//	{"action": "get", "index": "logs", "id": "1"}
//	{"action": "delete", "index": "logs", "id": "1"}
type ElasticsearchTool struct {
	BaseTool
	BaseURL    string
	Username   string // Basic auth (optional)
	Password   string
	APIKey     string // API key auth (optional, takes precedence)
	httpClient *http.Client
}

// NewElasticsearchTool creates a new Elasticsearch client tool.
func NewElasticsearchTool(baseURL string, opts ...func(*ElasticsearchTool)) *ElasticsearchTool {
	t := &ElasticsearchTool{
		BaseTool: BaseTool{
			NameValue:        "ElasticsearchTool",
			DescriptionValue: "Search and manage documents in Elasticsearch. Actions: search, index, get, delete, count. Input: {'action': '...', 'index': '...', 'query': {...}}",
		},
		BaseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func WithESBasicAuth(username, password string) func(*ElasticsearchTool) {
	return func(t *ElasticsearchTool) {
		t.Username = username
		t.Password = password
	}
}

func WithESAPIKey(apiKey string) func(*ElasticsearchTool) {
	return func(t *ElasticsearchTool) {
		t.APIKey = apiKey
	}
}

func (t *ElasticsearchTool) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, t.BaseURL+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	if t.APIKey != "" {
		req.Header.Set("Authorization", "ApiKey "+t.APIKey)
	} else if t.Username != "" {
		req.SetBasicAuth(t.Username, t.Password)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (t *ElasticsearchTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	action, _ := input["action"].(string)
	index, _ := input["index"].(string)
	if action == "" || index == "" {
		return "", fmt.Errorf("'action' and 'index' are required")
	}

	switch action {
	case "search":
		query := input["query"]
		size := 10
		if s, ok := input["size"].(float64); ok {
			size = int(s)
		}
		body := map[string]interface{}{
			"query": query,
			"size":  size,
		}
		data, err := t.doRequest(ctx, http.MethodPost, fmt.Sprintf("/%s/_search", index), body)
		if err != nil {
			return "", err
		}
		return prettyJSON(data), nil

	case "index":
		id, _ := input["id"].(string)
		doc := input["document"]
		path := fmt.Sprintf("/%s/_doc", index)
		if id != "" {
			path = fmt.Sprintf("/%s/_doc/%s", index, id)
		}
		data, err := t.doRequest(ctx, http.MethodPost, path, doc)
		if err != nil {
			return "", err
		}
		return prettyJSON(data), nil

	case "get":
		id, _ := input["id"].(string)
		if id == "" {
			return "", fmt.Errorf("'id' is required for get action")
		}
		data, err := t.doRequest(ctx, http.MethodGet, fmt.Sprintf("/%s/_doc/%s", index, id), nil)
		if err != nil {
			return "", err
		}
		return prettyJSON(data), nil

	case "delete":
		id, _ := input["id"].(string)
		if id == "" {
			return "", fmt.Errorf("'id' is required for delete action")
		}
		data, err := t.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/%s/_doc/%s", index, id), nil)
		if err != nil {
			return "", err
		}
		return prettyJSON(data), nil

	case "count":
		data, err := t.doRequest(ctx, http.MethodGet, fmt.Sprintf("/%s/_count", index), nil)
		if err != nil {
			return "", err
		}
		return prettyJSON(data), nil

	default:
		return "", fmt.Errorf("unsupported action: %s", action)
	}
}

func (t *ElasticsearchTool) RequiresReview() bool { return false }

func prettyJSON(data []byte) string {
	var buf bytes.Buffer
	if json.Indent(&buf, data, "", "  ") == nil {
		return buf.String()
	}
	return string(data)
}
