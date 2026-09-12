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

## Decisions

- No conflict exists between the feature artifacts, constitution, AGENTS.md and `specs/ralph-implementation-rules.md` for the planned shared-interceptor design.
- T002's red test cannot be committed alone under repository quality policy, so T002 and its paired T003 implementation form one validated work-unit commit.
- T004 and T005 are paired for the same green-suite requirement; allowed correlation values are bounded identifiers, while Retry-After and Server-Timing are parsed into safe canonical forms.
- T007 through T012 form one validated lower-layer US1 slice because transport/OAuth contract tests require their paired body, trace, stack and token-client implementations to remain green.
- US2 security coverage retains safe fallback correlation identifiers while dropping unsafe first values, and collects response cookie/API-key secrets before parsing allowed response metadata.

## Gotchas

- The baseline regex intentionally has no matching tests in several auth packages; their `[no tests to run]` result is not a failure.
- Existing verbose command tests that prohibit endpoint detail must evaluate command-owned progress after removing `api #` lines; machine stdout remains byte-clean while admitted diagnostics intentionally occupy stderr.

## Reusable Commands

- Baseline: `go test ./internal/services/httpc ./internal/services/auth/... ./cmd -run 'ReadRetry|ProcessInstanceConfirmationTerminal|RootHelp' -count=1`
- Record checks: `go test ./internal/services/httpc -run 'TestAPIDiagnosticsRecord' -race -count=1`
- Sanitizer checks: `go test ./internal/services/httpc -run 'TestAPIDiagnostics' -race -count=1`
- Full gate: `make test`
- US1 transport/auth gate: `go test ./internal/services/httpc ./internal/services/auth/oauth2 -run 'TestAPIDiagnostics' -race -count=1`
- US1 command gate: `go test ./internal/services/httpc ./internal/services/auth/oauth2 ./cmd -run 'TestAPIDiagnostics|RootHelp' -count=1`
- US2 gate: `go test ./internal/services/httpc ./internal/services/auth/... ./cmd -run 'TestAPIDiagnostics|ProcessInstanceConfirmationTerminal' -count=1`

## Do Not Repeat

- Do not add diagnostics to individual commands or generated clients; keep observation at the shared HTTP transport boundary.

## Current Handoff
- Continue US3 at T024: add retry/redirect and supported-client wiring coverage, then complete T026-T028 failure, isolation and validation work.
