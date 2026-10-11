# Tasks: Clarify Variable Destination Scopes

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [preview contract](contracts/preview-output.md), and [quickstart.md](quickstart.md).

**Branch**: `codex/327-clarify-variable-scopes` · **Issue**: #327

**Organization**: Two implementation work units, one per command family. Each includes implementation, fixtures, regression tests, formatting, help, and generated documentation. US3's bulk and compatibility criteria are included in both units, not deferred to another task.

**Execution rules**: Follow `AGENTS.md`, the constitution, and `specs/ralph-implementation-rules.md`. Read context and update task/progress records within each work unit. Extend existing tests before changing behavior where practical, without separate failing-test commits. Use existing infrastructure; add no dependencies, service interfaces, generated-client changes, or command modes. Paths are repository-relative.

## User Story 1 — User-Task Destination Previews (P1, MVP)

**Goal**: Carry existing process identity to the view and identify root, intermediate, and local writes without changing execution or machine results.

**Independent test**: For one task with root/intermediate/local changes and a local addition, normal confirmation and dry-run show matching typed destinations. With several tasks, compact output lists stable unique inherited destinations and verbose output shows every planned write. Requests and mutation targets remain unchanged.

- [x] T001 [US1] Deliver complete UT destination previews: add and map owner metadata in `internal/domain/usertask_update.go`, `internal/services/usertask/update.go`, `c8volt/task/model.go`, and `c8volt/task/convert.go`; render detailed and compact scope labels in `cmd/cmd_views_usertask_update.go`; extend the service, facade, view, output, terminal, and error tests listed below; update `cmd/update_usertask.go` and `README.md`, regenerate `docs/cli/c8volt_update_user-task.md` and `docs/index.md`, and pass formatting and targeted validation.

**Metadata and its tests** — `internal/services/usertask/update_test.go` and `c8volt/task/update_test.go`:

- Copy the already fetched task identity in the existing plan builder; add `json:"-"` to the public field. Keep service and facade method signatures and adapter conversions unchanged. Add appropriate comments to touched declarations and tests.
- Required model constraint: “Preserve existing task identity, scope ownership, payload, complete-discovery, and mutation-target validation; introduce no new precondition for process identity.”
- Required serialization constraint: “No added field is serialized in public plan JSON. Marshaling an otherwise identical plan with and without this metadata produces identical output.”
- Required compatibility constraint: “JSON round trips can omit display metadata; executing such plans must retain existing behavior.”
- Required mutation invariant: “Local additions continue to target the task element. Existing effective variables continue to target their returned owner scope.” The new field must not select or redirect mutation targets.
- Extend `TestPlanUserTaskVariableUpdatesBuildsCompleteScopeSafePlan` and shared-target coverage to assert per-task owner identity, sparse-page completion, unchanged target grouping/counts/order, and unchanged reads/writes. Preserve map/slice isolation and struct-copy behavior.
- Extend `TestVariableUpdateFacadeDelegatesAndCopiesNestedValues` for domain → facade → domain metadata preservation. Compare serialization with populated versus absent owner metadata and exercise existing execution of plans without the field.
- Validate with `go test ./internal/services/usertask/... -run 'TestPlanUserTaskVariableUpdates|TestExecuteUserTaskVariableUpdates' -count=1` and `go test ./c8volt/task -run 'TestVariableUpdateFacade|TestLegacyTaskConstructor' -count=1`. Keep new cases within these families or expand the filters explicitly.

**Rendering and its tests** — `cmd/cmd_views_usertask_update_test.go`, `cmd/update_usertask_output_test.go`, `cmd/update_usertask_terminal_test.go`, and existing `cmd/update_usertask_error_test.go` coverage as needed:

- Implement detailed and bulk scope classification together using only plan metadata. Additions/changes use `(local, element:<key>)`, `(inherited, pi:<key>)`, or `(inherited, element:<key>)`; a pure helper in the existing view may serve both call sites. Do not infer BPMN types.
- Required uncertainty invariant: “Unknown metadata must not trigger discovery or fabricate scope kind. Existing failures remain failures.” For missing legacy owner identity retain `(inherited scope <key>)` and the existing bare inherited-summary key; known local destinations still use `element:<key>`.
- Required grouping invariant: “Preserve shared-target grouping, counts, association order, map/slice isolation, and all status values.” Keep bulk deduplication by raw scope key and first-seen order. Local-only bulk plans must not acquire an inherited-target line.
- Preserve summary wording, value rendering, separators, unchanged-requested and untouched formatting, result rendering, tenant context, mode precedence, and existing detail eligibility. The current scope helper also formats unchanged entries: do not accidentally apply the new planned-write syntax to them.
- Extend exact view tests for root/intermediate/local changes, local additions, duplicate inherited targets, same-name variables with different owners, legacy fallback, normal/dry-run parity, single/bulk, and verbose behavior. Assert no per-variable bulk detail without verbose.
- Extend existing command fixtures and `TestUpdateUserTaskOutputModes` to validate the metadata path and count task reads, variable reads, mutations, and all relevant requests; reject unexpected endpoints. Scope formatting must add zero requests in every preview mode.
- Cover supported JSON, keys-only, quiet, quiet plus machine output, auto-confirm, automation, and no-wait combinations. Decode one envelope and require EOF, preserve exact existing payloads and key lines, and assert zero bytes when no keys are emitted. Preserve selected-resource dry-run key behavior rather than treating an unchanged selected resource as an empty selection.
- Use `testx.NewCmdTerminalRunner` in `TestUpdateUserTaskTerminal` with separate stdout/stderr to prove annotated normal confirmation previews, configured/inherited stderr routing, accept/decline/default/EOF behavior, and prompt-free dry-run/no-op with actual terminal stdin. Auto-confirm alone cannot prove the normal preview path.
- Retain sparse discovery, explicit keys, no-op/empty payload, errors, partial acceptance, supported v88/v89/v810, and accepted-versus-confirmed regression coverage. No added activity or diagnostics should leak into output.
- Update only delivered UT behavior in help/README, explain inherited destinations and local additions, regenerate documentation, and inspect unrelated generator churn. Do not document PI annotations before T002.
- Validate with `go test ./cmd -run 'Test(UserTaskVariableUpdate|UpdateUserTask)' -count=1`, `make docs-content`, and `git diff --check`. If shared concurrent fixture capture changes, run the same command test selection with `-race`.

**Checkpoint**: US1 and its US3 obligations are complete in one independently useful work unit. Do not start US2 in the same Ralph iteration.

## User Story 2 — Process-Instance Destination Previews (P1)

**Goal**: Annotate every PI addition/change with the selected root key in existing detailed previews.

**Independent test**: Preview a changed value plus an addition for one PI through normal confirmation and dry-run; repeat with multiple PIs and verbose output. Each detail identifies the correct selected root, while ordinary bulk summaries and machine results remain unchanged.

- [x] T002 [US2] Annotate PI additions/changes in `cmd/cmd_views_processinstance_update.go`; extend `cmd/update_processinstance_test.go` and add focused real-terminal coverage in `cmd/update_processinstance_terminal_test.go`; update `cmd/update_processinstance.go` and `README.md`, regenerate `docs/cli/c8volt_update_process-instance.md` and `docs/index.md`, and pass formatting and targeted validation; schedule after T001 to avoid shared documentation and command-test churn.

**Implementation and acceptance**:

- Use `(pi:<ProcessInstanceKey>)` only on planned additions and changes. Preserve existing summary/count/status text, values, semicolon layout, unchanged/untouched entries, compact bulk behavior, and final result rendering. No PI service/model change or command-side planning expansion is needed.
- Extend single-resource and verbose multi-resource normal/dry-run assertions; ordinary bulk must stay compact. Confirm every detail uses its own selected key, including distinct roots with the same variable name.
- Retain JSON schema/content, keys-only behavior, quiet and machine precedence, current flag-combination validation, explicit-key aliases, and no-op/empty-payload/error paths. Decode a single envelope through EOF and capture streams separately. Preserve existing dry-run selected-key output and zero-byte results where no keys are emitted.
- Reuse `testx.NewCmdTerminalRunner` for actual terminal stdin with redirected stdout and configured/inherited stderr. Cover normal annotated previews with accept/decline/default/EOF, no-op/dry-run without prompts, and supported auto-confirm/automation/no-wait behavior. No new prompt helper or routing logic is needed.
- Assert discovery and total request counts as well as mutation counts; labels add zero requests, and dry-run/no-op/abort add zero mutations. Preserve accepted/confirmed/failure outcomes and existing version coverage.
- Update PI help/README examples and regenerate CLI docs in this work unit. Keep the UT documentation and regression assertions delivered in T001 intact.
- Validate with `go test ./cmd -run 'Test(UpdatePI|UpdateProcessInstance|FormatProcessInstanceVariableUpdatePlan|UserTaskVariableUpdate|UpdateUserTask)' -count=1`, `make docs-content`, and `git diff --check`. Keep new PI terminal cases under `TestUpdateProcessInstance...` so the filter selects them. Use focused `-race` testing if concurrent fixture capture changes.

**Checkpoint**: Both command families meet their independent preview criteria, with all US3 acceptance scenarios covered.

## User Story 3 — Bulk and Compatibility Acceptance (P2)

No separate implementation task: validate these criteria within the command-family work that changes them.

**Independent test**: Exercise normal/dry-run bulk, verbose, output-mode, terminal, and request-count scenarios in [quickstart.md](quickstart.md). Shared inherited destinations appear once in compact output, verbose details name every write destination, and machine results and backend counts remain stable.

| US3 acceptance scenario | Owning task |
| --- | --- |
| 1: Compact inherited destinations and stable deduplication/order | T001 |
| 2: Complete verbose details | T001, T002 |
| 3: JSON/keys/quiet compatibility | T001 includes serialization; both include command modes |
| 4: Confirmation, execution, errors, and statuses | T001 includes legacy plan execution; both include command/terminal regressions |
| 5: Zero additional backend requests | T001 includes metadata provenance; both assert request counts |
| 6: Help and documentation | Both, with generated docs in each work unit |

## Dependencies and Validation

```text
T001 [US1: metadata + all UT previews + tests + docs]
  → T002 [US2: all PI previews + tests + docs]
  → Full feature complete, including US3 acceptance coverage
```

T002 has no functional dependency on UT metadata; sequential scheduling avoids shared README/generated-doc and command-fixture conflicts. No `[P]` task or artificial parallel workstream is warranted. Within T001, service and facade test commands may run independently after metadata changes. US2 uses its existing test harness; US3 reuses each family's evidence. Do not add `t.Parallel()` to tests that mutate global flags or the command tree.

Run gofmt on touched Go files, add required declaration/test comments, inspect generated-doc churn, and record actual validation results and skips within each task. Mark completion only after meaningful checks pass. When committing through an authorized implementation workflow, use a Conventional Commit subject ending with ` #327`.

Runtime tests are unnecessary for this documentation-only task revision. During implementation, broaden to `make test` only for shared runtime/concurrency changes with broad impact or unresolved failures that warrant the full race-enabled suite; record the reason. Routine context reading, review, progress updates, and polish do not need standalone tasks.

## Requirement Coverage

| Requirement / success criteria | Tasks |
| --- | --- |
| FR-001–FR-004; SC-001 typed destination details | T001, T002 |
| FR-005; SC-002 compact bulk and preview parity | T001, T002 |
| FR-006; SC-004 metadata-only labeling, zero requests | T001, T002 |
| FR-007 preserved write scopes | T001, T002 |
| FR-008–FR-009; SC-003 machine/status/workflow compatibility | T001, T002 |
| FR-010; SC-005 help and docs consistency | T001, T002 |

## Consolidation Rationale

The former metadata task and UT-rendering task are merged: the metadata change is a small field propagation that exists solely to enable the preview, so keeping it separate adds a handoff without an independently useful operator outcome. Its model constraints, serialization/copying tests, legacy fallback, and service/facade checks remain explicit within T001. Single, bulk, and verbose UT behavior are delivered together.

PI rendering remains separate because it is independently useful, uses an existing root key, and belongs to another primary story; this also respects Ralph's one-story iteration boundary. Combining both command families would add cross-story scope without simplifying the implementation. US3 coverage stays with each affected renderer. Deliver T001 for the MVP and T002 for full completion; no requirement or meaningful test was removed.

## Phase 1: Convergence

Assessment found the scope behavior implemented but the promised command-level preview coverage incomplete. F1 is a MEDIUM partial validation gap, not a demonstrated runtime defect. Existing tasks and their completion markers remain unchanged.

- [x] T003 [US3] Complete command-level destination-preview regression coverage in `cmd/update_usertask_output_test.go`, `cmd/update_usertask_terminal_test.go`, `cmd/update_processinstance_test.go`, and `cmd/update_processinstance_terminal_test.go`, reusing existing fixtures and `testx.NewCmdTerminalRunner`, per T001–T002 validation obligations, US1/AC4, US2/AC2, US3/AC1–2 and AC5, and plan: command-path validation (partial).

**F1 evidence**: `TestUserTaskVariableUpdateHumanPlanContract` checks bulk/verbose labels using constructed plans; `TestUpdateUserTaskDestinationMetadataFlowsThroughCommand` covers the mixed-scope fixture only as a single-task dry-run; `TestUpdateUserTaskTerminal` asserts one intermediate inherited change. `TestUpdateProcessInstanceCommand_DryRunDestinationsAddNoRequests` covers verbose bulk dry-run, while `TestUpdateProcessInstanceTerminal` covers one selected root. Existing multi-task UT command tests exercise machine output or execution rather than the required human preview combinations.

**Acceptance and validation**:

- Exercise UT compact and verbose bulk previews through the actual command path for dry-run and normal confirmation, including a repeated inherited destination, process-root and intermediate inheritance, local changes/additions, and local-only bulk selection. Assert exact normalized summaries, typed inherited destinations in stable deduplicated order, absence of per-variable compact bulk details, and complete verbose details with the correct owner per task.
- Extend real-terminal UT coverage to prove the mixed root/intermediate/local changes and local addition appear in the normal single-task confirmation preview with the same labels as dry-run; retain separate stdout/stderr, configured/inherited stderr coverage, and no writes on decline.
- Exercise PI normal bulk confirmation with verbose details containing both additions and changes for distinct selected roots; compare labels with dry-run, and prove normal non-verbose bulk stays compact. Use real terminal stdin for confirmation and preserve prompt wording/counts and selected output streams.
- Count task reads where applicable, variable reads, mutations, and all relevant requests in these scenarios; reject unexpected endpoints. Prove labels add no lookups and dry-run/declined runs issue zero mutations. Reuse existing passing mode/error/status tests rather than duplicating the full matrix.
- Keep this work unit limited to the missing US3 compatibility/preview tests and necessary fixtures; do not refactor production behavior or regenerate unchanged docs. Add test comments, run gofmt, then `go test ./cmd -run 'Test(UpdatePI|UpdateProcessInstance|FormatProcessInstanceVariableUpdatePlan|UserTaskVariableUpdate|UpdateUserTask)' -count=1`; run the affected selection with `-race` when shared concurrent fixture capture changes. Record actual results and platform skips.
