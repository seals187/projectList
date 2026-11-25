# Coding standards

Read during review. Mechanical rules are tests (`internal/service/seams_test.go`, `web/src/lib/seams.test.ts`, `web/src/lib/contract.test.ts`); these are the judgement calls around them.

## Backend

- **Lifecycle owns side effects.** A new kind of project change that should appear in the activity log or trigger automations is a `service.Lifecycle` method, called by the handler. Handlers decode, validate, check scope, call, encode.
- **Scope every entity read.** A handler that takes a customer-owned id loads it through a `load*` helper in `scope.go` before using it. A new customer-owned entity gets its own loader there and a row in `TestCustomerScope`.
- **Contract in sync.** A new response shape the web app relies on is added to `contractTypes` in `internal/httpapi/contract.go`, with a matching check in `contract.test.ts`. New error codes are snake_case and get an `errors.<code>` translation in all three languages.

## Frontend

- **Writes through the workspace.** A write that changes what the sidebar shows is a `workspace.*` operation (optimistic update, rollback, reload), not a component-side mutation of `sidebar`.
- **Prefs through `syncedPref`.** A new per-user preference saved to the server is a `syncedPref` instance.
- **One definition of a concept.** Due, open, progress and overdue come from `projectQuery.ts`; a view needing a variant adds a named concept there.
- **Logic outside components.** State transitions and calculations go in a plain `.ts` module with tests; `.svelte` files render and translate events.
