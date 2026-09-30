package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	// Resource-identifier guard: caller-supplied index/id must remain single
	// path segments. A fragment (#) or ? would discard the tool-generated
	// document route and repurpose the credentials (e.g. index deletion).
	if !validESIndex(index) {
		return "", fmt.Errorf("invalid Elasticsearch index name")
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
		data, err := t.doRequest(ctx, http.MethodPost, "/"+url.PathEscape(index)+"/_search", body)
		if err != nil {
			return "", err
		}
		return prettyJSON(data), nil

	case "index":
		id, _ := input["id"].(string)
		doc := input["document"]
		path := "/" + url.PathEscape(index) + "/_doc"
		if id != "" {
			if !validESID(id) {
				return "", fmt.Errorf("invalid Elasticsearch document id")
			}
			path = "/" + url.PathEscape(index) + "/_doc/" + url.PathEscape(id)
		}
		data, err := t.doRequest(ctx, http.MethodPost, path, doc)
		if err != nil {
			return "", err
		}
		return prettyJSON(data), nil

	case "get":
		id, _ := input["id"].(string)
		if !validESID(id) {
			return "", fmt.Errorf("'id' is required for get action")
		}
		data, err := t.doRequest(ctx, http.MethodGet, "/"+url.PathEscape(index)+"/_doc/"+url.PathEscape(id), nil)
		if err != nil {
			return "", err
		}
		return prettyJSON(data), nil

	case "delete":
		id, _ := input["id"].(string)
		if !validESID(id) {
			return "", fmt.Errorf("'id' is required for delete action")
		}
		data, err := t.doRequest(ctx, http.MethodDelete, "/"+url.PathEscape(index)+"/_doc/"+url.PathEscape(id), nil)
		if err != nil {
			return "", err
		}
		return prettyJSON(data), nil

	case "count":
		data, err := t.doRequest(ctx, http.MethodGet, "/"+url.PathEscape(index)+"/_count", nil)
		if err != nil {
			return "", err
		}
		return prettyJSON(data), nil

	default:
		return "", fmt.Errorf("unsupported action: %s", action)
	}
}

// validESIndex enforces Elasticsearch identifier rules so values remain
// single URL path segments (no fragments, queries, or extra routes).
func validESIndex(s string) bool {
	if s == "" || len(s) > 255 || s[0] == '-' || s[0] == '_' || s[0] == '+' {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '+' || r == '.':
		default:
			return false
		}
	}
	return true
}

// validESID requires a non-empty single-segment document id.
func validESID(s string) bool {
	if s == "" || len(s) > 512 {
		return false
	}
	for _, r := range s {
		if r == '/' || r == '?' || r == '#' || r == '\n' || r == '\r' {
			return false
		}
	}
	return true
}

func (t *ElasticsearchTool) Name() string        { return t.BaseTool.NameValue }
func (t *ElasticsearchTool) Description() string { return t.BaseTool.DescriptionValue }

// RequiresReview gates queries against an external cluster.
func (t *ElasticsearchTool) RequiresReview() bool { return true }

var _ Tool = (*ElasticsearchTool)(nil)

func prettyJSON(data []byte) string {
	var buf bytes.Buffer
	if json.Indent(&buf, data, "", "  ") == nil {
		return buf.String()
	}
	return string(data)
}
