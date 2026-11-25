# ProjectList

- **Toolchains run in Docker only**: the host has no Go, Node or Python. Checks: `scripts/test-go.sh` and `scripts/test-web.sh` (each takes an optional command, e.g. one test file). CI runs the same checks through the Dockerfile's `test` target.
- **Project changes with side effects** (status, board moves, tags, create/duplicate, from-template, reordering, activity log) go through `service.Lifecycle` in `internal/service/lifecycle.go`.
- **Single-entity reads** check customer access in `internal/httpapi/scope.go` (`load*` / `requireScope`).
- **HTTP contract**: after changing a response type, enum or error code in Go, run `scripts/test-go.sh "go run ./cmd/gentypes"` to regenerate `web/src/lib/api.gen.ts`.
- **Frontend read model and writes**: `web/src/lib/workspace.svelte.ts`; synced per-user prefs: `web/src/lib/syncedPref.svelte.ts`; overview/tasks filtering and sorting: `web/src/lib/projectQuery.ts`.
- Review rules: `CODING_STANDARDS.md`.
