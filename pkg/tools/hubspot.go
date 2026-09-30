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

// HubSpotTool allows agents to interact with HubSpot CRM.
type HubSpotTool struct {
	BaseTool
	client *http.Client
	token  string
}

// NewHubSpotTool creates a HubSpot CRM integration tool.
func NewHubSpotTool(token string) *HubSpotTool {
	if token == "" {
		token = os.Getenv("HUBSPOT_ACCESS_TOKEN")
	}
	return &HubSpotTool{
		BaseTool: BaseTool{
			NameValue:        "HubSpotTool",
			DescriptionValue: "Interacts with HubSpot CRM. Actions: search_contacts, get_deal, create_deal, create_contact, list_companies, create_company.",
		},
		client: &http.Client{Timeout: 30 * time.Second},
		token:  token,
	}
}

// Execute implements the Tool interface for HubSpot.
func (h *HubSpotTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	action, ok := input["action"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'action' parameter")
	}

	switch action {
	case "search_contacts":
		q, _ := input["query"].(string)
		return h.searchContacts(ctx, q)
	case "get_deal":
		dealID, _ := input["deal_id"].(string)
		return h.getDeal(ctx, dealID)
	case "create_deal":
		name, _ := input["deal_name"].(string)
		amount, _ := input["amount"].(float64)
		return h.createDeal(ctx, name, amount)
	case "create_contact":
		email, _ := input["email"].(string)
		firstName, _ := input["first_name"].(string)
		lastName, _ := input["last_name"].(string)
		return h.createContact(ctx, email, firstName, lastName)
	case "list_companies":
		return h.listCompanies(ctx)
	case "create_company":
		name, _ := input["company_name"].(string)
		return h.createCompany(ctx, name)
	default:
		return "", fmt.Errorf("unknown HubSpot action: %s", action)
	}
}

func (h *HubSpotTool) searchContacts(ctx context.Context, query string) (string, error) {
	data, _ := json.Marshal(map[string]string{"query": query, "limit": "10"})
	req, err := http.NewRequest("POST", "https://api.crm.hubspot.com/crm/v3/objects/contacts/search", strings.NewReader(string(data)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+h.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (h *HubSpotTool) getDeal(ctx context.Context, dealID string) (string, error) {
	if !validHubSpotID(dealID) {
		return "", fmt.Errorf("invalid deal id")
	}
	url := fmt.Sprintf("https://api.crm.hubspot.com/crm/v3/objects/deals/%s", url.PathEscape(dealID))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+h.token)
	resp, err := h.client.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (h *HubSpotTool) createDeal(ctx context.Context, name string, amount float64) (string, error) {
	data, _ := json.Marshal(map[string]interface{}{"properties": map[string]interface{}{"dealname": name, "amount": amount}})
	req, err := http.NewRequest("POST", "https://api.crm.hubspot.com/crm/v3/objects/deals", strings.NewReader(string(data)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+h.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (h *HubSpotTool) createContact(ctx context.Context, email, firstName, lastName string) (string, error) {
	data, _ := json.Marshal(map[string]interface{}{"properties": []map[string]interface{}{
		{"property": "email", "value": email},
		{"property": "firstname", "value": firstName},
		{"property": "lastname", "value": lastName},
	}})
	req, err := http.NewRequest("POST", "https://api.crm.hubspot.com/crm/v3/objects/contacts", strings.NewReader(string(data)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+h.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (h *HubSpotTool) listCompanies(ctx context.Context) (string, error) {
	req, err := http.NewRequest("GET", "https://api.crm.hubspot.com/crm/v3/objects/companies?limit=10", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+h.token)
	resp, err := h.client.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (h *HubSpotTool) createCompany(ctx context.Context, name string) (string, error) {
	data, _ := json.Marshal(map[string]interface{}{"properties": map[string]interface{}{"name": name}})
	req, err := http.NewRequest("POST", "https://api.crm.hubspot.com/crm/v3/objects/companies", strings.NewReader(string(data)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+h.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func validHubSpotID(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	for _, r := range s {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func (h *HubSpotTool) RequiresReview() bool { return true }
func (h *HubSpotTool) Name() string         { return h.BaseTool.NameValue }
func (h *HubSpotTool) Description() string  { return h.BaseTool.DescriptionValue }

var _ Tool = (*HubSpotTool)(nil)
