package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore is a persistent implementation of the Store interface using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore initializes a new SQLite database for persistent memory.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	// Create table with TTL support
	createTable := `
	CREATE TABLE IF NOT EXISTS memory_items (
		id TEXT PRIMARY KEY,
		text TEXT,
		vector TEXT,
		metadata TEXT,
		created_at TEXT DEFAULT (datetime('now')),
		expires_at TEXT DEFAULT ''
	);`
	_, err = db.Exec(createTable)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory_items table: %w", err)
	}

	// Add columns if upgrading from old schema (silently ignores if they exist)
	db.Exec("ALTER TABLE memory_items ADD COLUMN created_at TEXT DEFAULT ''")
	db.Exec("ALTER TABLE memory_items ADD COLUMN expires_at TEXT DEFAULT ''")

	// Performance and concurrency tuning
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA synchronous=NORMAL")

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Add(ctx context.Context, item *MemoryItem) error {
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}

	vectorJSON, err := json.Marshal(item.Vector)
	if err != nil {
		return fmt.Errorf("failed to marshal vector: %w", err)
	}

	metadataJSON, err := json.Marshal(item.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	expiresAt := ""
	if !item.ExpiresAt.IsZero() {
		expiresAt = item.ExpiresAt.Format(time.RFC3339)
	}

	query := `INSERT OR REPLACE INTO memory_items (id, text, vector, metadata, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = s.db.ExecContext(ctx, query, item.ID, item.Text, string(vectorJSON), string(metadataJSON),
		item.CreatedAt.Format(time.RFC3339), expiresAt)
	if err != nil {
		return fmt.Errorf("failed to insert memory item: %w", err)
	}

	return nil
}

// BulkAdd inserts multiple items in a single transaction.
func (s *SQLiteStore) BulkAdd(ctx context.Context, items []*MemoryItem) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT OR REPLACE INTO memory_items (id, text, vector, metadata, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, item := range items {
		if item.CreatedAt.IsZero() {
			item.CreatedAt = time.Now()
		}

		vectorJSON, err := json.Marshal(item.Vector)
		if err != nil {
			return fmt.Errorf("failed to marshal vector for item %s: %w", item.ID, err)
		}
		metadataJSON, err := json.Marshal(item.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata for item %s: %w", item.ID, err)
		}

		expiresAt := ""
		if !item.ExpiresAt.IsZero() {
			expiresAt = item.ExpiresAt.Format(time.RFC3339)
		}

		_, err = stmt.ExecContext(ctx, item.ID, item.Text, string(vectorJSON), string(metadataJSON),
			item.CreatedAt.Format(time.RFC3339), expiresAt)
		if err != nil {
			return fmt.Errorf("failed to insert item %s: %w", item.ID, err)
		}
	}

	return tx.Commit()
}

// Delete removes a memory item by ID.
func (s *SQLiteStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM memory_items WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete memory item: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("memory item not found: %s", id)
	}
	return nil
}

// Count returns the number of non-expired items.
func (s *SQLiteStore) Count(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM memory_items WHERE expires_at = '' OR expires_at > datetime('now')`).Scan(&count)
	return count, err
}

// Reset clears all data by truncating the table.
func (s *SQLiteStore) Reset(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM memory_items`)
	if err != nil {
		return fmt.Errorf("failed to reset memory store: %w", err)
	}
	return nil
}

// PurgeExpired removes all items that have passed their TTL.
func (s *SQLiteStore) PurgeExpired(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM memory_items WHERE expires_at != '' AND expires_at <= datetime('now')`)
	if err != nil {
		return 0, fmt.Errorf("failed to purge expired items: %w", err)
	}
	return result.RowsAffected()
}

func (s *SQLiteStore) Search(ctx context.Context, queryVector []float32, limit int) ([]*MemoryItem, error) {
	// Purge expired items before searching
	s.PurgeExpired(ctx)

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, text, vector, metadata, created_at, expires_at FROM memory_items`)
	if err != nil {
		return nil, fmt.Errorf("failed to query memory items: %w", err)
	}
	defer rows.Close()

	var items []*MemoryItem
	for rows.Next() {
		var item MemoryItem
		var vectorJSON, metadataJSON, createdStr, expiresStr string
		if err := rows.Scan(&item.ID, &item.Text, &vectorJSON, &metadataJSON, &createdStr, &expiresStr); err != nil {
			return nil, fmt.Errorf("failed to scan memory item: %w", err)
		}

		if err := json.Unmarshal([]byte(vectorJSON), &item.Vector); err != nil {
			return nil, fmt.Errorf("failed to unmarshal vector: %w", err)
		}
		if err := json.Unmarshal([]byte(metadataJSON), &item.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
		if createdStr != "" {
			item.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		}
		if expiresStr != "" {
			item.ExpiresAt, _ = time.Parse(time.RFC3339, expiresStr)
		}

		items = append(items, &item)
	}

	type scoredItem struct {
		item  *MemoryItem
		score float32
	}

	var results []scoredItem
	for _, item := range items {
		if len(item.Vector) != len(queryVector) {
			continue
		}
		sim, err := CosineSimilarity(queryVector, item.Vector)
		if err != nil {
			return nil, err
		}
		results = append(results, scoredItem{item: item, score: sim})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	var out []*MemoryItem
	for i := 0; i < limit && i < len(results); i++ {
		out = append(out, results[i].item)
	}

	return out, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
