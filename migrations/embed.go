// Package migrations contains the goose SQL migrations, embedded in the binary.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
