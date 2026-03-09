package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ---------------------------------------------------------------------------
// WeaviateStore — Weaviate Vector Database Client
// ---------------------------------------------------------------------------
//
// Weaviate is an open-source vector database with built-in vectorization,
// hybrid search (vector + keyword), and GraphQL API.
//
// Usage:
//
//	store, err := memory.NewWeaviateStore("http://localhost:8080", "CrewMemory", "")
//	store.Add(ctx, item)
//	results, _ := store.Search(ctx, queryVector, 10)
type WeaviateStore struct {
	Host       string
	ClassName  string // Weaviate class name (PascalCase required)
	APIKey     string // Optional — empty for local/anonymous
	httpClient *http.Client
}

// NewWeaviateStore creates a client for a Weaviate class.
// If the class doesn't exist, it will be created.
func NewWeaviateStore(host, className, apiKey string) (*WeaviateStore, error) {
	if host == "" || className == "" {
		return nil, fmt.Errorf("weaviate: host and className are required")
	}

	s := &WeaviateStore{
		Host:       host,
		ClassName:  className,
		APIKey:     apiKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}

	if err := s.ensureClass(); err != nil {
		return nil, fmt.Errorf("weaviate: failed to ensure class: %w", err)
	}

	return s, nil
}

func (s *WeaviateStore) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.Host+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.APIKey)
	}
	return s.httpClient.Do(req)
}

func (s *WeaviateStore) ensureClass() error {
	// Check if class exists
	resp, err := s.doRequest(context.Background(), http.MethodGet,
		fmt.Sprintf("/v1/schema/%s", s.ClassName), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	// Create class with text property and no vectorizer (we provide vectors)
	classDef := map[string]interface{}{
		"class":      s.ClassName,
		"vectorizer": "none",
		"properties": []map[string]interface{}{
			{"name": "text", "dataType": []string{"text"}},
			{"name": "itemId", "dataType": []string{"text"}},
			{"name": "metadata", "dataType": []string{"text"}},
			{"name": "createdAt", "dataType": []string{"text"}},
			{"name": "expiresAt", "dataType": []string{"text"}},
		},
	}

	resp, err = s.doRequest(context.Background(), http.MethodPost, "/v1/schema", classDef)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create class (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (s *WeaviateStore) Add(ctx context.Context, item *MemoryItem) error {
	metaJSON, _ := json.Marshal(item.Metadata)

	obj := map[string]interface{}{
		"class": s.ClassName,
		"properties": map[string]interface{}{
			"text":      item.Text,
			"itemId":    item.ID,
			"metadata":  string(metaJSON),
			"createdAt": item.CreatedAt.Format(time.RFC3339),
			"expiresAt": "",
		},
		"vector": item.Vector,
	}
	if !item.ExpiresAt.IsZero() {
		obj["properties"].(map[string]interface{})["expiresAt"] = item.ExpiresAt.Format(time.RFC3339)
	}

	// Use deterministic UUID from item ID for idempotent upserts
	path := fmt.Sprintf("/v1/objects/%s/%s", s.ClassName, item.ID)
	resp, err := s.doRequest(ctx, http.MethodPut, path, obj)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// Fallback: try POST (create) if PUT (update) fails
		resp2, err2 := s.doRequest(ctx, http.MethodPost, "/v1/objects", obj)
		if err2 != nil {
			return err2
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != http.StatusOK && resp2.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp2.Body)
			return fmt.Errorf("weaviate add error: %s", string(body))
		}
	}
	return nil
}

func (s *WeaviateStore) BulkAdd(ctx context.Context, items []*MemoryItem) error {
	type batchObj struct {
		Class      string                 `json:"class"`
		Properties map[string]interface{} `json:"properties"`
		Vector     []float32              `json:"vector,omitempty"`
		ID         string                 `json:"id,omitempty"`
	}

	objects := make([]batchObj, 0, len(items))
	for _, item := range items {
		metaJSON, _ := json.Marshal(item.Metadata)
		expiresAt := ""
		if !item.ExpiresAt.IsZero() {
			expiresAt = item.ExpiresAt.Format(time.RFC3339)
		}
		objects = append(objects, batchObj{
			Class: s.ClassName,
			ID:    item.ID,
			Properties: map[string]interface{}{
				"text":      item.Text,
				"itemId":    item.ID,
				"metadata":  string(metaJSON),
				"createdAt": item.CreatedAt.Format(time.RFC3339),
				"expiresAt": expiresAt,
			},
			Vector: item.Vector,
		})
	}

	payload := map[string]interface{}{"objects": objects}
	resp, err := s.doRequest(ctx, http.MethodPost, "/v1/batch/objects", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("weaviate batch error: %s", string(body))
	}
	return nil
}

func (s *WeaviateStore) Delete(ctx context.Context, id string) error {
	path := fmt.Sprintf("/v1/objects/%s/%s", s.ClassName, id)
	resp, err := s.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("weaviate delete error: %s", string(body))
	}
	return nil
}

func (s *WeaviateStore) Search(ctx context.Context, queryVector []float32, limit int) ([]*MemoryItem, error) {
	// Use GraphQL nearVector search
	vectorJSON, _ := json.Marshal(queryVector)
	gql := fmt.Sprintf(`{
		Get {
			%s(
				nearVector: { vector: %s }
				limit: %d
			) {
				text
				itemId
				metadata
				createdAt
				expiresAt
				_additional { id distance }
			}
		}
	}`, s.ClassName, string(vectorJSON), limit)

	payload := map[string]string{"query": gql}
	resp, err := s.doRequest(ctx, http.MethodPost, "/v1/graphql", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Get map[string][]struct {
				Text       string `json:"text"`
				ItemID     string `json:"itemId"`
				Metadata   string `json:"metadata"`
				CreatedAt  string `json:"createdAt"`
				ExpiresAt  string `json:"expiresAt"`
				Additional struct {
					ID       string  `json:"id"`
					Distance float64 `json:"distance"`
				} `json:"_additional"`
			} `json:"Get"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	objects := result.Data.Get[s.ClassName]
	items := make([]*MemoryItem, 0, len(objects))
	for _, obj := range objects {
		item := &MemoryItem{
			ID:   obj.ItemID,
			Text: obj.Text,
		}
		if obj.ItemID == "" {
			item.ID = obj.Additional.ID
		}
		if obj.Metadata != "" {
			json.Unmarshal([]byte(obj.Metadata), &item.Metadata)
		}
		if obj.CreatedAt != "" {
			item.CreatedAt, _ = time.Parse(time.RFC3339, obj.CreatedAt)
		}
		if obj.ExpiresAt != "" {
			item.ExpiresAt, _ = time.Parse(time.RFC3339, obj.ExpiresAt)
		}
		items = append(items, item)
	}

	return items, nil
}

func (s *WeaviateStore) Count(ctx context.Context) (int, error) {
	gql := fmt.Sprintf(`{ Aggregate { %s { meta { count } } } }`, s.ClassName)
	payload := map[string]string{"query": gql}
	resp, err := s.doRequest(ctx, http.MethodPost, "/v1/graphql", payload)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Aggregate map[string][]struct {
				Meta struct {
					Count int `json:"count"`
				} `json:"meta"`
			} `json:"Aggregate"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if agg, ok := result.Data.Aggregate[s.ClassName]; ok && len(agg) > 0 {
		return agg[0].Meta.Count, nil
	}
	return 0, nil
}

func (s *WeaviateStore) Reset(ctx context.Context) error {
	path := fmt.Sprintf("/v1/schema/%s", s.ClassName)
	resp, err := s.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return s.ensureClass()
}

func (s *WeaviateStore) Close() error {
	return nil
}
