# Implementation Plan: Align Process Instance Variable Updates

**Branch**: `328-align-pi-variable-updates` | **Date**: 2026-10-11 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/328-align-pi-variable-updates/spec.md`; authoritative issue: [#328](https://github.com/grafvonb/c8volt/issues/328).

## Summary

Make `update pi` execute only the additions and changes in its prepared root-variable plan. Complete reads, classification, exact-scope submission, worker scheduling, and outcome verification belong in internal services. Add thin facade plan/execute delegation and adapt the resulting plans to existing CLI views. Retain the plan through operator approval, preserve partial results, and remove the extra read after successful waiting.

## Technical Context

**Language/Version**: Go 1.26, repository toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2, Viper 1.21.0, generated Camunda clients, `toolx/pool`, variable waiter and retry helpers. No new dependency.

**Storage**: Camunda root-scope variables; in-memory plans/results only, no new persistence.

**Testing**: Go testing, testify, existing HTTP fixtures and `testx.NewCmdTerminalRunner`; targeted tests first, full race suite after shared runtime changes.

**Target Platform**: Existing CLI platforms and Camunda 8.8, 8.9, and 8.10 adapters.

**Project Type**: Go CLI and public facade library.

**Performance Goals**: Zero writes for unchanged instances; payloads contain only changes/additions; no duplicate post-success verification; bounded existing worker concurrency. Full reads necessarily scale with all variable pages; no invented latency guarantee.

**Constraints**: Root scope only, all planning before writes, local writes, stable public JSON fields and CLI grammar, complete-read cancellation and malformed-page handling, tenant filtering, preserved UT behavior. No scope traversal, new flags, generated-client edits, or general workflow framework.

**Scale/Scope**: Single and bulk explicit-key updates, including more variables than one backend page, capped totals, sparse pages, and partial execution failures.

## Constitution Check

| Gate | Before research | After design |
| --- | --- | --- |
| Operational proof | Pass: default confirmation required; no-wait retains acceptance distinction | Pass: one service-owned waiter using complete root reads; acceptance retained on verification failure |
| CLI-first, script-safe | Pass: preserve grammar and output contract | Pass: existing field schema, mode precedence, empty payloads, stderr prompts; additive statuses explicitly documented |
| Proportionate validation | Pass: planning is documentation-only | Pass: lightweight artifact checks now; targeted tests and shared-consumer/full-race validation during implementation |
| Documentation matches behavior | Pass: command behavior change requires documentation | Pass: update README and source metadata/help, regenerate with `make docs-content` in implementation |
| Small repository-native changes | Pass: established UT and variable writer patterns exist | Pass: PI-specific service and narrow page traversal; no generic framework or new dependencies |
| Layering and cohesion | Pass: backend work assigned below facade | Pass: command only approves/renders; facade converts/delegates; version differences stay explicit |

No unjustified violations or unresolved scope questions. The constitution's proportional-validation rules govern this planning-only change; no runtime tests are required now.

## Project Structure

### Documentation (this feature)

```text
specs/328-align-pi-variable-updates/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    ├── cli.md
    └── service.md
```

`tasks.md` is the next phase and is not created by this command.

### Source Code (repository root)

```text
cmd/update_processinstance.go                     # flags, approval, facade dispatch
cmd/update_processinstance_variables.go           # mechanical preview adaptation
cmd/cmd_views_processinstance_update.go           # all final output and partial errors
c8volt/client.go                                  # dependency composition
c8volt/process/{api,client,model,convert}.go        # facade wiring and public models
c8volt/process/variable_updates.go                # proposed thin plan/execute methods
internal/domain/processinstance_variable.go       # plan/page/outcome domain additions
internal/services/processinstance/update.go       # proposed composed PI workflow
internal/services/processinstance/update_validation.go # proposed plan validation
internal/services/variable/api.go                 # page contract alongside existing list/writer
internal/services/variable/search/                # proposed complete-read traversal
internal/services/variable/{v88,v89,v810}/         # normalized single-page adapters
internal/services/variable/waiter/                # existing root-value verification
README.md
cmd/*_test.go, c8volt/process/*_test.go             # behavior and boundary checks
internal/services/{processinstance,variable}/     # service/adapter tests beside code
```

**Structure Decision**: Reuse existing package ownership. The new variable search subpackage avoids a parent-factory/adapter import cycle and shares traversal across three actual adapters. No new CLI lifecycle is introduced; if implementation adds a distinct mode with three or more mode declarations, apply the dedicated-mode-file rule before completion.

## Phase 0: Research

Completed in [research.md](research.md): current defects, complete-read strategy, scope/tenant preservation, constructor integration, status compatibility, command grammar, and validation approach are resolved from local code.

## Phase 1: Design

### Complete root-variable discovery

Add page request/result domain types carrying items, raw count, request, total certainty, continuation, and cursor. Extend variable adapters with single-page reads; retain explicit generated filters and raw value decoding. Share traversal in the variable search subpackage. Existing list methods delegate to it, so callers receive complete results without adding facade loops. Preserve stable ordering and validate metadata/progress. Capped totals are lower bounds, sparse retained pages do not terminate traversal, and non-progressing pages fail visibly.

### Frozen plan and execution

Compose a PI `VariableUpdateService` from the PI reader, exact variable reader/writer, configuration, and logger. Add plan/execute methods patterned after UT, with one destination per selected root. Planning deduplicates keys in stable order, validates values, reads selected instance/tenant evidence and all root variables, and records requested values plus only additions/changes as writable payloads. Any read or classification error yields no executable plan.

Execution validates and deep-copies the entire plan before the first write. Reject invalid root destinations, duplicate/conflicting targets, or mismatch between classifications and payloads. Use bounded existing pool execution and preserve fail-fast/no-worker-limit/options semantics. Each changed root uses `UpdateScopeVariables(local=true)`; unchanged roots receive no write. Return a result for each selected key in stable order, including skipped work after fail-fast/cancellation.

Use `WaitForProcessInstanceVariables` once per accepted changed root, passing the full frozen requested values and the complete reader; reject truncated observations in the waiter. No-wait skips all verification reads. Do not route prepared execution through the legacy writer that already waits or follow successful waiting with `ConfirmProcessInstanceVariables`.

### Facade and constructor compatibility

Add public plan models and plan/execute methods with mechanical deep-copy conversion, option mapping, and `ferrors.FromDomain`. Wire the composed service in `c8volt/client.go` into a new additive process-facade constructor, retaining existing constructors and old public update entry points. Old constructors remain valid for their existing operations; invoking new plan/execute methods without the new dependency returns a clear precondition error. Update affected interfaces, implementations, and test doubles together so each work unit compiles. Do not silently replan a prepared execution or change legacy entry points to a different selection contract.

### CLI and outcomes

Replace command-side backend planning with facade planning and map its result into existing preview structs. Preserve tenant evidence from the plan without rediscovery. Dry-run renders and returns; an all-unchanged execution renders its existing no-op payload before approval; otherwise approve the prepared scope and execute that same plan. Preserve missing-target validation and existing prompt eligibility.

Keep existing result fields and four existing status strings. Add `unchanged` and `skipped` values for mixed prepared results, document this compatibility extension, and update successful-no-op totals without implying acceptance. Preserve `submitted` for accepted/no-wait work. Render retained partial results in one failure envelope before following existing nonzero exit behavior, using the UT error-rendering pattern. Keys-only execution emits accepted mutation keys once each, including accepted work whose confirmation later fails, and excludes unchanged/skipped/rejected writes. Existing no-op and dry-run preview shapes stay unchanged.

### Documentation and validation

Update command help/metadata and README alongside behavior, then regenerate CLI documentation. [quickstart.md](quickstart.md) defines the future runnable validation sequence and the request/output matrix. No backend operation is executed during planning.

### Work-unit guidance for task generation

Group complete-page contracts, all three adapters, their list delegation, and required test doubles into one compiling unit. Group the prepared workflow's models, service, facade composition, tests, and conversions cohesively; split planning and execution only if each unit independently compiles and preserves usable behavior. Integrate CLI dispatch, rendering, real-terminal regressions, and docs in the same acceptance slice. Review adjacent units for consolidation; avoid separate fixture-only or context-reading tasks. Build collection behavior once rather than rebuilding a single-instance path. Respect Ralph's story scope when later assigning work units.

## Complexity Tracking

No constitutional exception is needed. New domain plan/page models carry real workflow facts; the PI service reuses established UT design patterns without importing UT-specific scope orchestration. The page traversal subpackage is justified by three version adapters and existing factory import direction.
