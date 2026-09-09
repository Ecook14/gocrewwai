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

// SupabaseTool allows agents to interact with Supabase.
type SupabaseTool struct {
	BaseTool
	client   *http.Client
	url      string
	apiKey   string
	table    string
}

// NewSupabaseTool creates a Supabase integration tool.
func NewSupabaseTool(url, apiKey, table string) *SupabaseTool {
	if url == "" {
		url = os.Getenv("SUPABASE_URL")
	}
	if apiKey == "" {
		apiKey = os.Getenv("SUPABASE_ANON_KEY")
	}
	return &SupabaseTool{
		BaseTool: BaseTool{
			NameValue:        "SupabaseTool",
			DescriptionValue: "Interacts with Supabase. Actions: select, insert, update, delete, rpc.",
		},
		client: &http.Client{Timeout: 30 * time.Second},
		url:     url,
		apiKey:  apiKey,
		table:   table,
	}
}

func (s *SupabaseTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	action, ok := input["action"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'action' parameter")
	}

	table := s.table
	if t, ok := input["table"].(string); ok && t != "" {
		table = t
	}

	switch action {
	case "select":
		columns, _ := input["columns"].(string)
		if columns == "" { columns = "*" }
		filter, _ := input["filter"].(string)
		return s.selectRows(ctx, table, columns, filter)
	case "insert":
		record, _ := input["record"].(map[string]interface{})
		return s.insertRow(ctx, table, record)
	case "update":
		filter, _ := input["filter"].(string)
		updates, _ := input["updates"].(map[string]interface{})
		return s.updateRows(ctx, table, filter, updates)
	case "delete":
		filter, _ := input["filter"].(string)
		return s.deleteRows(ctx, table, filter)
	case "rpc":
		fn, _ := input["function"].(string)
		params, _ := input["params"].(map[string]interface{})
		return s.rpc(ctx, fn, params)
	default:
		return "", fmt.Errorf("unknown Supabase action: %s", action)
	}
}

func (s *SupabaseTool) urlFor(table string) string {
	return fmt.Sprintf("%s/rest/v1/%s", s.url, table)
}

func (s *SupabaseTool) selectRows(ctx context.Context, table, columns, filter string) (string, error) {
	url := fmt.Sprintf("%s?select=%s", s.urlFor(table), columns)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("apikey", s.apiKey)
	resp, err := s.client.Do(req.WithContext(ctx))
	if err != nil { return "", err }
	defer resp.Body.Close()
	var result []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (s *SupabaseTool) insertRow(ctx context.Context, table string, record map[string]interface{}) (string, error) {
	data, _ := json.Marshal(record)
	url := s.urlFor(table)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(data)))
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.apiKey)
	resp, err := s.client.Do(req.WithContext(ctx))
	if err != nil { return "", err }
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (s *SupabaseTool) updateRows(ctx context.Context, table, filter string, updates map[string]interface{}) (string, error) {
	data, _ := json.Marshal(updates)
	url := fmt.Sprintf("%s?%s", s.urlFor(table), filter)
	req, _ := http.NewRequest("PATCH", url, strings.NewReader(string(data)))
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req.WithContext(ctx))
	if err != nil { return "", err }
	defer resp.Body.Close()
	return fmt.Sprintf("Updated rows in %s", table), nil
}

func (s *SupabaseTool) deleteRows(ctx context.Context, table, filter string) (string, error) {
	url := fmt.Sprintf("%s?%s", s.urlFor(table), filter)
	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	resp, err := s.client.Do(req.WithContext(ctx))
	if err != nil { return "", err }
	defer resp.Body.Close()
	return fmt.Sprintf("Deleted rows from %s", table), nil
}

func (s *SupabaseTool) rpc(ctx context.Context, fn string, params map[string]interface{}) (string, error) {
	data, _ := json.Marshal(params)
	url := fmt.Sprintf("%s/rest/v1/rpc/%s", s.url, fn)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(data)))
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req.WithContext(ctx))
	if err != nil { return "", err }
	defer resp.Body.Close()
	var result []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (s *SupabaseTool) RequiresReview() bool { return true }
func (s *SupabaseTool) Name() string { return s.BaseTool.NameValue }
func (s *SupabaseTool) Description() string { return s.BaseTool.DescriptionValue }
