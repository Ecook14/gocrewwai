package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements `memory.Store`, providing persistent disk storage 
// for Agent contexts, mirroring Python CrewAI's LanceDB connections.
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS memory_items (
			id TEXT PRIMARY KEY,
			text TEXT,
			vector TEXT,
			metadata TEXT
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory table: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Add(ctx context.Context, item *MemoryItem) error {
	vectorBytes, _ := json.Marshal(item.Vector)
	metadataBytes, _ := json.Marshal(item.Metadata)

	_, err := s.db.ExecContext(ctx, "INSERT OR REPLACE INTO memory_items (id, text, vector, metadata) VALUES (?, ?, ?, ?)",
		item.ID, item.Text, string(vectorBytes), string(metadataBytes))
	return err
}

func (s *SQLiteStore) Search(ctx context.Context, queryVector []float32, limit int) ([]*MemoryItem, error) {
	// Simple fetching of all vectors to perform native cosine similarity in Go layer
	// (For massive scale, a `sqlite-vss` C extension or `pgvector` swap is assumed under this interface)
	rows, err := s.db.QueryContext(ctx, "SELECT id, text, vector, metadata FROM memory_items")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*MemoryItem
	for rows.Next() {
		var id, text, vectorStr, metaStr string
		if err := rows.Scan(&id, &text, &vectorStr, &metaStr); err != nil {
			return nil, err
		}

		var vector []float32
		json.Unmarshal([]byte(vectorStr), &vector)

		var meta map[string]interface{}
		json.Unmarshal([]byte(metaStr), &meta)

		items = append(items, &MemoryItem{
			ID:       id,
			Text:     text,
			Vector:   vector,
			Metadata: meta,
		})
	}

	// We can reuse the existing CosineSimilarity logic by loading these to a temporary struct
	tempStore := &InMemCosineStore{items: items}
	return tempStore.Search(ctx, queryVector, limit)
}
