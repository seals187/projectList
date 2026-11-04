// Package web embeds the static SvelteKit build into the binary.
package web

import "embed"

// Build holds the contents of web/build (incl. the _app/ directory, hence "all:").
//
//go:embed all:build
var Build embed.FS
