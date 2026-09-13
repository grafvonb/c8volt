# Ralph Memory

Feature: 316-delete-cancel-logging
Started: 2026-09-13T13:39:33Z

## Codebase Patterns

- Process-instance bulk completion callbacks already feed the command semantic reporter; concise per-tree failure detail can travel through the existing `FailureDetail` string without changing callback or report schemas.
- `handleCommandError` is the final shared command boundary: machine envelopes keep normalized error text/class, while process-instance failures can select a compact human message and emit the full chain at admitted DEBUG.

## Decisions

- Timeout evidence uses small internal wait/mutation wrappers with `Unwrap`; `c8volt/ferrors` projects only command-needed facts and preserves the original Error text and causes.
- The T001 fixture uses Camunda 8.9's real HTTP adapter and subprocess command path. It asserts child deletion conflict precedes accepted root cancellation, no root deletion follows timeout, and JSON remains exactly one envelope.

## Gotchas

- Camunda 8.9 deletion is `POST /v2/process-instances/<key>/deletion`, not HTTP DELETE.
- Concurrent waiters may end through either the timer select or a context-canceled HTTP lookup. Both branches must retain the prior successful state; machine-mode stderr may contain the existing lookup-stop diagnostic.
- `ferrors` class wrappers must preserve both the outer class precedence and the nested cause. A plain multi-`%w` format exposes causes but can let a nested normalized class win unless classification reads the outer wrapper explicitly.

## Reusable Commands

- `go test ./cmd -run '^TestProcessInstanceDeleteCancellationTimeoutTranscript$' -count=5`
- `go test ./internal/services/processinstance/waiter ./internal/services/processinstance/v89 ./internal/services/processinstance ./c8volt/process ./c8volt/ferrors -count=1`
- `make test`

## Do Not Repeat

- Do not make timeout transcript assertions depend on which concurrent waiter notices the deadline first.

## Current Handoff
- T002 is next per the mandated execution order. T001 is green; T005 remains incomplete and should build on T002's consolidated waiter observations before extending the failure facts across all adapters and edge cases.
