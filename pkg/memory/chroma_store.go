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

// ChromaStore is a persistent implementation of the Store interface for ChromaDB.
// It uses the ChromaDB REST API for operations.
type ChromaStore struct {
	BaseURL        string
	CollectionName string
	CollectionID   string
	httpClient     *http.Client
	Timeout        time.Duration
}

// NewChromaStore initializes a new ChromaDB store with configurable timeout.
func NewChromaStore(baseURL, collectionName string, timeout time.Duration) (*ChromaStore, error) {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	s := &ChromaStore{
		BaseURL:        baseURL,
		CollectionName: collectionName,
		httpClient:     &http.Client{Timeout: timeout},
		Timeout:        timeout,
	}

	// 1. Get or Create Collection
	if err := s.ensureCollection(); err != nil {
		return nil, fmt.Errorf("failed to ensure chroma collection: %w", err)
	}

	return s, nil
}

func (s *ChromaStore) ensureCollection() error {
	// Simple create or get logic via Chroma REST API
	createURL := fmt.Sprintf("%s/api/v1/collections", s.BaseURL)
	payload := map[string]string{"name": s.CollectionName}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := s.httpClient.Post(createURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		json.NewDecoder(resp.Body).Decode(&result)
		s.CollectionID = result.ID
		return nil
	}

	// If already exists, try to get it
	getURL := fmt.Sprintf("%s/api/v1/collections/%s", s.BaseURL, s.CollectionName)
	resp, err = s.httpClient.Get(getURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		json.NewDecoder(resp.Body).Decode(&result)
		s.CollectionID = result.ID
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("chroma api error: %s (status %d)", string(body), resp.StatusCode)
}

func (s *ChromaStore) Add(ctx context.Context, item *MemoryItem) error {
	addURL := fmt.Sprintf("%s/api/v1/collections/%s/add", s.BaseURL, s.CollectionID)

	metadataJSON, _ := json.Marshal(item.Metadata)
	var meta map[string]interface{}
	json.Unmarshal(metadataJSON, &meta)

	payload := map[string]interface{}{
		"ids":      []string{item.ID},
		"embeddings": [][]float32{item.Vector},
		"metadatas":  []map[string]interface{}{meta},
		"documents":  []string{item.Text},
	}

	jsonPayload, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, addURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chroma add error: %s", string(body))
	}

	return nil
}

func (s *ChromaStore) Search(ctx context.Context, queryVector []float32, limit int) ([]*MemoryItem, error) {
	queryURL := fmt.Sprintf("%s/api/v1/collections/%s/query", s.BaseURL, s.CollectionID)

	payload := map[string]interface{}{
		"query_embeddings": [][]float32{queryVector},
		"n_results":        limit,
	}

	jsonPayload, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, queryURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("chroma query error: %s", string(body))
	}

	var rawResult struct {
		IDs       [][]string                 `json:"ids"`
		Documents [][]string                 `json:"documents"`
		Metadatas [][]map[string]interface{} `json:"metadatas"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawResult); err != nil {
		return nil, err
	}

	var items []*MemoryItem
	if len(rawResult.IDs) > 0 {
		for i := 0; i < len(rawResult.IDs[0]); i++ {
			items = append(items, &MemoryItem{
				ID:       rawResult.IDs[0][i],
				Text:     rawResult.Documents[0][i],
				Metadata: rawResult.Metadatas[0][i],
			})
		}
	}

	return items, nil
}

// BulkAdd inserts multiple items by calling Add for each.
func (s *ChromaStore) BulkAdd(ctx context.Context, items []*MemoryItem) error {
	for _, item := range items {
		if err := s.Add(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes an item from the collection by ID.
func (s *ChromaStore) Delete(ctx context.Context, id string) error {
	deleteURL := fmt.Sprintf("%s/api/v1/collections/%s/delete", s.BaseURL, s.CollectionID)
	payload := map[string]interface{}{
		"ids": []string{id},
	}
	jsonPayload, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deleteURL, bytes.NewBuffer(jsonPayload))
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chroma delete error: %s", string(body))
	}
	return nil
}

// Count returns the number of items in the collection.
func (s *ChromaStore) Count(ctx context.Context) (int, error) {
	countURL := fmt.Sprintf("%s/api/v1/collections/%s/count", s.BaseURL, s.CollectionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, countURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var count int
	if err := json.NewDecoder(resp.Body).Decode(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *ChromaStore) Close() error {
	// HTTP client doesn't need explicit closing in this context
	return nil
}
