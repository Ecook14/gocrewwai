package tools

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteTool allows agents to interact with a SQLite database.
type SQLiteTool struct {
	BaseTool
	DBPath string
	db     *sql.DB
}

// NewSQLiteTool creates a new SQLite tool with the given database path.
func NewSQLiteTool(dbPath string) (*SQLiteTool, error) {
	if dbPath == "" {
		dbPath = "gocrew.db"
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite: %w", err)
	}

	return &SQLiteTool{
		BaseTool: BaseTool{
			NameValue:        "SQLiteTool",
			DescriptionValue: "Execute SQL queries against a SQLite database. Input: {'query': 'SQL statement'}. Supports SELECT, INSERT, UPDATE, DELETE. Returns formatted result rows or affected count.",
		},
		DBPath: dbPath,
		db:     db,
	}, nil
}

func (t *SQLiteTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	query, ok := input["query"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'query' in input")
	}
	if len(query) > 10000 {
		return "", fmt.Errorf("query too long (max 10000 chars)")
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))
	if strings.HasPrefix(queryLower, "select") || strings.HasPrefix(queryLower, "show") ||
		strings.HasPrefix(queryLower, "pragma") {
		return t.executeSelect(ctx, query)
	}
	return t.executeExec(ctx, query)
}

func (t *SQLiteTool) CacheFunction(input map[string]interface{}) string { return "" }

func (t *SQLiteTool) executeSelect(ctx context.Context, query string) (string, error) {
	rows, err := t.db.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("sqlite query failed: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(strings.Join(cols, " | ") + "\n")
	sb.WriteString(strings.Repeat("-", len(cols)*12) + "\n")

	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return "", err
		}
		strs := make([]string, len(cols))
		for i, v := range values {
			if v == nil {
				strs[i] = "NULL"
			} else if b, ok := v.([]byte); ok {
				strs[i] = string(b)
			} else {
				strs[i] = fmt.Sprintf("%v", v)
			}
		}
		sb.WriteString(strings.Join(strs, " | ") + "\n")
	}
	return sb.String(), nil
}

func (t *SQLiteTool) executeExec(ctx context.Context, query string) (string, error) {
	res, err := t.db.ExecContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("sqlite exec failed: %w", err)
	}
	affected, _ := res.RowsAffected()
	lastID, _ := res.LastInsertId()
	if lastID > 0 {
		return fmt.Sprintf("Success. Rows affected: %d, Last Insert ID: %d", affected, lastID), nil
	}
	return fmt.Sprintf("Success. Rows affected: %d", affected), nil
}

func (t *SQLiteTool) RequiresReview() bool { return true }
func (t *SQLiteTool) Name() string         { return t.BaseTool.NameValue }
func (t *SQLiteTool) Description() string  { return t.BaseTool.DescriptionValue }
func (t *SQLiteTool) Close() error         { return t.db.Close() }
