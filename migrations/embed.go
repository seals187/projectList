// Package migrations enthält die goose-SQL-Migrationen, eingebettet ins Binary.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
