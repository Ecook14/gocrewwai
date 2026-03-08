package tools

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLTool allows agents to interact with a MySQL/MariaDB database.
// Input: {"query": "SELECT * FROM users WHERE id = 1"}
type MySQLTool struct {
	ConnectionString string
	db               *sql.DB
}

// NewMySQLTool creates a new MySQL tool with the given DSN.
// DSN format: user:password@tcp(host:port)/dbname
func NewMySQLTool(dsn string) (*MySQLTool, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping mysql: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	return &MySQLTool{ConnectionString: dsn, db: db}, nil
}

func (t *MySQLTool) Name() string { return "MySQLTool" }

func (t *MySQLTool) Description() string {
	return "Execute SQL queries against a MySQL/MariaDB database. " +
		"Input: {'query': 'SQL statement'}. Supports SELECT, INSERT, UPDATE, DELETE. " +
		"Returns formatted result rows or affected count."
}

func (t *MySQLTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	query, ok := input["query"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'query' in input")
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))
	if strings.HasPrefix(queryLower, "select") || strings.HasPrefix(queryLower, "show") ||
		strings.HasPrefix(queryLower, "describe") || strings.HasPrefix(queryLower, "explain") {
		return t.executeSelect(ctx, query)
	}
	return t.executeExec(ctx, query)
}

func (t *MySQLTool) executeSelect(ctx context.Context, query string) (string, error) {
	rows, err := t.db.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("mysql query failed: %w", err)
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

func (t *MySQLTool) executeExec(ctx context.Context, query string) (string, error) {
	res, err := t.db.ExecContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("mysql exec failed: %w", err)
	}
	affected, _ := res.RowsAffected()
	lastID, _ := res.LastInsertId()
	if lastID > 0 {
		return fmt.Sprintf("Success. Rows affected: %d, Last Insert ID: %d", affected, lastID), nil
	}
	return fmt.Sprintf("Success. Rows affected: %d", affected), nil
}

func (t *MySQLTool) RequiresReview() bool { return true }

func (t *MySQLTool) Close() error { return t.db.Close() }
