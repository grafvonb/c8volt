# Ralph Memory

Feature: 305-api-request-diagnostics
Started: 2026-09-12T17:35:53Z

## Codebase Patterns

- `httpc.New` currently builds `ReadRetry -> Log -> default transport`; `InstallAuthEditor` later adds Auth outside that chain.
- OAuth token acquisition owns a separate unauthenticated HTTP client, while cookie authentication uses the shared client.
- Real-terminal command tests use `testx.NewCmdTerminalRunner` with terminal stdin and independently captured stdout/stderr.

## Decisions

- No conflict exists between the feature artifacts, constitution, AGENTS.md and `specs/ralph-implementation-rules.md` for the planned shared-interceptor design.

## Gotchas

- The baseline regex intentionally has no matching tests in several auth packages; their `[no tests to run]` result is not a failure.

## Reusable Commands

- Baseline: `go test ./internal/services/httpc ./internal/services/auth/... ./cmd -run 'ReadRetry|ProcessInstanceConfirmationTerminal|RootHelp' -count=1`

## Do Not Repeat

- Do not add diagnostics to individual commands or generated clients; keep observation at the shared HTTP transport boundary.

## Current Handoff
- Continue with T002: add record contract tests in `internal/services/httpc/diagnostics_record_test.go`, confirm they fail for the missing implementation, and keep T003 for a later coordinated work unit.
