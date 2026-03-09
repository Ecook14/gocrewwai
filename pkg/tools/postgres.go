package tools

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

// PostgresTool allows agents to interact with a PostgreSQL database.
type PostgresTool struct {
	BaseTool
	ConnectionString string
	db               *sql.DB
}

func NewPostgresTool(connStr string) (*PostgresTool, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &PostgresTool{
		BaseTool: BaseTool{
			NameValue: "PostgresTool",
			DescriptionValue: "Execute SQL queries against a PostgreSQL database. Input requires 'query' (the SQL statement). " +
				"Supports SELECT, INSERT, UPDATE, and DELETE. Returns results as a formatted string or affected rows count.",
		},
		ConnectionString: connStr,
		db:               db,
	}, nil
}


func (t *PostgresTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	queryRaw, ok := input["query"]
	if !ok {
		return "", fmt.Errorf("missing 'query' in input")
	}
	query, ok := queryRaw.(string)
	if !ok {
		return "", fmt.Errorf("'query' must be a string")
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))
	if strings.HasPrefix(queryLower, "select") {
		return t.executeSelect(ctx, query)
	}

	return t.executeExec(ctx, query)
}

func (t *PostgresTool) executeSelect(ctx context.Context, query string) (string, error) {
	rows, err := t.db.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("sql query failed: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(strings.Join(cols, " | ") + "\n")
	sb.WriteString(strings.Repeat("-", len(cols)*10) + "\n")

	for rows.Next() {
		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return "", err
		}

		rowStrs := make([]string, len(cols))
		for i, v := range values {
			if v == nil {
				rowStrs[i] = "NULL"
			} else {
				rowStrs[i] = fmt.Sprintf("%v", v)
			}
		}
		sb.WriteString(strings.Join(rowStrs, " | ") + "\n")
	}

	return sb.String(), nil
}

func (t *PostgresTool) executeExec(ctx context.Context, query string) (string, error) {
	res, err := t.db.ExecContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("sql exec failed: %w", err)
	}

	rows, _ := res.RowsAffected()
	return fmt.Sprintf("Success. Rows affected: %d", rows), nil
}

func (t *PostgresTool) RequiresReview() bool { return true } // SQL interaction should be reviewed
