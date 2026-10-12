# Tasks: Align Process Instance Variable Updates

**Input**: Design documents in `specs/328-align-pi-variable-updates/`
**Branch**: `328-align-pi-variable-updates`
**Created**: 2026-10-12
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [service contract](contracts/service.md), [CLI contract](contracts/cli.md), [quickstart.md](quickstart.md).

**Tests**: Required by the specification. Write regression/contract cases before the corresponding behavior within each work unit, then implement, format, and run its targeted checks. Tests, fixtures, interface implementations, and required test doubles belong to the same compiling unit; no separate failing-test commits.

**Organization**: Seven sequential work units, grouped by story. Each includes implementation, tests, and validation. Story labels identify primary acceptance ownership; required safety behavior must be implemented with the operation that needs it.

## Format and work-unit responsibilities

Every task uses `- [ ] Tnnn [USn] description` for story work or omits the story label for shared foundation. Paths are repository-relative; proposed files are identified as new. No `[P]` markers are used because these units have real dependencies or shared files.

Before each unit, read its feature context and `specs/ralph-implementation-rules.md`, preserve existing changes, inspect nearby code, and apply repository layering and comment rules. Run `gofmt` on touched Go files. Record actual checks and limitations in `specs/328-align-pi-variable-updates/progress.md` when implementing; these responsibilities are not standalone tasks. Any issue-associated commit must have a Conventional Commit subject ending in ` #328`. No commit or implementation is performed by task generation.

## Phase 1: Complete variable reads

**Goal**: Every existing root-variable list consumer receives complete, correctly scoped results across all supported versions.

- [ ] T001 Implement complete root-variable discovery in `internal/domain/processinstance_variable.go`, `internal/services/variable/api.go`, new `internal/services/variable/search/search.go`, and `internal/services/variable/v88/variables.go`, `internal/services/variable/v89/variables.go`, `internal/services/variable/v810/variables.go`, with metadata decoding in each version's `convert.go`. Add single-page reads and have existing list methods delegate to the shared traversal without a factory import cycle; update affected implementations/test doubles in the same unit. Enforce the data-model constraints: Request is “Positive bounded size, nonnegative offset or advancing cursor; checked arithmetic”; Items are “Existing `ProcessInstanceVariable` values, preserving raw value and `APITruncated`”; RawCount is “Backend item count before local filtering; nonnegative”; total is “Optional exact count or lower bound; absence is not zero”; continuation is “Backend progress evidence; contradictory or repeated non-progressing metadata is an error”. Preserve root/instance/tenant filters, stable ordering, `truncateValues=false`, raw values, and both truncation field spellings on every page. Prefer advancing cursors, otherwise advance offsets by raw count; retain lower bounds, validate exact totals, continue sparse/capped pages, and fail on impossible progress, overflow, cancellation, malformed metadata, or later-page errors instead of returning partial success. Add new `internal/services/variable/search/search_test.go` and extend each version's `service_test.go` with paging, filter, raw-decoding, and error fixtures; cover existing list/enrichment consumers in `internal/services/processinstance/enrichment_test.go`. Run `go test ./internal/services/variable/... -count=1` and `go test ./internal/services/processinstance/... -run 'Variable|Enrich' -count=1`; document complete-read behavior in relevant API comments and README if existing user-visible variable discovery wording changes.

**Checkpoint**: Complete-list behavior is independently usable. All adapter/interface changes compile together. This is not permission to start writes before whole-collection planning.

## Phase 2: User Story 1 — Trust the complete preview (Priority: P1)

**Goal**: Produce a complete, root-only, truncated-safe preview and fail before writes if any selected instance cannot be planned.

**Independent test**: Preview several instances with variables on later/sparse pages, nested duplicate names, and truncated values. Check all four classifications and verify zero mutations, including a failure on the final instance.

- [ ] T002 [US1] Implement collection planning in new `internal/services/processinstance/update.go` and plan models in `internal/domain/processinstance_variable.go`, with thin public planning methods/models/conversions in `c8volt/process/api.go`, `c8volt/process/model.go`, `c8volt/process/convert.go`, and new `c8volt/process/variable_updates.go`; wire the composed dependency through `c8volt/process/client.go` and `c8volt/client.go` using an additive constructor. Preserve old constructors/entry points and return a precondition error for new methods without their dependency; add only the implemented planning method to interfaces now. Enforce these model constraints verbatim: RequestedKeys are “Unique selected keys in stable input order”; RequestedValues are “Deep-copied JSON-compatible name/value input, immutable during use”; Instances are “One per selected key, each with the instance/root destination and classifications”; tenant evidence is “Facts read during preparation, sufficient for existing approval/envelope behavior”; counts are “Selected/changed instance counts and classification counts, computed from the same plan”. Root identifiers “Both identify the same selected root; neither may be blank”. Additions are “Requested names absent from that root; desired values”; Changes are “Requested names with different or unproven truncated values; before/after values and truncation indication”; UnchangedRequested are “Requested names with complete, equal observed values”; Untouched are “Existing root names absent from the request”. MutationVariables are “Exactly the union of additions and change after-values; absent/empty for an unchanged instance”; VerificationValues are “All requested values for this instance, including initially unchanged ones”. Enforce “Names in the four classifications are disjoint” and “Every requested name appears once in additions, changes, or unchanged”. Read selected-instance/tenant facts and complete root variables; use UT-compatible normalized equality, reject conflicting duplicate observations, ignore descendant values, and expose no executable plan after any failure. Deep-copy nested objects/arrays across service/facade boundaries. Add new `internal/services/processinstance/update_test.go` and extend `c8volt/process/client_test.go` and `c8volt/client_test.go` for classifications, invalid inputs, stable deduplication, nested-copy isolation, empty internal plans, last-target failure, zero writes, option mapping/error conversion, and constructor compatibility. Run `go test ./internal/services/processinstance/... -run 'Variable' -count=1`, `go test ./c8volt/process -run 'Variable' -count=1`, and `go test ./c8volt -count=1`.

- [ ] T003 [US1] Replace command-owned discovery/comparison in `cmd/update_processinstance_variables.go` with mechanical adaptation of the facade plan, and use it in `cmd/update_processinstance.go` and existing preview views in `cmd/cmd_views_processinstance_update.go`. Retain the prepared plan for subsequent execution integration; carry tenant evidence without rereads. Preserve explicit/repeated/stdin-key validation, JSON approval validation, JSON/verbose rejection, preview fields, root labels, omission/null-array conventions, and existing mode precedence. Extend `cmd/update_processinstance_test.go` with multi-page/capped/sparse/root-isolation/truncated preview cases and collection planning failures through the real command path; assert payload shape, counts, and zero writes. Update preview-related source help/metadata and `README.md`, then run `make docs-content`; do not claim selective execution is fixed yet. Run `go test ./cmd -run 'Test(UpdatePI|UpdateProcessInstance|FormatProcessInstanceVariable)' -count=1` and `git diff --check`.

**Checkpoint / preview MVP**: US1 is demonstrable independently. Ordinary execution remains the legacy path until T005; this preview-only increment is not the completed issue fix.

## Phase 3: User Story 2 — Apply exactly the prepared changes (Priority: P1)

**Goal**: Validate a frozen collection plan, submit only additions/changes locally to its recorded roots, and preserve it across approval.

**Independent test**: Two instances already match; a third needs one addition and one change while another requested value is unchanged. Exactly one mutation reaches the third root, with exactly the two prepared values and `local=true`.

- [ ] T004 [US2] Implement `ExecuteProcessInstanceVariableUpdates` in `internal/services/processinstance/update.go` with new `internal/services/processinstance/update_validation.go`, extending the service interface and thin facade delegation in `c8volt/process/api.go`, `c8volt/process/variable_updates.go`, and `c8volt/process/convert.go` atomically with all required test doubles. Validate/deep-copy the whole plan before writes, checking all T002 classification constraints, root identity, duplicate/conflicting targets, and exact payload correspondence. Enforce “Execution validates correspondence between the classifications, requested values, destination, and payload before any mutation” and “Untouched names never enter a submitted payload”. Execute collection inputs once using existing bounded pool behavior, forwarding fail-fast/no-worker-limit/tenant/cancellation/suppression/no-wait options, and write only nonempty payloads via the existing exact `UpdateScopeVariables` primitive. Include the required single waiter workflow now so the new executor is operationally complete: harden `internal/services/variable/waiter/waiter.go` against truncated equality, verify all frozen requested values at each changed root with complete reads, retain acceptance on failure, and omit verification entirely with no-wait. Do not call the legacy nested-wait writer or add a post-success read. Add domain/public `unchanged` and `skipped` constants in `internal/domain/processinstance_variable.go` and `c8volt/process/model.go`, retaining `submitted`, `confirmed`, `mutation_failed`, `confirmation_failed` and existing fields/JSON tags; unchanged/skipped have `mutationAccepted=false`, submitted/confirmed/confirmation_failed retain true, and mutation_failed has false. Enforce “Result order follows selected-key order regardless of worker completion order” and “Fail-fast cannot undo concurrent in-flight writes; collect their actual outcomes instead of marking them skipped”. Make unchanged successful in `OK()`/totals but never infer acceptance from success. Add new `internal/services/processinstance/update_execution_test.go`, extend `internal/services/variable/waiter/waiter_test.go`, `c8volt/process/client_test.go`, and `c8volt/process/model_test.go` for exact payloads/scopes, forged-plan rejection before all writes, nested-copy isolation, unchanged/empty plans, fail-fast/in-flight/skipped outcomes, partial acceptance, complete/truncated verification, no-wait, and ordered results. Document the additive status values in public comments and `README.md`. Run `go test ./internal/services/processinstance/... -run 'Variable' -count=1`, `go test ./internal/services/variable/... -count=1`, and `go test ./c8volt/process -run 'Variable' -count=1`.

- [ ] T005 [US2] Switch the ordinary execution branch in `cmd/update_processinstance.go` to pass the same prepared plan through approval to facade execution, removing submission of original keys/full input; keep backend mechanics out of `cmd/update_processinstance_variables.go`. Preserve early dry-run/no-op return, existing approval wording/eligibility, configured/inherited stderr, default-answer/EOF/decline behavior, and no writes on abort. Extend `cmd/update_processinstance_test.go` and `cmd/update_processinstance_terminal_test.go` for selective mixed-instance payloads, nested root isolation, planning failure before any writes, retained destinations through approval, repeated/stdin keys, and default/no-wait execution on all supported versions. Update one-page immediate-success fixtures to one planning and one verification read, not three. Include successful-result keys-only filtering in `cmd/cmd_views_processinstance_update.go` using `MutationAccepted`, excluding unchanged/skipped/rejected work. Update `cmd/update_processinstance.go` help/metadata and `README.md` for selective local writes and waiting, regenerate with `make docs-content`, then run `go test ./cmd -run 'Test(UpdatePI|UpdateProcessInstance|FormatProcessInstanceVariable)' -count=1` and `git diff --check`.

**Checkpoint**: Normal and no-wait command execution apply only approved prepared changes. Core verification and truthful service outcomes already work; US3 exposes complete partial-failure reporting at the CLI boundary.

## Phase 4: User Story 3 — Understand what actually happened (Priority: P2)

**Goal**: Retain successful submissions and distinguish unchanged, skipped, mutation-failed, and confirmation-failed work in command results.

**Independent test**: Combine accepted and rejected writes, then separately fail verification after acceptance. Check keyed outcomes, nonzero exit behavior, exactly one result envelope, and accepted-only key output.

- [ ] T006 [US3] Implement retained partial-result rendering and error exit integration in `cmd/cmd_views_processinstance_update.go` and `cmd/update_processinstance.go`, following the UT failure renderer and `exitAfterRenderedResult` pattern. Preserve existing result field names/omission rules and `confirmationStatus` wording; document the additive statuses without inventing an `accepted` status. Enforce “Mutation summaries and keys-only filtering use `MutationAccepted`, not success totals alone”. Emit one failure envelope containing available results, never a second generic envelope; keys-only emits each accepted key once even if later confirmation failed. Keep compact/verbose/quiet/activity and independent DEBUG behavior, and do not reread backend state while rendering. Add new `cmd/update_processinstance_outcomes_test.go` with normal wait, no-wait, delayed/multi-page confirmation, truncated and wrong-scope observations, initially unchanged requested-value drift, partial writer failure, accepted-then-verification failure, and fail-fast/skipped cases. Decode one JSON envelope and require EOF, verify existing nonzero error classification via `testx.RunCmdSubprocess` where needed, separately capture streams, and count requests to prove one verification workflow and zero verification with no-wait. Reuse service tests from T004 rather than rebuilding execution. Update outcome-related help in `cmd/update_processinstance.go` and `README.md`, regenerate via `make docs-content`, then run `go test ./cmd -run 'Test(UpdatePI|UpdateProcessInstance|FormatProcessInstanceVariable)' -count=1`, `go test ./cmd -run 'TestUpdateUserTask' -count=1`, and `git diff --check`.

**Checkpoint**: Both service and CLI preserve partial outcomes without false all-or-nothing reporting.

## Phase 5: User Story 4 — Safely preview or finish with nothing to change (Priority: P2)

**Goal**: Complete valid no-ops without interaction or mutation and preserve dry-run/output/automation contracts.

**Independent test**: Execute all-unchanged plans with real terminal stdin and no response, and run the supported mode/flag matrix with stdout and stderr captured separately. No prompt/write/wait occurs; machine output remains exact.

- [ ] T007 [US4] Complete mode-aware early returns and terminal/output regressions in `cmd/update_processinstance.go`, `cmd/cmd_views_processinstance_update.go`, `cmd/update_processinstance_test.go`, and `cmd/update_processinstance_terminal_test.go`, reusing `testx.NewCmdTerminalRunner`. Enforce “Dry-run never enters execution states” and “An entirely unchanged CLI run renders the existing no-op preview, with `mutationSubmitted=false`, rather than manufacturing mutation reports”. Cover normal and dry-run execution across human, JSON, keys-only, quiet, quiet+JSON, quiet+keys and supported auto-confirm/automation/no-wait combinations; preserve JSON-over-keys precedence and invalid-combination validation. Assert exact human text on its established stream, one JSON envelope followed by EOF with existing null/omission semantics, zero-byte executed-no-op keys, and selected-key lines for dry-run even when unchanged. Require no mutation approval, writes, verification, or rendering-induced discovery; complete sparse-page reads must precede no-op classification. Preserve missing-key/empty-stdin errors, selected-root-with-no-variables additions, planning errors, nonempty outputs, explicit-key paths, and aborts. Real-terminal tests must cover prompt-free no-op completion when otherwise eligible, redirected stdout, configured/inherited stderr, and existing nonempty prompt/default/EOF behavior; mocked terminal checks alone are insufficient. Update no-op/help wording in `cmd/update_processinstance.go` and `README.md` as needed and regenerate with `make docs-content`. Run the targeted command pattern from T006, then the remaining shared-consumer checks in `specs/328-align-pi-variable-updates/quickstart.md` (`go test ./internal/services/usertask/... -count=1`, `go test ./c8volt/... -count=1`) and `make test` for the final shared-reader/waiter/concurrency impact, followed by `git diff --check`. Record actual acceptance results and any platform coverage gaps in `specs/328-align-pi-variable-updates/progress.md`; do not rerun previously passed checks unless later relevant changes invalidate them.

**Checkpoint**: All four stories and the execution-path/terminal matrix are covered; UT and legacy shared consumers remain compatible.

## Dependencies & Execution Order

```text
T001 complete reads
  → US1: T002 service/facade planning → T003 command preview
  → US2: T004 prepared executor → T005 command execution
  → US3: T006 partial outcome presentation
  → US4: T007 complete no-op/terminal acceptance
```

T004 can be verified directly through its service/facade independently of CLI routing, but scheduled execution remains sequential. US3 relies on the executor's correct outcomes; it does not postpone confirmation or partial-result preservation within the service. US4 extends mode integration and coverage, not permission to regress existing no-ops in earlier units.

## Consolidation Review

Reviewed on 2026-10-12: retain seven tasks. There are no setup-only, fixture-only, or polish tasks, and no useful independent implementation pairs to mark parallel.

| Boundary | Why keep it |
| --- | --- |
| T001 / T002 | Paging across three adapters is a shared, independently verifiable change; classification and frozen plans are a separate workflow. |
| T002 / T003 | Planning already includes models, service logic, facade conversion, constructors, and tests. Adding CLI preview integration would make that unit too large. |
| T003 / T004 | Preview integration completes US1; mutation execution introduces a separate lifecycle and safety checks. |
| T004 / T005 | The executor already includes plan validation, concurrency, waiting, statuses, and facade tests. CLI approval and routing can be verified after it works. |
| T005 / T006 | Successful selective execution and partial-error rendering have distinct command contracts; the latter must preserve accepted work and nonzero exits. |
| T006 / T007 | Partial mutation outcomes and prompt-free no-ops require different execution and real-terminal fixtures. Combining them would enlarge an already broad output matrix. |

Keep documentation with each behavior change and final cross-cutting validation in T007. Reuse fixtures and service mechanics across tasks; do not repeat checks without a relevant change. Every unit must compile and pass its checks.

## Requirement Coverage

| Requirements | Primary task ownership |
| --- | --- |
| FR-001 complete pages | T001, T003 |
| FR-002 root filtering/classification | T001–T003 |
| FR-003 truncated equality | T002, T004, T006 |
| FR-004 all planning before writes | T002–T005 |
| FR-005 selective local writes | T004–T005 |
| FR-006 frozen destinations/values | T002, T004–T005 |
| FR-007 one verification workflow | T004–T006 |
| FR-008 no-wait verification opt-out | T004–T007 |
| FR-009 truthful partial outcomes | T004, T006 |
| FR-010 authoritative prompt-free no-op | T002–T005, T007 |
| FR-011 honest dry-run/no-op status | T003, T005, T007 |
| FR-012 output/automation compatibility | T003, T005–T007 |
| FR-013 prompt streams/eligibility | T005, T007 |
| FR-014 grammar/UT compatibility | T001–T007 |
| FR-015 help/documentation alignment | T003–T007 |

SC-001 is demonstrated by T001–T003; SC-002–SC-003 by T004–T005; SC-004–SC-005 by T004–T006; SC-006–SC-007 by T006–T007. Detailed fixture assertions remain in [quickstart.md](quickstart.md).

## Implementation Strategy

Deliver the foundation, then the US1 preview MVP (T001–T003). This proves trustworthy complete planning without claiming that the ordinary update path is fixed. The first usable selective-mutation increment requires US2 through T005. Continue through US3 and US4 for the full issue acceptance; all seven units are required to close #328. Follow the existing Ralph story-scope rule: finish the current unit/story scope, record its results, and leave later stories for their own iteration. Do not run optional implementation or commit hooks merely because task generation completed.
