package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// NotionTool allows agents to interact with Notion.
type NotionTool struct {
	BaseTool
	client *http.Client
	token  string
}

// NewNotionTool creates a Notion integration tool.
func NewNotionTool(token string) *NotionTool {
	if token == "" {
		token = os.Getenv("NOTION_TOKEN")
	}
	return &NotionTool{
		BaseTool: BaseTool{
			NameValue:        "NotionTool",
			DescriptionValue: "Interacts with Notion. Actions: search_pages, read_page, create_page, list_databases, query_database.",
		},
		client: &http.Client{Timeout: 30 * time.Second},
		token:  token,
	}
}

// Execute implements the Tool interface for Notion.
func (n *NotionTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	action, ok := input["action"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'action' parameter")
	}

	switch action {
	case "search_pages":
		query, _ := input["query"].(string)
		return n.searchPages(ctx, query)
	case "read_page":
		pageID, _ := input["page_id"].(string)
		return n.readPage(ctx, pageID)
	case "create_page":
		title, _ := input["title"].(string)
		content, _ := input["content"].(string)
		return n.createPage(ctx, title, content)
	case "list_databases":
		return n.listDatabases(ctx)
	case "query_database":
		dbID, _ := input["database_id"].(string)
		return n.queryDatabase(ctx, dbID)
	default:
		return "", fmt.Errorf("unknown Notion action: %s", action)
	}
}

func (n *NotionTool) searchPages(ctx context.Context, query string) (string, error) {
	reqBody, _ := json.Marshal(map[string]string{"query": query})
	req, _ := http.NewRequest("POST", "https://api.notion.com/v1/search", strings.NewReader(string(reqBody)))
	req.Header.Set("Authorization", "Bearer "+n.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")
	resp, err := n.client.Do(req.WithContext(ctx))
	if err != nil { return "", fmt.Errorf("Notion search failed: %w", err) }
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (n *NotionTool) readPage(ctx context.Context, pageID string) (string, error) {
	url := fmt.Sprintf("https://api.notion.com/v1/blocks/%s", pageID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+n.token)
	req.Header.Set("Notion-Version", "2022-06-28")
	resp, err := n.client.Do(req.WithContext(ctx))
	if err != nil { return "", fmt.Errorf("Notion read failed: %w", err) }
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (n *NotionTool) createPage(ctx context.Context, title, content string) (string, error) {
	heading := map[string]interface{}{
		"object": "block", "type": "heading_2",
		"heading_2": map[string]interface{}{
			"text": []map[string]interface{}{
				{"type": "text", "text": map[string]interface{}{"content": title}},
			},
		},
	}
	blocks := []map[string]interface{}{heading}
	if content != "" {
		blocks = append(blocks, map[string]interface{}{
			"object": "block", "type": "paragraph",
			"paragraph": map[string]interface{}{
				"text": []map[string]interface{}{
					{"type": "text", "text": map[string]interface{}{"content": content}},
				},
			},
		})
	}
	reqBody, _ := json.Marshal(map[string]interface{}{
		"parent": map[string]interface{}{"type": "page_id", "page_id": title},
		"children": blocks,
	})
	req, _ := http.NewRequest("POST", "https://api.notion.com/v1/pages", strings.NewReader(string(reqBody)))
	req.Header.Set("Authorization", "Bearer "+n.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")
	resp, err := n.client.Do(req.WithContext(ctx))
	if err != nil { return "", fmt.Errorf("Notion create failed: %w", err) }
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (n *NotionTool) listDatabases(ctx context.Context) (string, error) {
	req, _ := http.NewRequest("GET", "https://api.notion.com/v1/databases", nil)
	req.Header.Set("Authorization", "Bearer "+n.token)
	req.Header.Set("Notion-Version", "2022-06-28")
	resp, err := n.client.Do(req.WithContext(ctx))
	if err != nil { return "", fmt.Errorf("Notion list databases failed: %w", err) }
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (n *NotionTool) queryDatabase(ctx context.Context, dbID string) (string, error) {
	url := fmt.Sprintf("https://api.notion.com/v1/databases/%s/query", dbID)
	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bearer "+n.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")
	resp, err := n.client.Do(req.WithContext(ctx))
	if err != nil { return "", fmt.Errorf("Notion query database failed: %w", err) }
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (n *NotionTool) RequiresReview() bool { return true }
func (n *NotionTool) Name() string { return n.BaseTool.NameValue }
func (n *NotionTool) Description() string { return n.BaseTool.DescriptionValue }
