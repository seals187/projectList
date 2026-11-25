package service_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestLifecycleIsTheOnlyWriter: project status/board place and the activity log
// are written only by service (Lifecycle), so their side effects (activity
// entry, automation event) cannot be skipped by a handler.
func TestLifecycleIsTheOnlyWriter(t *testing.T) {
	banned := regexp.MustCompile(`\.(SetProjectBoardPlace|AddProjectActivity)\(`)
	root := filepath.Join("..", "..")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(path)
		if d.IsDir() {
			if strings.Contains(slash, "node_modules") || strings.HasSuffix(slash, "/web") || strings.HasSuffix(slash, "/.git") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") ||
			strings.Contains(slash, "internal/service/") || strings.Contains(slash, "internal/db/sqlc/") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if m := banned.Find(src); m != nil {
			t.Errorf("%s calls %s directly: go through service.Lifecycle", slash, m)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
