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
// QdrantStore — Qdrant Vector Database Client
// ---------------------------------------------------------------------------
//
// Qdrant is a high-performance vector similarity search engine with rich
// filtering, payload storage, and horizontal scaling.
//
// Usage:
//
//	store, err := memory.NewQdrantStore("http://localhost:6333", "crew_memory", 1536)
//	store.Add(ctx, item)
//	results, _ := store.Search(ctx, queryVector, 10)
type QdrantStore struct {
	BaseURL        string
	CollectionName string
	VectorSize     int
	httpClient     *http.Client
}

// NewQdrantStore creates a client for a Qdrant collection.
// If the collection doesn't exist, it will be created with the given vector size.
func NewQdrantStore(baseURL, collectionName string, vectorSize int) (*QdrantStore, error) {
	s := &QdrantStore{
		BaseURL:        baseURL,
		CollectionName: collectionName,
		VectorSize:     vectorSize,
		httpClient:     &http.Client{Timeout: 15 * time.Second},
	}

	if err := s.ensureCollection(); err != nil {
		return nil, fmt.Errorf("qdrant: failed to ensure collection: %w", err)
	}

	return s, nil
}

func (s *QdrantStore) ensureCollection() error {
	url := fmt.Sprintf("%s/collections/%s", s.BaseURL, s.CollectionName)

	// Check if collection exists
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil // Already exists
	}

	// Create collection
	payload := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     s.VectorSize,
			"distance": "Cosine",
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection (status %d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (s *QdrantStore) Add(ctx context.Context, item *MemoryItem) error {
	return s.BulkAdd(ctx, []*MemoryItem{item})
}

func (s *QdrantStore) BulkAdd(ctx context.Context, items []*MemoryItem) error {
	url := fmt.Sprintf("%s/collections/%s/points", s.BaseURL, s.CollectionName)

	type point struct {
		ID      string                 `json:"id"`
		Vector  []float32              `json:"vector"`
		Payload map[string]interface{} `json:"payload"`
	}

	points := make([]point, 0, len(items))
	for _, item := range items {
		payload := make(map[string]interface{})
		if item.Metadata != nil {
			for k, v := range item.Metadata {
				payload[k] = v
			}
		}
		payload["text"] = item.Text
		if !item.CreatedAt.IsZero() {
			payload["created_at"] = item.CreatedAt.Format(time.RFC3339)
		}
		if !item.ExpiresAt.IsZero() {
			payload["expires_at"] = item.ExpiresAt.Format(time.RFC3339)
		}

		points = append(points, point{
			ID:      item.ID,
			Vector:  item.Vector,
			Payload: payload,
		})
	}

	body, _ := json.Marshal(map[string]interface{}{"points": points})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant upsert error (status %d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (s *QdrantStore) Delete(ctx context.Context, id string) error {
	url := fmt.Sprintf("%s/collections/%s/points/delete", s.BaseURL, s.CollectionName)
	payload := map[string]interface{}{
		"points": []string{id},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant delete error: %s", string(respBody))
	}
	return nil
}

func (s *QdrantStore) Search(ctx context.Context, queryVector []float32, limit int) ([]*MemoryItem, error) {
	url := fmt.Sprintf("%s/collections/%s/points/search", s.BaseURL, s.CollectionName)
	payload := map[string]interface{}{
		"vector":       queryVector,
		"limit":        limit,
		"with_payload": true,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Result []struct {
			ID      interface{}            `json:"id"`
			Payload map[string]interface{} `json:"payload"`
			Score   float64                `json:"score"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	items := make([]*MemoryItem, 0, len(result.Result))
	for _, r := range result.Result {
		text, _ := r.Payload["text"].(string)
		item := &MemoryItem{
			ID:       fmt.Sprintf("%v", r.ID),
			Text:     text,
			Metadata: r.Payload,
		}
		items = append(items, item)
	}

	return items, nil
}

func (s *QdrantStore) Count(ctx context.Context) (int, error) {
	url := fmt.Sprintf("%s/collections/%s", s.BaseURL, s.CollectionName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Result struct {
			PointsCount int `json:"points_count"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	return result.Result.PointsCount, nil
}

func (s *QdrantStore) Reset(ctx context.Context) error {
	url := fmt.Sprintf("%s/collections/%s", s.BaseURL, s.CollectionName)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return s.ensureCollection()
}

func (s *QdrantStore) Close() error {
	return nil
}
