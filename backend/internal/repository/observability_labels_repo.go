package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// loadObservabilityDimensionNames resolves display names for the small set of
// dimensions returned by a heatmap. Raw dimension keys remain the stable API
// identifiers; names are presentation-only metadata.
func loadObservabilityDimensionNames(
	ctx context.Context,
	db *sql.DB,
	dimension string,
	ids []int64,
) (map[int64]string, error) {
	names := make(map[int64]string)
	if db == nil || len(ids) == 0 {
		return names, nil
	}

	var table string
	var nameColumn string
	switch dimension {
	case "user_id":
		table = "`users`"
		nameColumn = "email"
	case "group_id":
		table = "`groups`"
		nameColumn = "name"
	case "account_id":
		table = "`accounts`"
		nameColumn = "name"
	default:
		return names, nil
	}

	unique := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return names, nil
	}

	placeholders := make([]string, len(unique))
	args := make([]any, len(unique))
	for index, id := range unique {
		placeholders[index] = "?"
		args[index] = id
	}
	query := fmt.Sprintf(
		"SELECT id, %s FROM %s WHERE id IN (%s)",
		nameColumn,
		table,
		strings.Join(placeholders, ","),
	)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return names, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id int64
		var name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return names, err
		}
		if value := strings.TrimSpace(name.String); value != "" {
			names[id] = value
		}
	}
	return names, rows.Err()
}
