package httpapi

import (
	"bytes"
	"os"
	"testing"
)

// TestContractUpToDate fails when web/src/lib/api.gen.ts no longer matches the
// Go side. Fix: go run ./cmd/gentypes
func TestContractUpToDate(t *testing.T) {
	want, err := TypeScriptContract(".")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile("../../web/src/lib/api.gen.ts")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.ReplaceAll(got, []byte("\r\n"), []byte("\n")), want) {
		t.Fatal("web/src/lib/api.gen.ts is stale: run `go run ./cmd/gentypes`")
	}
}
