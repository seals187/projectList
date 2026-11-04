// Package web bettet den statischen SvelteKit-Build ins Binary ein.
package web

import "embed"

// Build enthält den Inhalt von web/build (inkl. _app/-Verzeichnis, daher "all:").
//
//go:embed all:build
var Build embed.FS
