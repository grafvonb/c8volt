# Ralph Memory

Feature: 316-delete-cancel-logging
Started: 2026-09-13T13:39:33Z

## Codebase Patterns

- Process-instance bulk completion callbacks already feed the command semantic reporter; concise per-tree failure detail can travel through the existing `FailureDetail` string without changing callback or report schemas.
- `handleCommandError` is the final shared command boundary: machine envelopes keep normalized error text/class, while process-instance failures can select a compact human message and emit the full chain at admitted DEBUG.

## Decisions

- Process-instance mutation commands retain per-key lifecycle suppression in every mode, but admit service-owned workflow explanations only for verbose one-line human output; quiet, automation, JSON, and keys-only keep workflow suppression.
- Required narration is owned by the four versioned adapters at actual transition sites. A shared wait formatter records phase, root/scope, desired states, effective timeout, and configured backoff once per real confirmation wait.
- Timeout evidence uses small internal wait/mutation wrappers with `Unwrap`; `c8volt/ferrors` projects only command-needed facts and preserves the original Error text and causes.
- The T001 fixture uses Camunda 8.9's real HTTP adapter and subprocess command path. It asserts child deletion conflict precedes accepted root cancellation, no root deletion follows timeout, and JSON remains exactly one envelope.

## Gotchas

- Waiter `InfoIfVerbose` calls must also honor process-instance detail suppression; otherwise command-level compact mode leaks one line per polling attempt while the adapter owns the single wait explanation.
- Camunda 8.9 deletion is `POST /v2/process-instances/<key>/deletion`, not HTTP DELETE.
- Concurrent waiters may end through either the timer select or a context-canceled HTTP lookup. Both branches must retain the prior successful state; machine-mode stderr may contain the existing lookup-stop diagnostic.
- `ferrors` class wrappers must preserve both the outer class precedence and the nested cause. A plain multi-`%w` format exposes causes but can let a nested normalized class win unless classification reads the outer wrapper explicitly.
- Waiter observations must remain pending until the existing continuation decision: emit `next_delay` only after the sleep completes, and omit it for terminal state, lookup failure, retry exhaustion, or interrupted/deadline stop.
- `WithSuppressNestedProcessInstanceLookupLogs` is deliberately narrower than workflow/detail suppression. Waiters add it to nested calls; v8.7 gates its tenant-safe search message, while v8.8-v8.10 gate direct-get/state messages. HTTP transport diagnostics remain unaffected.
- OAuth cache reuse is silent at DEBUG. Token fetch/store and the shared HTTP exchange diagnostic still show acquisition, while expiry-skew refresh behavior and credential redaction remain unchanged.
- The exact polling budget is deterministic through `expect process-instance`: ACTIVE for checks 1-35 and CANCELED on check 36 yields 36 state observations and 36 process-instance HTTP records, separate from the one token bootstrap exchange.
- Joined mutation errors contain an enriched delete failure wrapping the original cancel failure for each tree. Failure collection must stop at the outer domain mutation node on each joined branch to avoid double-counting roots or accepted submissions.

## Reusable Commands

- `go test ./cmd -run '^TestProcessInstanceDeleteCancellation(Timeout|Success)Transcript$' -count=1 -v`
- `go test ./cmd -run '^TestProcessInstanceDeleteCancellationTimeoutTranscript$' -count=5`
- `go test ./cmd -run '^TestProcessInstancePollingRecordBudget$' -count=1 -v`
- `go test ./internal/services/auth/oauth2 -run 'Test(APIDiagnosticsOAuth|RetrieveTokenForAPI)' -count=1`
- `go test ./internal/services/processinstance/waiter ./internal/services/processinstance/v89 ./internal/services/processinstance ./c8volt/process ./c8volt/ferrors -count=1`
- `make test`

## Do Not Repeat

- Do not make timeout transcript assertions depend on which concurrent waiter notices the deadline first.

## Current Handoff
- T006 is next: complete the combined output compatibility matrix and affected facade, adapter, process-definition-delete, prompt, request-order, and exit-classification regressions without changing output or mutation contracts.
