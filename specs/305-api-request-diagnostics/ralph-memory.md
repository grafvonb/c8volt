# Ralph Memory

Feature: 305-api-request-diagnostics
Started: 2026-09-12T17:35:53Z

## Codebase Patterns

- `httpc.New` currently builds `ReadRetry -> Log -> default transport`; `InstallAuthEditor` later adds Auth outside that chain.
- OAuth token acquisition owns a separate unauthenticated HTTP client, while cookie authentication uses the shared client.
- Real-terminal command tests use `testx.NewCmdTerminalRunner` with terminal stdin and independently captured stdout/stderr.
- Diagnostic record optional numeric and boolean evidence uses pointers so observed zero/false remains distinct from absence; `diagnosticRecord.format` owns stable field order and ASCII duration units.
- Diagnostic sanitization is exchange-owned and collects configured, URL, header and parsed cookie secrets before producing allowed request/response metadata; malformed queries fail closed without body access.

## Decisions

- No conflict exists between the feature artifacts, constitution, AGENTS.md and `specs/ralph-implementation-rules.md` for the planned shared-interceptor design.
- T002's red test cannot be committed alone under repository quality policy, so T002 and its paired T003 implementation form one validated work-unit commit.
- T004 and T005 are paired for the same green-suite requirement; allowed correlation values are bounded identifiers, while Retry-After and Server-Timing are parsed into safe canonical forms.

## Gotchas

- The baseline regex intentionally has no matching tests in several auth packages; their `[no tests to run]` result is not a failure.

## Reusable Commands

- Baseline: `go test ./internal/services/httpc ./internal/services/auth/... ./cmd -run 'ReadRetry|ProcessInstanceConfirmationTerminal|RootHelp' -count=1`
- Record checks: `go test ./internal/services/httpc -run 'TestAPIDiagnosticsRecord' -race -count=1`
- Sanitizer checks: `go test ./internal/services/httpc -run 'TestAPIDiagnostics' -race -count=1`
- Full gate: `make test`

## Do Not Repeat

- Do not add diagnostics to individual commands or generated clients; keep observation at the shared HTTP transport boundary.

## Current Handoff
- Continue with T006: implement and test the invocation collector in `internal/services/httpc/diagnostics.go`, retaining the invocation logger and resolved verbose gate, allocating atomic sequences, freezing snapshots once under private state locks, and emitting only after locks are released.
