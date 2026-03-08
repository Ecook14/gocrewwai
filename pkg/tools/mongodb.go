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

// MongoDBTool allows agents to interact with MongoDB via the Data API.
// This uses the MongoDB Atlas Data API (REST-based) to avoid requiring
// the full MongoDB Go driver as a dependency.
//
// Input examples:
//
//	{"action": "find", "collection": "users", "filter": {"name": "Alice"}}
//	{"action": "insertOne", "collection": "users", "document": {"name": "Bob"}}
//	{"action": "updateOne", "collection": "users", "filter": {"name": "Bob"}, "update": {"$set": {"age": 30}}}
//	{"action": "deleteOne", "collection": "users", "filter": {"name": "Bob"}}
type MongoDBTool struct {
	BaseTool
	Endpoint   string // Data API endpoint URL
	APIKey     string
	DataSource string
	Database   string
	httpClient *http.Client
}

// NewMongoDBTool creates a MongoDB Data API tool.
func NewMongoDBTool(endpoint, apiKey, dataSource, database string) *MongoDBTool {
	return &MongoDBTool{
		BaseTool: BaseTool{
			NameValue:        "MongoDBTool",
			DescriptionValue: "Interact with MongoDB via Data API. Actions: find, findOne, insertOne, insertMany, updateOne, deleteOne. Input: {'action': '...', 'collection': '...', 'filter': {...}, 'document': {...}}",
		},
		Endpoint:   endpoint,
		APIKey:     apiKey,
		DataSource: dataSource,
		Database:   database,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (t *MongoDBTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	action, _ := input["action"].(string)
	collection, _ := input["collection"].(string)
	if action == "" || collection == "" {
		return "", fmt.Errorf("'action' and 'collection' are required")
	}

	// Build Data API request body
	body := map[string]interface{}{
		"dataSource": t.DataSource,
		"database":   t.Database,
		"collection": collection,
	}

	switch action {
	case "find":
		body["filter"] = input["filter"]
		if limit, ok := input["limit"]; ok {
			body["limit"] = limit
		}
	case "findOne":
		body["filter"] = input["filter"]
	case "insertOne":
		body["document"] = input["document"]
	case "insertMany":
		body["documents"] = input["documents"]
	case "updateOne":
		body["filter"] = input["filter"]
		body["update"] = input["update"]
	case "deleteOne":
		body["filter"] = input["filter"]
	default:
		return "", fmt.Errorf("unsupported action: %s", action)
	}

	endpoint := fmt.Sprintf("%s/action/%s", strings.TrimRight(t.Endpoint, "/"), action)

	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", t.APIKey)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("mongodb request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("mongodb error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Pretty format the response
	var pretty bytes.Buffer
	if json.Indent(&pretty, respBody, "", "  ") == nil {
		return pretty.String(), nil
	}
	return string(respBody), nil
}

func (t *MongoDBTool) RequiresReview() bool { return true }
