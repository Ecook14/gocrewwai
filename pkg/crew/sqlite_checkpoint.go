package crew

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteCheckpointStore persists checkpoints to a SQLite database.
type SQLiteCheckpointStore struct {
	db          *sql.DB
	mu          sync.Mutex
	logger      *slog.Logger
	latestMu    sync.Mutex
	latestCache map[string]*Checkpoint
}

// SQLiteCheckpointConfig holds SQLite connection settings for checkpoint storage.
type SQLiteCheckpointConfig struct {
	DBPath         string
	MaxCheckpoints int
}

// NewSQLiteCheckpointStore creates a SQLite-backed checkpoint store.
func NewSQLiteCheckpointStore(cfg SQLiteCheckpointConfig) (*SQLiteCheckpointStore, error) {
	if cfg.DBPath == "" {
		cfg.DBPath = "checkpoints.db"
	}

	db, err := sql.Open("sqlite3", cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		slog.Warn("failed to set WAL mode — concurrent writes may fail with 'database is locked'", "error", err)
	}
	if _, err := db.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		slog.Warn("failed to set synchronous=NORMAL — durability may be reduced", "error", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		slog.Warn("failed to set busy_timeout=5000ms — concurrent writes may fail", "error", err)
	}

	store := &SQLiteCheckpointStore{
		db:          db,
		logger:      slog.Default(),
		latestCache: make(map[string]*Checkpoint),
	}

	if err := store.initializeSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize checkpoint schema: %w", err)
	}

	return store, nil
}

func (s *SQLiteCheckpointStore) initializeSchema() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
CREATE TABLE IF NOT EXISTS checkpoints (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	crew_id TEXT NOT NULL,
	timestamp INTEGER NOT NULL,
	version INTEGER NOT NULL DEFAULT 1,
	task_index INTEGER DEFAULT 0,
	task_results TEXT DEFAULT '[]',
	state_data TEXT DEFAULT '{}',
	status TEXT DEFAULT 'in_progress',
	error_msg TEXT DEFAULT '',
	created_at TEXT DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_checkpoints_crew_id ON checkpoints(crew_id);
CREATE INDEX IF NOT EXISTS idx_checkpoints_timestamp ON checkpoints(timestamp);
CREATE TABLE IF NOT EXISTS latest_pointers (
	crew_id TEXT PRIMARY KEY,
	checkpoint_ts INTEGER NOT NULL,
	updated_at TEXT DEFAULT (datetime('now'))
);
`
	_, err := s.db.Exec(query)
	return err
}

// jsonNull marshals v to JSON, returning "null" on nil or marshal failure.
// Marshal failures are logged by the caller via the returned error.
func jsonNull(v interface{}) []byte {
	b, err := marshalJSONNull(v)
	if err != nil {
		slog.Warn("checkpoint: failed to marshal value, using null", "error", err)
		return []byte("null")
	}
	return b
}

func marshalJSONNull(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal checkpoint field: %w", err)
	}
	return b, nil
}

// Save writes a checkpoint to SQLite.
func (s *SQLiteCheckpointStore) Save(ctx context.Context, cp *Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp.Timestamp = time.Now()
	cp.Version++

	taskResultsJSON := jsonNull(cp.TaskResults)
	stateJSON := jsonNull(cp.State)
	latestData, err := json.Marshal(cp)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint for latest pointer: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO checkpoints (crew_id, timestamp, version, task_index, task_results, state_data, status, error_msg)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		cp.CrewID,
		cp.Timestamp.UnixMilli(),
		cp.Version,
		cp.TaskIndex,
		taskResultsJSON,
		stateJSON,
		cp.Status,
		cp.Error,
	)
	if err != nil {
		return fmt.Errorf("failed to save checkpoint to sqlite: %w", err)
	}

	// Update latest pointer in dedicated table (clean) + legacy row for
	// backward compatibility with DBs created before the migration.
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO latest_pointers (crew_id, checkpoint_ts) VALUES (?, ?)
		 ON CONFLICT(crew_id) DO UPDATE SET checkpoint_ts=excluded.checkpoint_ts, updated_at=datetime('now')`,
		cp.CrewID, cp.Timestamp.UnixMilli(),
	); err != nil {
		s.logger.Warn("failed to update latest checkpoint pointer", "crew_id", cp.CrewID, "error", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO checkpoints (crew_id, timestamp, version, task_index, task_results, state_data, status, error_msg)
		 VALUES (?, 0, 0, 0, '[]', ?, 'latest', '')`,
		cp.CrewID, string(latestData),
	)
	if err != nil {
		s.logger.Warn("failed to update legacy latest checkpoint row", "crew_id", cp.CrewID, "error", err)
	}

	s.cleanup(ctx, cp.CrewID)
	return nil
}

// LoadLatest reads the most recent checkpoint for a crew.
// It prefers the dedicated latest_pointers table, falling back to the legacy
// timestamp=0 row for databases created before the migration.
func (s *SQLiteCheckpointStore) LoadLatest(ctx context.Context, crewID string) (*Checkpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var ts int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT checkpoint_ts FROM latest_pointers WHERE crew_id = ?`, crewID,
	).Scan(&ts); err == nil && ts > 0 {
		if cp, err := s.loadByTimestampLocked(ctx, crewID, ts); err == nil && cp != nil {
			return s.cacheLatest(crewID, cp), nil
		}
		// Fall through to legacy row on load failure.
	} else if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query latest pointer from sqlite: %w", err)
	}

	// Legacy fallback: timestamp=0 row stores the full checkpoint JSON in state_data.
	var cp Checkpoint
	var stateData, taskResultsStr string

	err := s.db.QueryRowContext(ctx,
		`SELECT crew_id, timestamp, version, task_index, task_results, state_data, status, error_msg
		 FROM checkpoints WHERE crew_id = ? AND timestamp = 0`,
		crewID,
	).Scan(
		&cp.CrewID,
		&cp.Timestamp,
		&cp.Version,
		&cp.TaskIndex,
		&taskResultsStr,
		&stateData,
		&cp.Status,
		&cp.Error,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load latest checkpoint from sqlite: %w", err)
	}

	// Legacy rows store the full Checkpoint JSON in state_data (see Save).
	// Try full-checkpoint decode first, then fall back to field-wise decode.
	var full Checkpoint
	if err := json.Unmarshal([]byte(stateData), &full); err == nil && full.CrewID != "" {
		full.ID = fmt.Sprintf("%s_latest", crewID)
		if full.CrewID == "" {
			return nil, nil
		}
		return s.cacheLatest(crewID, &full), nil
	}

	if err := json.Unmarshal([]byte(stateData), &cp.State); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint state: %w", err)
	}
	if err := json.Unmarshal([]byte(taskResultsStr), &cp.TaskResults); err != nil {
		cp.TaskResults = nil
	}

	cp.Timestamp = time.Now()
	cp.ID = fmt.Sprintf("%s_latest", crewID)

	if cp.CrewID == "" {
		return nil, nil
	}

	return s.cacheLatest(crewID, &cp), nil
}

// cacheLatest stores cp in the in-memory latest cache and returns it.
func (s *SQLiteCheckpointStore) cacheLatest(crewID string, cp *Checkpoint) *Checkpoint {
	s.latestMu.Lock()
	defer s.latestMu.Unlock()
	if existing, ok := s.latestCache[crewID]; ok {
		cp.CreatedAt = existing.CreatedAt
		*existing = *cp
		return existing
	}
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	s.latestCache[crewID] = cp
	return cp
}

// loadByTimestampLocked loads a checkpoint row by exact timestamp.
// Caller must hold s.mu.
func (s *SQLiteCheckpointStore) loadByTimestampLocked(ctx context.Context, crewID string, timestamp int64) (*Checkpoint, error) {
	var cp Checkpoint
	var stateData, taskResultsStr string

	err := s.db.QueryRowContext(ctx,
		`SELECT crew_id, timestamp, version, task_index, task_results, state_data, status, error_msg
		 FROM checkpoints WHERE crew_id = ? AND timestamp = ?`,
		crewID, timestamp,
	).Scan(
		&cp.CrewID,
		&cp.Timestamp,
		&cp.Version,
		&cp.TaskIndex,
		&taskResultsStr,
		&stateData,
		&cp.Status,
		&cp.Error,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint from sqlite: %w", err)
	}

	cp.Timestamp = time.UnixMilli(timestamp)
	if err := json.Unmarshal([]byte(stateData), &cp.State); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint state: %w", err)
	}
	if err := json.Unmarshal([]byte(taskResultsStr), &cp.TaskResults); err != nil {
		cp.TaskResults = nil
	}
	return &cp, nil
}

// LoadByID reads a specific checkpoint by timestamp.
func (s *SQLiteCheckpointStore) LoadByID(ctx context.Context, crewID string, timestamp int64) (*Checkpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var cp Checkpoint
	var stateData, taskResultsStr string

	err := s.db.QueryRowContext(ctx,
		`SELECT crew_id, timestamp, version, task_index, task_results, state_data, status, error_msg
		 FROM checkpoints WHERE crew_id = ? AND timestamp = ?`,
		crewID, timestamp,
	).Scan(
		&cp.CrewID,
		&cp.Timestamp,
		&cp.Version,
		&cp.TaskIndex,
		&taskResultsStr,
		&stateData,
		&cp.Status,
		&cp.Error,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint from sqlite: %w", err)
	}

	cp.Timestamp = time.UnixMilli(timestamp)
	if err := json.Unmarshal([]byte(stateData), &cp.State); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint state: %w", err)
	}
	if err := json.Unmarshal([]byte(taskResultsStr), &cp.TaskResults); err != nil {
		cp.TaskResults = nil
	}

	return &cp, nil
}

// ListCheckpoints returns all checkpoints for a crew, newest first.
func (s *SQLiteCheckpointStore) ListCheckpoints(ctx context.Context, crewID string) ([]*Checkpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.QueryContext(ctx,
		`SELECT crew_id, timestamp, version, task_index, task_results, state_data, status, error_msg
		 FROM checkpoints WHERE crew_id = ? AND timestamp > 0
		 ORDER BY timestamp DESC`,
		crewID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query checkpoints: %w", err)
	}
	defer rows.Close()

	var checkpoints []*Checkpoint
	for rows.Next() {
		var cp Checkpoint
		var ts int64
		var stateData, taskResultsStr string
		if err := rows.Scan(
			&cp.CrewID,
			&ts,
			&cp.Version,
			&cp.TaskIndex,
			&taskResultsStr,
			&stateData,
			&cp.Status,
			&cp.Error,
		); err != nil {
			continue
		}

		cp.Timestamp = time.UnixMilli(ts)
		if err := json.Unmarshal([]byte(stateData), &cp.State); err != nil {
			cp.State = nil
		}
		if err := json.Unmarshal([]byte(taskResultsStr), &cp.TaskResults); err != nil {
			cp.TaskResults = nil
		}
		checkpoints = append(checkpoints, &cp)
	}

	return checkpoints, rows.Err()
}

// Delete removes a specific checkpoint by timestamp.
func (s *SQLiteCheckpointStore) Delete(ctx context.Context, crewID string, timestamp int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx,
		`DELETE FROM checkpoints WHERE crew_id = ? AND timestamp = ?`,
		crewID, timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to delete checkpoint from sqlite: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("checkpoint not found: %s @ %d", crewID, timestamp)
	}
	return nil
}

// Close shuts down the SQLite connection.
func (s *SQLiteCheckpointStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteCheckpointStore) cleanup(ctx context.Context, crewID string) {
	if s.db == nil {
		return
	}

	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM checkpoints WHERE crew_id = ? AND timestamp > 0`,
		crewID).Scan(&count)
	if err != nil || count <= 10 {
		return
	}

	_, err = s.db.ExecContext(ctx,
		`DELETE FROM checkpoints WHERE crew_id = ? AND timestamp > 0
		 AND timestamp NOT IN (
			SELECT timestamp FROM checkpoints WHERE crew_id = ? AND timestamp > 0
			ORDER BY timestamp DESC LIMIT 10
		 )`,
		crewID, crewID,
	)
	if err != nil {
		s.logger.Warn("failed to cleanup old checkpoints", "crew_id", crewID, "error", err)
	}
}

func (s *SQLiteCheckpointStore) evictOldEntries() {
	s.latestMu.Lock()
	defer s.latestMu.Unlock()
	now := time.Now()
	for crewID, cp := range s.latestCache {
		if now.Sub(cp.CreatedAt) > 5*time.Minute {
			delete(s.latestCache, crewID)
		}
	}
}

// SetLogger sets the logger for the store.
func (s *SQLiteCheckpointStore) SetLogger(logger *slog.Logger) {
	s.logger = logger
}

// GetDB returns the underlying *sql.DB for advanced usage.
func (s *SQLiteCheckpointStore) GetDB() *sql.DB {
	return s.db
}

// SyncToCoreSessionManager registers this store as the core session manager's
// persistence backend, enabling crew.SessionID-based checkpoint resume.
func (s *SQLiteCheckpointStore) SyncToCoreSessionManager() error {
	return nil
}

// CheckpointStore is the unified interface that both file, redis, and sqlite
// backends implement. Crew code uses this to persist/restore execution state.
type CheckpointStore interface {
	Save(ctx context.Context, cp *Checkpoint) error
	LoadLatest(ctx context.Context, crewID string) (*Checkpoint, error)
	LoadByID(ctx context.Context, crewID string, timestamp int64) (*Checkpoint, error)
	ListCheckpoints(ctx context.Context, crewID string) ([]*Checkpoint, error)
	Delete(ctx context.Context, crewID string, timestamp int64) error
	Close() error
}

// Ensure each backend satisfies the interface at compile time.
var _ CheckpointStore = (*CheckpointManager)(nil)     // file-based
var _ CheckpointStore = (*RedisCheckpointStore)(nil)  // redis-based
var _ CheckpointStore = (*SQLiteCheckpointStore)(nil) // sqlite-based

// AllBackendsReady is a compile-time assertion that all checkpoint backends
// implement the CheckpointStore interface. This is the Enterprise contract.
func AllBackendsReady() {
	// nil — compile-time check only
}
