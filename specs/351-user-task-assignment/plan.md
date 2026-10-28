# Implementation Plan: User-Task Assignment and Unassignment

**Branch**: `351-user-task-assignment` | **Date**: 2026-10-28 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/351-user-task-assignment/spec.md`

## Summary

Extend `update user-task` with `--assign <assignee>`, `--unassign`, and assignment-only `--allow-override`. Support single and collection explicit keys through the same validated, ordered bulk workflow. Protect existing assignees by explicitly submitting false for override unless enabled. Await each documented native response, preserve uncertain and partial outcomes, and perform no preflight task reads or additional state polling.

Use thin `c8volt/task` methods, a shared assignment-family coordinator in `internal/services/usertask`, and concrete mappings in v88/v89/v810 adapters. Reuse completion's key handling and narrowly extracted native submission-evidence mechanics while preserving its existing preflight, public types, messages and output. Retain configured logger routing, one JSON envelope, success-only execution keys, and all existing variable-update behavior. Research decisions are recorded in [research.md](research.md).

## Technical Context

**Language/Version**: Go 1.26.0; repository toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2, pflag 1.0.10, Viper 1.21.0, standard `context`/`net/http`/`log/slog`, generated Camunda clients, `toolx/pool`, `services.RetryCamundaMutation`, and facade/error helpers. No new dependency.

**Storage**: No persistence or migration. Frozen inputs and ordered outcomes live in invocation memory; Camunda remains authoritative for assignment state.

**Testing**: Go tests and testify, generated-client mocks, test-owned HTTP servers, existing isolated command subprocess and real-terminal helpers. Documentation-only validation for this planning change.

**Target Platform**: Existing CLI/library platforms; Camunda 8.8, 8.9, and 8.10 adapters. Terminal-specific tests retain current supported-platform behavior and report any skipped coverage accurately.

**Project Type**: Go CLI with public facade library and internal versioned service adapters.

**Performance Goals**: O(N) selection/result bookkeeping for N unique keys; one logical operation per key except fail-fast/cancellation, bounded concurrent submissions, no extra discovery/read/poll requests. Native listener processing and configured retry/timeout bounds determine response duration; no invented latency SLA.

**Constraints**: Explicit false override, no implicit identity or completion, unchanged completion/variables contracts, deterministic ordered outcomes, uncertainty retained across retries, configured logging and exact machine stream separation, no generated-client edits or live-environment checks.

**Scale/Scope**: Two closely related modes for single/multiple explicit keys, three adapters, one shared assignment coordinator, narrow input and response-evidence extraction, command/public contracts, and documentation. No search, assignment-state waiter, generic mutation framework, or new transport policy.

## Constitution Check

*Gate reviewed before research and re-evaluated after design.*

| Principle | Before research | Post-design evidence |
| --- | --- | --- |
| I. Operational proof over intent | PASS: success must require a documented native response | PASS: native 204 confirms the lifecycle operation including blocking listeners; no independent read-back claim; timeout/cancellation retain unknown submission |
| II. CLI-first, script-safe interfaces | PASS: additive modes on the existing command | PASS: aliases, output precedence, confirmation guards and existing exits preserved; one-envelope and keys-only contracts defined |
| III. Proportionate validation | PASS: planning uses document checks only | PASS: staged isolated tests, bounded race checks for concurrency, full suite justified once by shared helper/option/interface changes; no live gate |
| IV. Documentation matches behavior | PASS: command behavior requires documentation updates | PASS: same implementation unit updates command/parent metadata, README and generated docs through `make docs-content` |
| V. Small, compatible, repository-native changes | PASS: existing helpers and ownership retained | PASS: concrete two-mode coordinator and private evidence helper only; no broad completion redesign or generalized framework |

No unresolved constitution or architecture exception. Backend mechanics remain in internal services; facade methods only map/delegate/convert. Project implementation boundaries apply to every validation command. No feature requirement conflicts with the reviewed Ralph layering and work-unit rules.

## Project Structure

### Documentation (this feature)

```text
specs/351-user-task-assignment/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    ├── cli-user-task-assignment.md
    └── facade-user-task-assignment.md
```

`tasks.md` is the next-phase output and is not generated by this planning command.

### Source Code (repository root)

```text
cmd/
├── update_usertask.go                       # flags, mode selection, dispatch, help
├── update_usertask_assignment.go            # new assign/unassign approval and runner
├── update_usertask_assignment_input.go      # new frozen intent and validation
├── update_usertask_input.go                 # extracted strict key handling only
├── update_usertask_complete_input.go        # retain completion-specific checks/payload
├── cmd_views_usertask_assignment.go         # new report and final formatting
├── update.go                               # parent command metadata
└── *_test.go                               # execution, logger, terminal, regressions
c8volt/
├── task/{api.go,assignment.go,model.go,convert.go}
└── foptions/{options.go,options_test.go}
internal/
├── domain/usertask_assignment.go            # new neutral assignment-family outcomes
└── services/
    ├── calloption.go                       # explicit assignment-override option
    ├── common/usertask_submission.go        # narrow private native evidence mechanics
    ├── common/usertask_completion.go        # unchanged external completion wrapper
    ├── common/usertask_assignment.go        # new assignment-family wrappers
    └── usertask/
        ├── api.go
        ├── assignment.go                   # validation, snapshot, ordered bulk execution
        └── {v88,v89,v810}/{contract.go,assignment.go,*_test.go}
testx/                                     # existing shared test infrastructure
README.md
docs/cli/                                  # generated from command metadata only
```

**Structure Decision**: Extend the existing owners. New source paths above are planned outputs, not files created during planning. Assignment and unassignment share one cohesive mode file/family; the base command does not acquire backend lifecycle mechanics. Generated clients and `toolx/pool` remain unchanged.

## Phase 0 — Research Decisions

Completed in [research.md](research.md): supported-version routes and override serialization, native success/listener semantics and provenance limits, existing retry policy, local input/approval, facade option mapping, shared assignment scheduling, machine schema, diagnostics and fixtures. No unresolved design clarification remains.

## Phase 1 — Design

### Input, approval and dispatch

Resolve active modes before completion/variable dispatch: assignment by flag presence, unassignment/completion by true value. Reject conflicting modes; reject assignment-only override flag presence outside assignment, even false. Reject variable flag presence in either new mode, and true no-wait. Preserve false boolean semantics, existing JSON confirmation guards and JSON-plus-verbose rejection.

Extract strict key parsing without changing legacy variable input behavior. Flag keys precede stdin lines under the existing contract. Freeze copied keys, opaque nonblank assignee, operation and boolean override before any plan/confirmation output. Preview is local with no facade or backend call; normal execution approves once then invokes the matching facade method. No completion preparation call or task lookup.

Use known explicit-key tenant context without discovery. Plans and previews honor quiet/machine visibility and configured INFO logging. Confirmation names the action, selected scope, assignee and replacement policy; plain prompts use `cmd.ErrOrStderr()`, preserving terminal eligibility, nonterminal bypass, decline, EOF and redirected stdout behavior.

### Public and internal service boundaries

Add two explicit public operations, defined in [facade contract](contracts/facade-user-task-assignment.md), returning `UserTaskAssignmentResults`. Add `foptions.WithUserTaskAssignmentOverride()` and its narrowly named internal mapping. The value defaults false and is neither derived from auto-confirm/automation nor from generic force flags. No public prepare/execute pair is needed.

The facade delegates to assignment/unassignment wrappers around one internal coordinator. It maps inputs/options, converts all result items even on failure, and normalizes the original error with `ferrors.FromDomain`. The coordinator validates and snapshots inputs/options before scheduling, rejects no-wait and inappropriate override, handles public dry-run locally, preinitializes one item per unique key and merges per-index results. No backend reads.

All adapter interfaces and affected doubles change in the same compiling unit. Existing read/completion doubles fail on unexpected assignment; new assignment fixtures fail on any read, completion or variable mutation. No generated types escape adapters.

### Native submission and scheduling

Each adapter maps assignment to explicit assignee/override pointers and unassignment to the bodyless native deletion. Both require documented 204 acceptance. Common code retains last response metadata, transport-error precedence and uncertainty across retries. Private helper extraction keeps existing completion messages/types/classification unchanged and is protected by existing completion tests. Existing throttling/temporary-network retry behavior is preserved, including no new retry of generic 504 or 409.

Use `DetermineNoOfWorkers` and `pool.ExecuteSlice`. One worker is sequential; automatic selection and no-worker-limit keep current precedence. Separate scheduling stop from caller cancellation, retaining already in-flight evidence. Join errors after ordered results are merged, preserve the caller error where appropriate, and never fabricate acceptance for missing outcomes. [Data model](data-model.md) defines execution and submission transitions.

### Results, logging and exit

The new CLI report retains the completion-style phase/keys/items/counts shape, with operation `assign` or `unassign`. Assignment includes assignee and explicit boolean override; unassignment omits those two fields. Counts include zeros; item error/status-code omission follows existing conventions. The shared envelope schema is unchanged.

Trace the failure path: adapter/common error and evidence → ordered service results plus joined error → facade preserves results and converts error → runner creates report → renderer emits one error envelope with payload, or selected-format results → non-JSON runner emits one bounded aggregate ERROR → `exitAfterRenderedResult` preserves the original class/exit. Never send a raw joined/transport error to normal human output. If rendering fails, report a compact logger error without invoking a second renderer; retain the operation error for exit precedence when present.

JSON and keys-only go to stdout. Human preview, result and verbose lines use `renderHumanLine` only with the configured logger attached. Prompts remain plain stderr. Group repeated failure reasons, cap distinct groups, preserve unknown evidence, and keep raw machine error evidence out of human text. DEBUG uses shared diagnostics and redaction independent of verbose.

### Compatibility and documentation

Preserve existing completion preflight, warnings, optional variables, result schema and wording; reuse only strict keys and private evidence mechanics. Preserve legacy variable updates, reads, task resolution, explicit-key tenant handling and `get pi --has-user-tasks`. Update command and parent help, README and examples in the command delivery unit; regenerate CLI docs with `make docs-content`, never by hand.

## Validation Strategy

The [quickstart](quickstart.md) is an isolated validation guide for implementation, not a request to run runtime tests during planning. It covers:

- All three adapters: real generated request serialization against test-owned HTTP fakes, explicit false/true override, identity preservation, correct paths and no body for unassignment, documented and malformed responses, zero extra routes.
- Shared submission: temporary/timeout error, 429/resource-exhausted 503, generic 503/504, ambiguous-then-rejected, later success, partial 204 with call error, cancellation/backoff and error classification; completion regression after extraction.
- Coordinator/facade: validate all input before mutation, stable deduplication/snapshotting, sequential/bounded execution, out-of-order completion, fail-fast in-flight evidence, cancellation before/after submission, ordered partial results, option mapping and no reads.
- Commands: all modes/aliases/input forms, false/present flags, no-wait/payload rejection, dry-run zero requests, strict JSON plus EOF, partial failures/exits, keys-only empty-byte cases, quiet+machine, auto-confirm/automation and preserved existing behavior.
- Diagnostics: success/dry-run/verbose INFO routing and suppression, single bounded ERROR, configured/inherited destinations, plain and structured handlers, both DEBUG entry points and redaction; tests fail if direct-print or raw-aggregate regressions are introduced.
- Terminal: real terminal stdin with separately captured stdout/stderr, configured/inherited destinations, redirect eligibility, approve/decline/EOF and dry-run/no-request cases.

Start with package-specific targeted tests; add race checks for new scheduling and changed common evidence. After these pass, run the repository race-enabled suite once because shared call-option mapping, public/internal interfaces and completion evidence extraction span packages. Do not repeat it for subsequent prose-only changes. Remove inherited live-test selectors and helper activation before every run; prohibit integration targets/build tags and real Camunda endpoints. Any terminal/platform limitation or unverified engine behavior must be reported rather than claimed passed.

## Work-Unit Guidance for Task Generation

Use approximately four cohesive outcomes, refined against implementation size rather than a fixed task quota:

1. Native assignment-family outcomes/options and all three adapter operations with narrow common-evidence extraction, interface/double updates and focused tests. Include completion regression validation here.
2. Shared bulk assignment/unassignment coordinator and thin facade APIs, options/partial-result conversion, all affected doubles and validation/race tests. Build single and collection behavior together.
3. CLI mode resolution, frozen input, approval, reporting, logger/terminal execution coverage and command/README/generated documentation in one usable command delivery unit. Split only if a practical iteration requires it and each result remains compiling and independently verifiable; do not expose untested output paths as complete.
4. Cross-layer compatibility and final acceptance evidence only where not already covered by earlier units; avoid a duplicate test-only task or routine review/commit tasks. Consolidate this into preceding units if no distinct work remains.

User stories define acceptance coverage rather than forcing duplicate assignment/unassignment or single/bulk implementations. Interface changes and their implementations/test doubles must not be separated into broken intermediate tasks. No hypothetical parallel-task splitting, live-target designation, or cleanup gate.

## Complexity Tracking

No constitution violations require an exception. The private native user-task evidence helper is limited to three concrete task operations and retains existing completion wrappers. Separate assignment-family types prevent completion preflight semantics from leaking into new behavior; this is narrower than a generic mutation framework.
