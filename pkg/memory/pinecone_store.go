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
// PineconeStore — Pinecone Vector Database Client
// ---------------------------------------------------------------------------
//
// Pinecone is a managed vector database optimized for similarity search
// at scale. It uses serverless or pod-based deployment.
//
// Usage:
//
//	store, err := memory.NewPineconeStore("https://index-xxx.svc.aped-xxx.pinecone.io", "your-api-key", "crew")
//	store.Add(ctx, item)
//	results, _ := store.Search(ctx, queryVector, 10)
type PineconeStore struct {
	Host       string // Full index host URL
	APIKey     string
	Namespace  string
	httpClient *http.Client
}

// NewPineconeStore creates a client for a Pinecone index.
func NewPineconeStore(host, apiKey, namespace string) (*PineconeStore, error) {
	if host == "" || apiKey == "" {
		return nil, fmt.Errorf("pinecone: host and apiKey are required")
	}
	return &PineconeStore{
		Host:       host,
		APIKey:     apiKey,
		Namespace:  namespace,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (s *PineconeStore) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
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
	req.Header.Set("Api-Key", s.APIKey)

	return s.httpClient.Do(req)
}

func (s *PineconeStore) Add(ctx context.Context, item *MemoryItem) error {
	return s.BulkAdd(ctx, []*MemoryItem{item})
}

func (s *PineconeStore) BulkAdd(ctx context.Context, items []*MemoryItem) error {
	type vector struct {
		ID       string                 `json:"id"`
		Values   []float32              `json:"values"`
		Metadata map[string]interface{} `json:"metadata,omitempty"`
	}

	vectors := make([]vector, 0, len(items))
	for _, item := range items {
		meta := make(map[string]interface{})
		if item.Metadata != nil {
			for k, v := range item.Metadata {
				meta[k] = v
			}
		}
		meta["text"] = item.Text
		if !item.CreatedAt.IsZero() {
			meta["created_at"] = item.CreatedAt.Format(time.RFC3339)
		}
		if !item.ExpiresAt.IsZero() {
			meta["expires_at"] = item.ExpiresAt.Format(time.RFC3339)
		}

		vectors = append(vectors, vector{
			ID:       item.ID,
			Values:   item.Vector,
			Metadata: meta,
		})
	}

	payload := map[string]interface{}{
		"vectors":   vectors,
		"namespace": s.Namespace,
	}

	resp, err := s.doRequest(ctx, http.MethodPost, "/vectors/upsert", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pinecone upsert error (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (s *PineconeStore) Delete(ctx context.Context, id string) error {
	payload := map[string]interface{}{
		"ids":       []string{id},
		"namespace": s.Namespace,
	}

	resp, err := s.doRequest(ctx, http.MethodPost, "/vectors/delete", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pinecone delete error: %s", string(body))
	}
	return nil
}

func (s *PineconeStore) Search(ctx context.Context, queryVector []float32, limit int) ([]*MemoryItem, error) {
	payload := map[string]interface{}{
		"vector":          queryVector,
		"topK":            limit,
		"includeMetadata": true,
		"namespace":       s.Namespace,
	}

	resp, err := s.doRequest(ctx, http.MethodPost, "/query", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pinecone query error: %s", string(body))
	}

	var result struct {
		Matches []struct {
			ID       string                 `json:"id"`
			Score    float64                `json:"score"`
			Metadata map[string]interface{} `json:"metadata"`
		} `json:"matches"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	items := make([]*MemoryItem, 0, len(result.Matches))
	for _, m := range result.Matches {
		text, _ := m.Metadata["text"].(string)
		items = append(items, &MemoryItem{
			ID:       m.ID,
			Text:     text,
			Metadata: m.Metadata,
		})
	}
	return items, nil
}

func (s *PineconeStore) Count(ctx context.Context) (int, error) {
	payload := map[string]interface{}{
		"namespace": s.Namespace,
	}
	resp, err := s.doRequest(ctx, http.MethodPost, "/describe_index_stats", payload)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Namespaces map[string]struct {
			VectorCount int `json:"vectorCount"`
		} `json:"namespaces"`
		TotalVectorCount int `json:"totalVectorCount"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if ns, ok := result.Namespaces[s.Namespace]; ok {
		return ns.VectorCount, nil
	}
	return result.TotalVectorCount, nil
}

func (s *PineconeStore) Close() error {
	return nil
}
