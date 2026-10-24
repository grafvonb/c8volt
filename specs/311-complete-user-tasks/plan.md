# Implementation Plan: Complete User Tasks by Explicit Key

**Branch**: `311-complete-user-tasks` | **Date**: 2026-10-24 | **Spec**: [spec.md](spec.md)

**Input**: `specs/311-complete-user-tasks/spec.md`, GitHub issue #311.

## Summary

Extend `update user-task` with a dedicated `--complete` mode for one or more explicit keys. Validate and freeze the local request, preview locally or check tasks before confirming the eligible selection once, delegate through the thin facade, and execute native completion through the existing user-task services with bounded concurrency. Return ordered accepted/failure/unattempted evidence without task-state polling. Variables are optional completion data, never a scope-variable plan. Preserve the existing variable-update path, aliases, schemas, and verification semantics.

Use existing generated clients for Camunda 8.8, 8.9, and 8.10, shared mutation retry policy, and transport DEBUG instrumentation. Add no dependency, discovery mode, assignment, generic mutation framework, or generated-client changes.

## Technical Context

**Language/Version**: Go 1.26; toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2/pflag 1.0.10, public facade options/errors, generated Camunda clients, `toolx/pool`, `toolx.CopyJSONMap`, mutation retries and configured slog logger. No new dependencies.

**Storage**: In-memory frozen inputs and ordered completion outcomes; backend owns task state. No local persistence or migration.

**Testing**: Go/testify, generated-client stubs, test-owned HTTP servers, subprocess and real-terminal tests. Documentation validation in this planning phase; focused runtime checks and a final race suite during implementation.

**Target Platform**: Existing Go CLI platforms; real-terminal tests use the repository's supported-platform runner and explicitly report unsupported coverage.

**Project Type**: Existing CLI plus public Go facade library.

**Performance Goals**: One native lookup attempt per unique key before confirmation, then one logical native completion per eligible key; existing finite retries may produce additional HTTP attempts. Zero variable reads, scope writes, assignment calls, repeated task reads, or polling. Preflight alone opts out of shared transport read retries. Automatic workers follow existing `min(count, 2*GOMAXPROCS, 32)` policy; explicit bounds and no-worker-limit precedence remain intact. Memory grows with unique keys and one immutable variable snapshot, plus ordered evidence. No new latency benchmark.

**Constraints**: Explicit-key backend authorization; backend-confirmed completion without independent read-back; complete local validation before submission; no empty-input no-op; reject completion with no-wait; fail-fast stops scheduling without canceling already submitted work; configured logging; one JSON envelope; no live implementation validation.

**Scale/Scope**: Single and multiple explicit tasks through the same workflow, same optional variables for every key, three supported adapters. No search, assignment/unassignment (#351), task-attribute updates, per-task payloads, downstream waiting, or changes to job completion.

## Constitution Check

| Principle / gate | Before research | After design |
|---|---|---|
| Operational proof | Pass: completion relies on the native successful response | Pass: documented HTTP 204 confirms backend completion; human output distinguishes this from additional polling, machine identifiers remain accepted, and unknown submission remains uncertain |
| CLI-first, script-safe | Pass: extend existing command and aliases | Pass: mode-local validation, stderr prompts, stdout machine results and logged human summaries, stable completion payload, one envelope, unchanged variable mode |
| Proportionate validation | Pass: documents need lightweight checks now | Pass: targeted package/command tests first; final race suite justified by cross-package interfaces, adapters, and concurrent mutation execution |
| Documentation matches behavior | Pass: both modes must be discoverable | Pass: source metadata, parent help, README and generated docs are part of implementation delivery |
| Small compatible changes | Pass: existing packages and dependencies | Pass: one collection facade and area coordinator; native adapters; no framework, constructor hierarchy, or generated changes |
| Layering and cohesion | Pass: backend mechanics belong in services | Pass: completion mode file owns CLI lifecycle; views own formatting; service owns deduplication, snapshot, scheduling, retries and results |
| Implementation test boundary | Pass: isolated dependencies only | Pass: fake-only guide, sanitized environment, real-terminal harness, no environment-backed gate |

No gate exceptions. The successful native response confirms backend completion, including blocking task listeners. The existing accepted machine contract remains unchanged and does not claim independent read-back or downstream completion. Existing logger-backed variable-mode human output is preserved for compatibility; its raw-error rendering and shared parsing gaps are not copied into completion. Post-design review found no conflict with the [constitution](../../.specify/memory/constitution.md), [repository guidance](../../AGENTS.md), or [Ralph rules](../ralph-implementation-rules.md).

## Project Structure

### Documentation (this feature)

```text
specs/311-complete-user-tasks/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── cli.md
│   └── facade-service.md
└── checklists/requirements.md
```

The next workflow creates `tasks.md`; this phase does not create implementation tasks or start Ralph.

### Source Code (repository root)

```text
cmd/
  update.go                                # parent descriptions/examples
  update_usertask.go                        # mode flag, dispatch, shared metadata
  update_usertask_complete.go               # new mode input, confirmation, execution dispatch
  cmd_views_usertask_complete.go            # new preview/result and safe failure formatting
  update_usertask_complete*_test.go         # execution, output, diagnostic and terminal cases
  command_contract_test.go                  # capability metadata regression
c8volt/task/
  api.go                                   # collection completion method
  completion.go                            # new thin delegate
  model.go                                 # public completion models
  convert.go                               # mechanical copying/mapping
  completion_test.go                       # mappings, options, partial errors
internal/domain/
  usertask_completion.go                   # new completion facts and statuses
internal/services/usertask/
  api.go                                   # native scalar completion contract
  completion.go                            # new shared collection coordinator
  completion_test.go                       # validation, ordering, failure and worker tests
  v88/{contract.go,completion.go}           # generated interface + native completion
  v89/{contract.go,completion.go}
  v810/{contract.go,completion.go}
  v*/completion_test.go                    # version mapping and errors
README.md
docs/cli/                                  # regenerated during implementation
```

New filenames above describe intended additions; existing files and all affected test doubles must be updated together when interfaces change. Inventory stubs across the repository, not just this area. Factories, constructor signatures, existing variable workflows, and generated clients remain intact.

**Structure Decision**: Reuse the current `task` facade and `usertask` area. A shared collection function delegates scalar submissions to version adapters, following existing collection-read ownership. A locally validated request is enough for preview. For execution, the service prepares an opaque in-memory plan containing frozen inputs, ordered preflight evidence and advisory counts. The CLI uses those counts for warnings and confirmation, then executes that same plan without rereading. No persistence, new service dependency or state machine is introduced.

## Design

### Mode selection and local input

Dispatch completion before the required-variable parser and variable workflow. Preserve `validateUpdateUserTaskJSONConfirmation` and existing bootstrap/error helpers. Completion validates all flag/stdin keys, optional variable input, worker values, and complete/no-wait incompatibility before any native mutation. The completion parser applies the existing 16-digit user-task stdin grammar to all sources, permits valid flag keys with empty stdin, and rejects an empty final selection. It must not silently reuse the shared helper that validates only stdin.

Freeze ordered unique keys and decode the completion payload with `json.Decoder.UseNumber` before preview/confirmation, requiring one non-null JSON object followed by EOF. Reuse source selection only; retain the existing variable-update decoder. The newly decoded object is already owned by the CLI and needs no additional JSON copy. Preview is a local description, not a claim of task existence or eligibility. Attach existing explicit-key tenant context from configuration, with no filtering or fabricated resolved tenant evidence. After service preparation, warn through the configured logger and confirm the eligible count once through `cmd.ErrOrStderr()` only when nonzero; honor auto-confirm, automation, EOF, abort, and output guardrails.

### Facade and service ownership

Add `task.API.CompleteUserTasks(ctx, keys, variables, wantedWorkers, opts...)`. The facade maps options and delegates input directly to `usertask.CompleteUserTasks`, which owns validation and snapshotting; it maps every returned item even on failure, and normalizes the aggregate error through `ferrors.FromDomain`. The thin `PrepareUserTaskCompletions` and `ExecuteUserTaskCompletions` methods expose preparation and execution around CLI approval; `CompleteUserTasks` remains a convenience composition for direct callers. No facade workers or retries.

The coordinator rejects invalid/empty input and no-wait, snapshots values, applies existing options once at the owning layer, and runs unique keys using `DetermineNoOfWorkers` and `pool.ExecuteSlice`. `DryRun` is safe for direct callers and returns preview outcomes without backend calls. Preparation reads each key once through `GetNativeUserTask`, using `httpc.WithoutReadRetry` only for that read. Completed observations become failed/not_submitted conflicts; missing assignees and unchecked observations become advisory counts. Read errors never imply completion or unassignment; caller cancellation aborts preparation. Execution reuses the frozen plan, skips completed tasks, and submits nothing if fail-fast encounters any preflight conflict. Track started state and callback errors per index. Pool cancellation controls scheduling; original caller context controls already-started submissions. Join real failures and caller cancellation without substituting fail-fast's synthetic cancellation for the real error.

Each version adapter calls generated `CompleteUserTaskWithResponse`, checks response validity and documented 204 successful completion, and uses existing HTTP/domain error conversion. No `RequirePayload` on a no-content success. Nil variables encode nullable `null`; supplied empty objects remain `{}`. No task reads, variable reads/writes, assignments, legacy fallback, or state reconciliation.

### Retry and uncertain submission

Use the existing finite `RetryCamundaMutation` policy. Keep last response/status and whether an earlier attempt had an ambiguous outcome even when cancellation during backoff makes the helper return a zero response. A final accepted response wins; timeout, cancellation after submission, 504, malformed response, or transport ambiguity must not imply definite rejection. Earlier ambiguity survives a later rejection. Retry progress uses only bounded verbose messages with a constant operation label; shared DEBUG transport diagnostics remain independent of verbose.

### Output, errors, and exit

See [CLI contract](contracts/cli.md) for the stable completion payload and mode matrix. The completion renderer retains JSON and keys-only results on stdout and routes all human previews, summaries and verbose rows through the existing configured INFO helper `renderHumanLine`. Human mode leaves stdout empty and honors logger format, destination, level and quiet suppression. Human success for one key says `updated user-task <key>: completion confirmed`. Batch and failure summaries use `update user-task`, label successful responses `completed`, and omit zero counters. This confirmation comes from the native response, without additional reads or polling. Diagnostics lead with grouped reasons rather than repeating result counters. Completed-task conflicts use singular/plural state-and-consequence wording, mentioning unapplied variables only when report.VariablesProvided is true. Plans and previews use `variables: submit` only when a source was supplied, including `{}`. Verbose may add safe per-task facts. This clarification changes no execution behavior, machine fields, statuses, or exit codes. Keys-only execution emits accepted keys only. Success uses accepted-envelope rendering explicitly; preview uses succeeded with a preview phase.

Trace failure as: adapter structured evidence and error → coordinator ordered outcomes plus aggregate error → facade preserves outcomes and converts error → renderer logs human results at INFO or writes machine results/one JSON envelope → non-JSON configured logger writes one bounded aggregate ERROR → exit using the original converted error. Validation uses existing pre-execution helpers. Never call a second error renderer after an execution envelope. Do not format raw `Error` fields in human or verbose rows. Known status/count facts drive grouped diagnostics, at most three named reasons plus an additional-failure count. Review every new print and raw error formatting site at this boundary.

### Compatibility and documentation

Do not alter variable-update required sources, planning, scopes, shared-target behavior, unchanged-data no-ops, verification, no-wait, schemas, or existing read commands. Reuse common parsers only without changing their existing callers. Update test flag reset/registration helpers for the new flag so cases cannot leak state.

Broaden worker/dry-run/fail-fast descriptions to explain both modes. Update `cmd/update.go`, command examples, capability metadata, and README. Regenerate CLI documentation with `make docs-content` only after implementation changes the command source. No retired-version compatibility suite or default-version change.

## Validation and Delivery Boundaries

The [quickstart](quickstart.md) defines runnable isolated validation and the acceptance matrix. Shared HTTP classification, retry and cancellation tests live once beside `common.CompleteUserTaskSubmission`; version adapters retain generated-wire, ownership, logging-option and response-delegation checks. Required coverage includes three adapters, public mappings, collection concurrency/order/cancellation, complete local validation, no remote preview, repeated lookup, or polling paths, all output/automation combinations, real-terminal confirmation, one-envelope EOF, logger framing/severity/destinations, repeated backend failures, DEBUG redaction and original exit codes. Mutation-style checks must demonstrate that replacing logging with a direct print or rendering raw joined errors causes failure.

Build single and collection execution once. For task generation, keep native interface changes, all adapters, and required stubs compiling in one cohesive unit; keep coordinator behavior with its tests; keep CLI behavior and output/error/terminal checks with the relevant implementation; keep help/source metadata with regenerated docs. Tightly coupled portions may be combined when practical, but do not create separate “single task first” and “bulk later” rebuilds. Do not split tests, formatting, review, or reading into stand-alone implementation tasks. Cross-story acceptance coverage does not authorize a Ralph iteration to start unrelated stories.

Planning validation is document consistency, paths, links, identifiers, and whitespace only. Runtime coverage remains planned, not passed. Real-engine listener, assignment eligibility, and eventual state behavior remain outside isolated implementation proof.

## Complexity Tracking

No constitutional violations or new framework. Separate execution status and submission evidence are necessary to distinguish timeout/cancellation from backend rejection. The adapter's small retry-evidence accumulator and coordinator's started slots are required to avoid losing partial results; reuse existing helpers around them rather than adding a general orchestration layer.

### Update command rhythm (FR-021)

Reuse `renderAttachedTenantContext` and the configured INFO logger for known selection scope and the approval plan, then WARN diagnostics, a plain stderr prompt, and the INFO result. Single-key plans/prompts name the key; bulk plans use existing ready counts plus a completed count captured by the same preflight reads. CompletedCount is advisory, never added to JSON. Mixed fail-fast plans explain zero submissions; all-completed selections go directly to conflict results. Auto-confirm/automation skip the approval plan; quiet and machine modes suppress INFO. Local dry-run shows known scope and an intent preview with zero reads. Preserve all scheduling, request counts and machine contracts.

## Scalar job outcome follow-up (FR-022)

Extend `validateUpdateJobPlanPreconditions` with one shared COMPLETED guard for the three worker outcomes. Reuse `plan.Current` from `planUpdateJob`; keep read/retry policy and service/facade APIs unchanged. Format bounded human messages in `cmd/cmd_views_job_logging.go` and pass them as human-only overrides to `handleCommandError`. Preserve original backend errors for JSON and exit resolution. Do not add eligibility allowlists or route scalar operations through the bulk lifecycle. Update source help/README, regenerate docs, and validate command execution with isolated HTTP/terminal fixtures plus the affected command/docs race suite.
