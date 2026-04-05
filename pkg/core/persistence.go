package core

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	//"time"

	_ "github.com/mattn/go-sqlite3" // Default SQLite driver
)

// SessionManager handles saving and loading of Crew/Task state.
type SessionManager struct {
	db *sql.DB
	mu sync.Mutex
}

var (
	globalManager *SessionManager
	managerOnce   sync.Once
)

// InitSessionManager initializes the global session manager with the provided driver and connection.
func InitSessionManager(driver, connection string) (*SessionManager, error) {
	if driver == "" || connection == "" {
		return nil, fmt.Errorf("persistence driver and connection string are required")
	}

	db, err := sql.Open(driver, connection)
	if err != nil {
		return nil, err
	}

	m := &SessionManager{db: db}
	if err := m.initializeSchema(); err != nil {
		return nil, err
	}

	globalManager = m
	return m, nil
}

// GetSessionManager returns the global session manager.
func GetSessionManager() (*SessionManager, error) {
	if globalManager == nil {
		return nil, fmt.Errorf("session manager not initialized")
	}
	return globalManager, nil
}

func (m *SessionManager) initializeSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS checkpoints (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		state_blob TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_session_id ON checkpoints(session_id);
	`
	_, err := m.db.Exec(query)
	return err
}

// SaveCheckpoint persists the current state for a given session.
func (m *SessionManager) SaveCheckpoint(sessionID string, state interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	blob, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	_, err = m.db.Exec("INSERT INTO checkpoints (session_id, state_blob) VALUES (?, ?)", sessionID, string(blob))
	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return nil
}

// LoadLatestCheckpoint retrieves the most recent state for a session.
func (m *SessionManager) LoadLatestCheckpoint(sessionID string, target interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var blob string
	err := m.db.QueryRow("SELECT state_blob FROM checkpoints WHERE session_id = ? ORDER BY timestamp DESC LIMIT 1", sessionID).Scan(&blob)
	if err == sql.ErrNoRows {
		return fmt.Errorf("no checkpoint found for session: %s", sessionID)
	} else if err != nil {
		return err
	}

	return json.Unmarshal([]byte(blob), target)
}

// ListSessions returns a list of all active session IDs.
func (m *SessionManager) ListSessions() ([]string, error) {
	rows, err := m.db.Query("SELECT DISTINCT session_id FROM checkpoints")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err == nil {
			sessions = append(sessions, s)
		}
	}
	return sessions, nil
}
