// Command gentypes writes the HTTP contract (enums, response shapes, error
// codes) as TypeScript to web/src/lib/api.gen.ts. Run from the repository
// root: go run ./cmd/gentypes
package main

import (
	"fmt"
	"os"

	"github.com/seals187/projectList/internal/httpapi"
)

const out = "web/src/lib/api.gen.ts"

func main() {
	b, err := httpapi.TypeScriptContract("internal/httpapi")
	if err != nil {
		fmt.Fprintln(os.Stderr, "gentypes:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "gentypes:", err)
		os.Exit(1)
	}
	fmt.Println("wrote", out)
}
