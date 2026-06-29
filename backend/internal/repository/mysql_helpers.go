package repository

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// jsonArrayParam serializes a Go slice for MySQL JSON/JSON_TABLE parameters.
// Returning a string (rather than []byte) also avoids the driver treating the
// value as a binary string when MySQL validates it as JSON.
func jsonArrayParam(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

type jsonScanValue struct {
	destination any
}

func scanJSON(destination any) *jsonScanValue {
	return &jsonScanValue{destination: destination}
}

func (s *jsonScanValue) Scan(src any) error {
	if src == nil {
		return nil
	}
	var data []byte
	switch value := src.(type) {
	case []byte:
		data = value
	case string:
		data = []byte(value)
	default:
		return fmt.Errorf("scan JSON from unsupported type %T", src)
	}
	return json.Unmarshal(data, s.destination)
}

type jsonDriverValue struct {
	value any
}

func jsonValue(value any) driver.Valuer {
	return jsonDriverValue{value: value}
}

func (v jsonDriverValue) Value() (driver.Value, error) {
	return jsonArrayParam(v.value)
}

func quoteMySQLIdentifier(identifier string) string {
	return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ", ")
}

func appendInt64Args(args []any, values []int64) []any {
	for _, value := range values {
		args = append(args, value)
	}
	return args
}

func int64InCondition(column string, values []int64) (string, []any) {
	if len(values) == 0 {
		return "1 = 0", nil
	}
	return column + " IN (" + placeholders(len(values)) + ")", appendInt64Args(nil, values)
}

func int64NotInCondition(column string, values []int64) (string, []any) {
	if len(values) == 0 {
		return "1 = 1", nil
	}
	return column + " NOT IN (" + placeholders(len(values)) + ")", appendInt64Args(nil, values)
}

func mysqlNullsLastOrder(column, direction string) string {
	return "ORDER BY (" + column + " IS NULL) ASC, " + column + " " + direction
}

func numberedComment(index int) string {
	return "?/*" + strconv.Itoa(index) + "*/"
}
