// Package dbtypes holds column types for the generated sqlc code.
package dbtypes

import (
	"database/sql/driver"
	"fmt"
)

// JSONText is a JSON value stored as TEXT. It scans TEXT and BLOB (SQLite
// returns either, depending on how the value was written), is written as TEXT
// and appears as raw JSON in API responses. Empty = "[]".
type JSONText []byte

// Scan implements sql.Scanner.
func (j *JSONText) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*j = JSONText("[]")
	case string:
		*j = JSONText(v)
	case []byte:
		*j = append(JSONText(nil), v...)
	default:
		return fmt.Errorf("JSONText: unsupported type %T", src)
	}
	return nil
}

// Value implements driver.Valuer (stored as TEXT).
func (j JSONText) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "[]", nil
	}
	return string(j), nil
}

// MarshalJSON writes the stored JSON as is.
func (j JSONText) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("[]"), nil
	}
	return j, nil
}
