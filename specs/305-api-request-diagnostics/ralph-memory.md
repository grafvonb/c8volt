# Ralph Memory

Feature: 305-api-request-diagnostics
Started: 2026-09-12T17:35:53Z

## Codebase Patterns

- `httpc.New` currently builds `ReadRetry -> Log -> default transport`; `InstallAuthEditor` later adds Auth outside that chain.
- OAuth token acquisition owns a separate unauthenticated HTTP client, while cookie authentication uses the shared client.
- Real-terminal command tests use `testx.NewCmdTerminalRunner` with terminal stdin and independently captured stdout/stderr.
- Diagnostic record optional numeric and boolean evidence uses pointers so observed zero/false remains distinct from absence; `diagnosticRecord.format` owns stable field order and ASCII duration units.
- Diagnostic sanitization is exchange-owned and collects configured, URL, header and parsed cookie secrets before producing allowed request/response metadata; malformed queries fail closed without body access.
- The invocation collector is absent unless verbose INFO is admitted, copies only private context/redaction seeds, allocates atomic exchange sequences, freezes deep snapshots under an exchange lock and emits through the existing logger after unlocking.
- Diagnostics are installed beneath the existing log and read-retry transports; `httptrace.WithClientTrace` composes callbacks, response termination owns final timing, and request/response wrappers retain `GetBody`, Read/Close results and `io.WriterTo` when supplied.
- `httpc.ShareDiagnostics` finds the invocation collector through known wrappers and attaches it to OAuth's timeout-preserving, unauthenticated, non-retrying token client so token and API exchanges share one sequence.
- Root bootstrap passes `flagVerbose` and the existing configured invocation logger to `httpc.WithDiagnostics` before authenticator construction, so OAuth token and cookie login traffic are observed without separate command logic.
- Root bootstrap must resolve the executing leaf's `cmd.ErrOrStderr()` before activity wrapping and set only the root to that wrapper; setting the child writer too hides its configured destination and risks stale invocation routing.
- Real-terminal diagnostic coverage can reuse the process-instance confirmation subprocess helper: terminal stdin remains genuine while stdout/stderr are independently captured, and a configured child destination can be mirrored with `io.MultiWriter` for exact routing assertions.
- Cookie login usernames must join passwords in the invocation-private credential seed because the authenticator sends both as query values; request sanitization then omits those reflected login credentials before formatting.
- Request-body reads establish explicit incomplete upload evidence until EOF or a successful `WriterTo`; request-body Close preserves the delegate result without claiming upload completion.
- Trace phase queues retain completed zero-duration samples, match overlapping TCP attempts by network/address, order output by start time, and freeze before logger emission; the shared logging writer keeps concurrently completed records indivisible.
- Read-retry closes discarded response bodies without draining them, so the diagnostic response wrapper emits one attempt record with observed zero bytes and `response-complete=false`; the eventual response remains a separate sequence.
- A single `httpc.Service` client supplied to the top-level `c8volt.New` factory reaches Camunda clients for 8.7-8.10, the v8.7 Operate adapter and v8.8/v8.9 Tasklist fallback without generated-client changes.
- Pre-header failure phase resolution prefers typed DNS/TLS/connect evidence, retains a phase only when trace failures or active setup identify one phase, and uses a successful `WroteRequest` as response-header evidence; conflicting phases are omitted.
- Client timeout finalization must compare the terminal observation with the request context deadline in addition to checking `Context.Err()`, because cancellation state can lag the transport return under race-suite load.
- Separate command invocations should be tested in subprocesses: each owns a fresh Cobra singleton, diagnostic collector, stderr destination and sequence beginning at 1.
- Root `Long`, `Example` and persistent-flag metadata are the source for generated root and inherited help; README content is also copied into the generated documentation homepage by `make docs-content`.

## Decisions

- No conflict exists between the feature artifacts, constitution, AGENTS.md and `specs/ralph-implementation-rules.md` for the planned shared-interceptor design.
- T002's red test cannot be committed alone under repository quality policy, so T002 and its paired T003 implementation form one validated work-unit commit.
- T004 and T005 are paired for the same green-suite requirement; allowed correlation values are bounded identifiers, while Retry-After and Server-Timing are parsed into safe canonical forms.
- T007 through T012 form one validated lower-layer US1 slice because transport/OAuth contract tests require their paired body, trace, stack and token-client implementations to remain green.
- US2 security coverage retains safe fallback correlation identifiers while dropping unsafe first values, and collects response cookie/API-key secrets before parsing allowed response metadata.

## Gotchas

- The baseline regex intentionally has no matching tests in several auth packages; their `[no tests to run]` result is not a failure.
- Existing verbose command tests that prohibit endpoint detail must evaluate command-owned progress after removing `api #` lines; machine stdout remains byte-clean while admitted diagnostics intentionally occupy stderr.
- The 25ms OAuth timeout test can expose scheduler-sensitive context cancellation under full race-suite load; deadline-based terminal classification removes that race without changing the client timeout.

## Reusable Commands

- Baseline: `go test ./internal/services/httpc ./internal/services/auth/... ./cmd -run 'ReadRetry|ProcessInstanceConfirmationTerminal|RootHelp' -count=1`
- Record checks: `go test ./internal/services/httpc -run 'TestAPIDiagnosticsRecord' -race -count=1`
- Sanitizer checks: `go test ./internal/services/httpc -run 'TestAPIDiagnostics' -race -count=1`
- Full gate: `make test`
- US1 transport/auth gate: `go test ./internal/services/httpc ./internal/services/auth/oauth2 -run 'TestAPIDiagnostics' -race -count=1`
- US1 command gate: `go test ./internal/services/httpc ./internal/services/auth/oauth2 ./cmd -run 'TestAPIDiagnostics|RootHelp' -count=1`
- US2 gate: `go test ./internal/services/httpc ./internal/services/auth/... ./cmd -run 'TestAPIDiagnostics|ProcessInstanceConfirmationTerminal' -count=1`
- US3 gate: `go test ./internal/services/httpc ./internal/services/auth/... ./c8volt ./cmd -run 'APIDiagnostics|ReadRetry|ProcessInstanceConfirmationTerminal|RootHelp' -race -count=1`

## Do Not Repeat

- Do not add diagnostics to individual commands or generated clients; keep observation at the shared HTTP transport boundary.

## Current Handoff
- Continue Phase 6 at T030: run `make docs-content`, inspect the generated root/inherited CLI pages and README homepage, and fix source metadata rather than generated files if needed.
