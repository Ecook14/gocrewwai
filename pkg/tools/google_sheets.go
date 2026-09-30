package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// GoogleSheetsTool allows agents to interact with Google Sheets.
type GoogleSheetsTool struct {
	BaseTool
	client *http.Client
	token  string
}

// NewGoogleSheetsTool creates a Google Sheets integration tool.
func NewGoogleSheetsTool(token string) *GoogleSheetsTool {
	if token == "" {
		token = os.Getenv("GOOGLE_SHEETS_TOKEN")
	}
	return &GoogleSheetsTool{
		BaseTool: BaseTool{
			NameValue:        "GoogleSheetsTool",
			DescriptionValue: "Interacts with Google Sheets. Actions: read_range, append_rows, update_cells, clear_range.",
		},
		client: &http.Client{Timeout: 30 * time.Second},
		token:  token,
	}
}

// Execute implements the Tool interface for Google Sheets.
func (g *GoogleSheetsTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	action, ok := input["action"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'action' parameter")
	}

	spreadsheetID, _ := input["spreadsheet_id"].(string)
	if spreadsheetID == "" {
		return "", fmt.Errorf("missing 'spreadsheet_id' parameter")
	}

	switch action {
	case "read_range":
		rangeName, _ := input["range"].(string)
		return g.readRange(ctx, spreadsheetID, rangeName)
	case "append_rows":
		rangeName, _ := input["range"].(string)
		values, _ := input["values"].([][]interface{})
		return g.appendRows(ctx, spreadsheetID, rangeName, values)
	case "update_cells":
		rangeName, _ := input["range"].(string)
		values, _ := input["values"].([][]interface{})
		return g.updateCells(ctx, spreadsheetID, rangeName, values)
	case "clear_range":
		rangeName, _ := input["range"].(string)
		return g.clearRange(ctx, spreadsheetID, rangeName)
	default:
		return "", fmt.Errorf("unknown Google Sheets action: %s", action)
	}
}

func (g *GoogleSheetsTool) url(spreadsheetID string) string {
	return fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s", url.PathEscape(spreadsheetID))
}

func validSheetsID(s string) bool {
	if s == "" || len(s) > 128 {
		return false
	}
	for _, r := range s {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func validSheetsRange(s string) bool {
	if s == "" || len(s) > 256 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case r == '!' || r == ':' || r == '.' || r == '_' || r == '-' || r == ' ' || r == '\'' || r == '(' || r == ')':
		default:
			return false
		}
	}
	return true
}

func (g *GoogleSheetsTool) readRange(ctx context.Context, spreadsheetID, rangeName string) (string, error) {
	if !validSheetsID(spreadsheetID) || !validSheetsRange(rangeName) {
		return "", fmt.Errorf("invalid spreadsheet id or range")
	}
	url := g.url(spreadsheetID) + "/values/" + url.PathEscape(rangeName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
	resp, err := g.client.Do(req.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("Google Sheets read failed: %w", err)
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (g *GoogleSheetsTool) appendRows(ctx context.Context, spreadsheetID, rangeName string, values [][]interface{}) (string, error) {
	if !validSheetsID(spreadsheetID) || !validSheetsRange(rangeName) {
		return "", fmt.Errorf("invalid spreadsheet id or range")
	}
	reqBody, _ := json.Marshal(map[string]interface{}{"values": values})
	url := g.url(spreadsheetID) + "/values/" + url.PathEscape(rangeName) + ":append?valueInputOption=RAW"
	req, err := http.NewRequest("POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("Google Sheets append failed: %w", err)
	}
	defer resp.Body.Close()
	return fmt.Sprintf("Appended %d rows to %s", len(values), rangeName), nil
}

func (g *GoogleSheetsTool) updateCells(ctx context.Context, spreadsheetID, rangeName string, values [][]interface{}) (string, error) {
	if !validSheetsID(spreadsheetID) || !validSheetsRange(rangeName) {
		return "", fmt.Errorf("invalid spreadsheet id or range")
	}
	reqBody, _ := json.Marshal(map[string]interface{}{"values": values})
	url := g.url(spreadsheetID) + "/values/" + url.PathEscape(rangeName) + "?valueInputOption=RAW"
	req, err := http.NewRequest("PUT", url, strings.NewReader(string(reqBody)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("Google Sheets update failed: %w", err)
	}
	defer resp.Body.Close()
	return fmt.Sprintf("Updated %d cells in %s", len(values), rangeName), nil
}

func (g *GoogleSheetsTool) clearRange(ctx context.Context, spreadsheetID, rangeName string) (string, error) {
	if !validSheetsID(spreadsheetID) || !validSheetsRange(rangeName) {
		return "", fmt.Errorf("invalid spreadsheet id or range")
	}
	url := g.url(spreadsheetID) + "/values/" + url.PathEscape(rangeName) + ":clear"
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
	resp, err := g.client.Do(req.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("Google Sheets clear failed: %w", err)
	}
	defer resp.Body.Close()
	return fmt.Sprintf("Cleared range %s", rangeName), nil
}

func (g *GoogleSheetsTool) RequiresReview() bool { return true }
func (g *GoogleSheetsTool) Name() string         { return g.BaseTool.NameValue }
func (g *GoogleSheetsTool) Description() string  { return g.BaseTool.DescriptionValue }

var _ Tool = (*GoogleSheetsTool)(nil)
