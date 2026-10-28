# Research: User-Task Assignment and Unassignment

**Date**: 2026-10-28
**Feature**: [spec.md](spec.md)
**Scope**: Repository-grounded design research; no implementation, runtime tests, service discovery, or live-environment validation.

## 1. Supported-version endpoint and response contract

**Decision**: Use the existing generated native assignment and unassignment operations in all three adapters. Require HTTP 204 with valid response metadata and no overriding transport error to establish acceptance.

| Adapter | Generated assignment operation | Assignment route | Unassignment route | Documented success |
| --- | --- | --- | --- | --- |
| v88 | `AssignUserTaskWithResponse` | POST `/user-tasks/{userTaskKey}/assignment` | DELETE `/user-tasks/{userTaskKey}/assignee` | 204 |
| v89 | `AssignUserTaskWithResponse` | Same | Same | 204 |
| v810 | `AssignUserTaskWithResponse` | Same | Same | 204 |

Unassignment uses `UnassignUserTaskWithResponse` with no request body. Assignment supplies a nonnil assignee pointer and a nonnil override pointer. `AllowOverride *bool` has no `omitempty`; nil sends null, which is not explicit protection. Send false by default and true only with explicit permission. Leave optional custom `Action` unset to preserve the backend default action. Do not add Tasklist-v1 fallback or hand-edit generated clients.

**Rationale/evidence**:

- `internal/clients/camunda/v88/camunda/client.gen.go` and `v89/camunda/client.gen.go`: `UserTaskAssignmentRequest` around line 7524; builders around 23223; response contracts/parsers preserve status metadata.
- `internal/clients/camunda/v810/camunda/client.gen.go`: request around line 10777 and builders around 32406.
- `api/camunda/zeebe/gateway-protocol/src/main/proto/v2/user-tasks.yaml`: assignment around line 53 and unassignment around line 240 document 204, blocking task listeners, and possible 504; override default is permissive.
- Typed error responses include 400, 404, 409, 500, 503 and 504. Raw status handling must also preserve authentication/authorization errors even when no typed response field exists. A nonnil generated response alone proves nothing about success.

**Alternatives considered**: Default/null override weakens safety; read-before-write checks race with assignment changes; unassign-then-assign introduces a second mutation; generated-client regeneration is unnecessary.

**Evidence limit**: `api/3-generate-clients-from-fetched-specs.sh` generates v88/v89 from the same prepared working specification. Matching generated clients establish the repository's contract, not independently verified behavior of every released patch version. `internal/clients/camunda/v810/camunda/provenance.json` pins tag `8.10.3`, commit `3961b79c3fdaf43c48691ed77672d6db42d0abcc`. Real-engine listener behavior remains unverified; no live checks are a planning or implementation gate.

## 2. Response waiting and operational proof

**Decision**: Await each native response, including backend blocking-listener processing, under existing timeout/cancellation controls. Retain machine `accepted` terminology for a documented successful response. Human output may say assignment/unassignment confirmed, with no independent read-back claim. Reject enabled `--no-wait` and perform no extra polling.

**Rationale**: The native success response is evidence of the completed endpoint operation. This meets Constitution I without an extra read that might be eventually consistent or already reflect another actor's change.

**Alternatives considered**: Fire-and-forget would make acceptance unsupported; polling adds out-of-scope latency and new failure modes; interpreting timeout as rejection destroys uncertainty evidence.

## 3. Retry and submission evidence reuse

**Decision**: Extract a small private native user-task submission/evidence helper inside `internal/services/common`, retaining `CompleteUserTaskSubmission` as a compatibility wrapper with identical completion types, messages, and behavior. Add assignment-family wrappers for assign/unassign with their own domain result types. Limit extraction to response validation, retry evidence and failure classification; no generalized mutation framework or coordinator registry.

**Rationale/evidence**: `internal/services/common/usertask_completion.go` already retains evidence outside retry return values, validates 204, classifies errors through `httpc.HttpStatusErr`, and prevents a later rejection from erasing earlier ambiguity. `internal/services/retry.go` supplies the existing finite retry policy: 429, resource-exhausted 503, and temporary/timeout network errors. It does not automatically retry generic 504 or 409. Pass original transport errors into `RetryCamundaMutation`, not prematurely converted status errors, and preserve cancellation-during-backoff evidence.

Explicit rejection statuses follow the existing helper; generic 500/503/504, malformed responses, and transport failures remain uncertain. A transport error overrides a partial 204. Later 204 can establish acceptance after ambiguity. Do not add retries from endpoint prose or retry same-assignee conflicts as successes.

**Alternatives considered**: Calling the completion helper directly exposes completion-named results and diagnostics; copying the entire algorithm into every adapter risks divergence; replacing all mutation workflows with a framework exceeds scope.

## 4. Public facade and shared assignment execution

**Decision**: Expose `AssignUserTasks` and `UnassignUserTasks` in `c8volt/task`. Both delegate to one assignment-family coordinator in `internal/services/usertask/assignment.go`, with concrete action-specific wrappers. Public override configuration lives in `c8volt/foptions.WithUserTaskAssignmentOverride()` and maps to the matching internal call option, default false. Public results use assignment-specific types; no internal/generated types escape the facade.

The coordinator validates the whole input, snapshots keys and evaluated options, deduplicates, preinitializes ordered not-attempted outcomes, and uses `toolx.DetermineNoOfWorkers` and `toolx/pool.ExecuteSlice`. Zero workers is automatic for facade callers; negative is invalid. CLI explicit workers must be positive. Unassignment rejects an enabled assignment-override option, and both operations reject no-wait. Facade dry-run remains fully local and returns preview results.

Use the parent caller context for already submitted operations and a separate scheduling-stop signal, as completion does. Fail-fast must not discard in-flight evidence or cancel requests solely because another key failed. Results are merged by input index, not completion order. No prepare/execute API pair is needed because there is no backend planning lookup.

**Alternatives considered**: Single-key-first design duplicates later bulk work; CLI or facade worker pools violate layering; reuse of `PrepareUserTaskCompletions` adds unwanted reads and eligibility rules; a generic action API broadens the public surface unnecessarily.

## 5. Mode resolution and strict key parsing

**Decision**: Resolve new mode conflicts before the existing completion/variable dispatch. Assignment is active when `--assign` is present, even empty; empty/whitespace fails validation. Unassignment and completion are active by boolean value. Reject variable flags by presence, enabled no-wait by value, and `--allow-override` presence outside assignment even when explicitly false. In assignment, override uses its boolean value.

Extract only key collection and the pflag empty-occurrence tracker from `cmd/update_usertask_complete_input.go` into a focused shared user-task input helper. Keep completion variable decoding, completion checks, and legacy variable input behavior separate. Preserve flag-key order followed by stdin-line order, trimming/validation, empty-component rejection, scanner-error bounds, and stable deduplication. Nonblank assignees remain opaque, unmodified identifiers; only blankness is tested with trimmed whitespace.

**Rationale/evidence**: The completion parser already solves pflag's loss of empty `--key=` occurrences; `TestUpdateUserTaskCompletionInputKeyTrackingPreservesLegacy` protects variable-mode behavior. `validateUpdateUserTaskJSONConfirmation` in `cmd/update_usertask_variables.go` also preserves existing JSON/verbose rejection and JSON confirmation guardrails.

**Alternatives considered**: Cobra changed-flag mutual exclusion alone treats explicit false selectors as active; falling back from empty assignment to variables hides invalid intent; reusing the complete input parser would accept completion payloads and bind unrelated validations.

## 6. Local approval and command file ownership

**Decision**: Add focused assignment runner/input files and an assignment view file. The base command owns flags, metadata and dispatch only. Freeze local input before preview/approval; invoke the facade only after approval for normal execution. Use existing tenant-context, prompt and output helpers. No task reads, discovery, completion, variable writes, or unassigned-task warnings are allowed in either new mode.

**Rationale/evidence**: `cmd/update_usertask_complete.go` demonstrates runner/error ownership; `cmd/cmd_cli.go` handles real-terminal stdin, default-no approval and nonterminal behavior. Reuse `attachExplicitKeysTenantContext`, `collectExplicitAdminInputOptions`, `shouldImplicitlyConfirm`, and `confirmCmdOrAbortFn(cmd.ErrOrStderr(), ...)`. Preserve stdout-redirection behavior and output-mode guardrails.

**Alternatives considered**: Adding all declarations to the base command violates cohesion; reusing completion's preparation or fixture would require unintended reads; introducing a new prompt system risks stream and EOF regressions.

## 7. Result and diagnostic contract

**Decision**: Mirror completion's envelope/report organization, replacing completion-specific payload fields with `operation: assign|unassign` and assignment intent fields. For assignment, always include `assignee` and boolean `allowOverride`, including false; omit both for unassignment. Preserve all counts, zero included, initialized arrays, item status/submission identifiers and optional status/error evidence. See [CLI contract](contracts/cli-user-task-assignment.md).

Preview succeeds; successful execution uses envelope `accepted`; operation failures use the existing classified error envelope with partial payload once. Validation errors have no invented operation report. Render results first, emit one compact non-JSON aggregate ERROR when applicable, then exit using the original error class without another renderer. On a renderer failure, retain the original operation error when present and never retry envelope rendering.

Human summaries and verbose details use `renderHumanLine` with an attached configured logger; prompts use plain configured stderr. Keys-only prints preview keys or execution successes, with zero bytes when no item qualifies. Quiet is checked after machine-mode branches. Group bounded failure reasons, at most three groups plus an aggregate remainder; raw item errors stay machine evidence. Never inherit completion's special interpretation of failed/not-submitted as an already completed task.

**Alternatives considered**: Printing joined errors leaks HTTP details; direct stderr prints bypass configured logger level/destination/format; sharing completion report types forces irrelevant variables fields and wording into assignment.

## 8. Isolated validation and compatibility work

**Decision**: Add mutation-only fakes that reject every read and unrelated route. Reuse existing subprocess/terminal/log-record helpers, not the completion fixture that expects preflight reads. Extend all affected interfaces, adapters and test doubles together so each implementation unit compiles. Keep legacy doubles fail-closed on unexpected new operations.

Useful fixtures include `cmd/update_usertask_test.go`, `cmd/update_usertask_complete_terminal_test.go`, `cmd/update_usertask_complete_logging_test.go`, `cmd/update_usertask_complete_output_test.go`, and `testx` terminal helpers. Adapter and common submission tests must cover serialization of false, delayed 204, unsupported success statuses, transport ambiguity and retry evidence. Command tests must detect direct-print routing and raw aggregate regressions, not merely match substrings.

`Makefile` confirms `docs-content` runs `docsgen` and `test` runs the Go suite with race detection. Tests for this change must use only test-owned dependencies with live tags, profiles and inherited helper activation removed. See [quickstart](quickstart.md) for staged validation after implementation; no runtime test is required for these design documents.

**Alternatives considered**: A live Camunda target or smoke gate violates the implementation boundary; renderer-only tests miss logger setup and duplicate exit rendering; interface-only tasks create uncompilable intermediate states.

## Research resolution

All design unknowns are resolved from the repository and accepted feature contract. No new dependency, generated-client change, live target, or user clarification is required. The repository provenance limitations above remain explicit coverage limits, not blocked implementation gates.
