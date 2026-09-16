# Tasks: Consistent Listener Timestamps

**Input**: Design documents in `specs/321-listener-timestamps/`.
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [output contract](contracts/listener-timestamps.md), and [quickstart.md](quickstart.md).
**Branch**: `codex/321-listener-timestamps` · **Issue**: #321.

**Tests**: Regression tests are required by the originating issue's acceptance criteria and the feature's acceptance scenarios. Extend existing tests rather than duplicating coverage. Write behavioral regression assertions before their corresponding implementation; demonstrate the expected failure when executable, then verify they pass after the change. Documentation-only work does not trigger runtime tests.

**Organization**: Shared transport prerequisites precede independently verifiable user-story increments. Paths below are repository-relative. All checkboxes describe future implementation work; task generation does not mark implementation complete.

## Format: `[ID] [P?] [Story] Description`

`[P]` identifies tasks that can run concurrently with the listed peers after their prerequisites are complete. Story labels apply only within story phases. Complete each phase's validation before declaring its checkpoint achieved. Run `gofmt` on touched Go files before the relevant validation, and retain earlier passing evidence unless subsequent changes invalidate it.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm existing project context; no new dependencies or scaffolding are needed.

- [x] T001 Verify `AGENTS.md`, `.specify/memory/constitution.md`, and `specs/321-listener-timestamps/plan.md` against the current branch and feature pointer in `.specify/feature.json`; inspect existing test helpers and confirm the Go toolchain from `go.mod`. If executing through Ralph, also read `specs/ralph-implementation-rules.md` and surface any conflicting instructions before implementation.

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Make recorded timestamps available to every listener workflow without altering discovery or lifecycle behavior.

- [x] T002 Add regression assertions in `internal/domain/job_test.go` for `RuntimeListenerJobFromJob`, covering both timestamps, each independently absent, neither present, preserved offsets, and a retained non-active deadline; enforce “Non-nil creation/end values retain the same instant and timezone offset through mappings.”
- [x] T003 Add `CreationTime *time.Time` and `EndTime *time.Time` with `json:"creationTime,omitempty"` and `json:"endTime,omitempty"` to `Job` and `RuntimeListenerJob` in `internal/domain/job.go`, and copy them in `RuntimeListenerJobFromJob`; enforce “Nil is absence, and fields are independent.” and “JSON preserves supplied deadlines in every state.” (depends on T002).
- [x] T004 [P] Extend get/search conversion fixtures in `internal/services/job/v88/service_test.go`, then map both pointers in `internal/services/job/v88/convert.go:fromJobSearchResult`; cover distinct supplied values, each missing, both missing, and explicit null without a version cutoff. Enforce “v87 lookup remains unsupported; v88/v89/v810 responses with missing fields remain valid.” (depends on T003).
- [x] T005 [P] Extend get/search conversion fixtures in `internal/services/job/v89/service_test.go`, then map both pointers in `internal/services/job/v89/convert.go:fromJobSearchResult`; cover independent missing/null values, retained deadlines, and unchanged instants without synthesizing timestamps (depends on T003).
- [x] T006 [P] Extend get/search conversion fixtures in `internal/services/job/v810/service_test.go`, then map both pointers in `internal/services/job/v810/convert.go:fromJobSearchResult`; cover independent missing/null values, retained deadlines, and unchanged instants without synthesizing timestamps (depends on T003).
- [x] T007 Validate the domain and adapter changes using the domain/job commands in `specs/321-listener-timestamps/quickstart.md`, and run existing unsupported tests in `internal/services/job/v87/service_test.go` by their actual names; confirm generated files under `internal/clients/camunda/` are unchanged and no discovery requests were added (depends on T004–T006).

**Checkpoint**: All adapters preserve available timestamps and domain listener projection preserves them; absence remains valid. No user-story phase starts until this checkpoint passes.

## Phase 3: User Story 1 — Distinguish Listener Creation, End, and Deadline (Priority: P1) — MVP

**Goal**: `get element --with-listeners` displays trustworthy listener lifecycle facts.

**Independent Test**: Execute keyed and search element lookups using fixture jobs with different creation, end, and deadline times. Completed/canceled/non-active rows omit `d:`; activated rows show only an available deadline; missing creation/end values omit their tags independently.

### Tests for User Story 1

- [x] T008 [P] [US1] Add timestamp-column and rendered-row regression tests in new `cmd/cmd_views_listener_test.go` and existing `cmd/cmd_views_element_test.go`; cover completed, activated, canceled, created, failed, blank, unfamiliar, and noncanonical lowercase states, both listener kinds, all creation/end presence combinations, absent deadlines, activated jobs with supplied end times, mixed rows with worker/error columns, full-date millisecond precision, and timezone-offset display. Enforce “Human rows show each available creation/end time independently of state.” and “Only exact `ACTIVATED` plus non-nil deadline produces human `d:`.”
- [x] T009 [P] [US1] Extend keyed/search execution fixtures in `cmd/get_element_test.go` with timestamp-bearing listener responses and capture stdout/stderr separately; assert exact human tags, no deadline substitution, unchanged nesting and duration, and unchanged request counts, retaining no-listener, without-enrichment, validation, and error coverage.

### Implementation for User Story 1

- [x] T010 [US1] Extend listener mapping assertions in `c8volt/element/client_test.go`, then add optional timestamp fields in `c8volt/element/model.go` and copy them in `c8volt/element/convert.go:fromDomainRuntimeListenerJob`; use the field types/tags from T003 and enforce “nil means not requested; a pointer to an empty slice means requested with no matches.” for the existing listener collection.
- [x] T011 [US1] Implement the private timestamp-column helper in new `cmd/cmd_views_listener.go` and use it in `cmd/cmd_views_element.go:flatRowElementListenerWithTimezone`; return exactly three columns in `s:`, `e:`, `d:` order after worker and before errors, with empty strings for absent/ineligible columns. Reuse `toolx/timestamp.go:FormatTime` and existing flat-row alignment without changing either helper or the standalone job renderer (depends on T008–T010).
- [x] T012 [US1] Document the element command's creation-versus-worker-start meaning, end time, activated-only deadline, and omitted missing values in `cmd/get_element.go` and `README.md`; add a full-date completed-listener example, update affected help assertions in `cmd/get_element_test.go`, and run `make docs-content` to regenerate affected content under `docs/cli/`.
- [x] T013 [US1] Run `go test ./c8volt/element -run 'Listener|Timestamp' -count=1` and `go test ./cmd -run 'GetElement|ElementListener|ListenerTimestamp' -count=1`, explicitly selecting new tests if names differ; verify the independent scenarios from `specs/321-listener-timestamps/spec.md` and review generated element docs before marking US1 complete (depends on T011–T012).

**Checkpoint**: US1 can be demonstrated independently on a real element command path. This is the MVP, not completion of issue #321.

## Phase 4: User Story 2 — Read the Same Timeline Across Investigation Commands (Priority: P1)

**Goal**: Process get, walk, and slow analysis use the same listener grammar and preserve their own layouts and duration results.

**Independent Test**: Feed equivalent listener facts into all four commands with offset display on and off; compare tags/values and verify identical process/element durations and analysis results for unchanged inputs.

### Tests for User Story 2

- [x] T014 [P] [US2] Extend human-output execution tests in `cmd/get_processinstance_test.go` and `cmd/walk_test.go`, plus row assertions in `cmd/cmd_views_processinstance_activity_test.go`, for timestamp-bearing listeners; cover both listener kinds and completed/activated/canceled/other non-active cases, process get keyed/list paths, walk default/children/parent/flat modes, missing fields, offsets, unchanged tree ownership and request counts, and uncontaminated result streams.
- [x] T015 [P] [US2] Extend human-output tests in `cmd/cmd_views_ops_slow_process_analysis_test.go` and command execution coverage in `cmd/ops_analyse_slow_process_instances_test.go` for the same listener states and independent missing timestamps; exercise actual command dispatch with existing service/HTTP fixtures, normal and full-timeline output, timezone display, and separate stdout/stderr capture.

### Implementation for User Story 2

- [x] T016 [P] [US2] Extend facade mapping coverage in `c8volt/process/client_test.go`, add optional timestamp fields in `c8volt/process/model.go`, copy them in `c8volt/process/convert.go:fromDomainRuntimeListenerJob`, and use the US1 helper in `cmd/cmd_views_processinstance_activity.go:flatRowProcessInstanceElementListenerWithTimezone`; preserve fields/collection constraints from T003 and T010 (depends on T014 and completed US1).
- [x] T017 [P] [US2] Extend facade mapping coverage in `c8volt/ops/client_test.go`, add optional timestamp fields in `c8volt/ops/model.go`, copy them in `c8volt/ops/convert.go:fromDomainRuntimeListenerJob`, and use the US1 helper in `cmd/cmd_views_ops_slow_process_analysis.go:flatRowOpsSlowProcessAnalysisListenerWithTimezone`; preserve fields/collection constraints from T003 and T010 (depends on T015 and completed US1).
- [x] T018 [US2] Extend existing enrichment/analysis fixtures in `internal/services/element/enrichment_test.go`, `internal/services/processinstance/enrichment_test.go`, and `internal/services/ops/slow_process_analysis_test.go` to assert timestamp retention, unmatched-job omission, stable ordering/request counts, and zero differences in process/element durations and slow-analysis outcomes for identical execution data; enforce “No new state transitions, mutations, persistence, migrations, duration values, or validation of lifecycle ordering are introduced.”
- [x] T019 [US2] Update listener guidance/examples in `cmd/get_processinstance.go`, `cmd/walk_processinstance.go`, `cmd/ops_analyse_slow_process_instances.go`, and `README.md` for the common timestamp contract; extend relevant help assertions in `cmd/get_processinstance_test.go`, `cmd/walk_test.go`, and `cmd/ops_analyse_slow_process_instances_test.go`, then run `make docs-content` and review affected generated references under `docs/cli/`.
- [x] T020 [US2] Run the facade, enrichment/analysis, and command test selections from `specs/321-listener-timestamps/quickstart.md` relevant to this story, ensuring the new tests actually execute; compare all four human outputs and preserve existing unsupported-version, without-listener, empty-array, and invalid-mode regression results (depends on T016–T019).

**Checkpoint**: All four human views share the timestamp semantics; durations and investigation behavior remain unchanged.

## Phase 5: User Story 3 — Retain Available Timestamps for Programmatic Consumers (Priority: P2)

**Goal**: Public job retrieval and all listener JSON results expose available times and omit missing values without losing recorded deadlines.

**Independent Test**: Retrieve timestamp-bearing and partially populated jobs through public get/search/page interfaces and decode each command's listener JSON. Verify exact property names, equivalent instants, missing-field omission, retained non-active deadlines, and unchanged envelopes/collections.

### Tests for User Story 3

- [x] T021 [P] [US3] Extend public job get/search/page conversion tests in `c8volt/job/client_test.go` with creation-only, end-only, both, and neither cases, plus numeric offsets and retained non-active deadlines; add marshaling assertions enforcing “Missing or null source values remain nil and are omitted from public JSON.”
- [x] T022 [P] [US3] Extend JSON execution tests in `cmd/get_element_test.go`, `cmd/get_processinstance_test.go`, `cmd/walk_test.go`, and `cmd/ops_analyse_slow_process_instances_test.go` to decode exactly one existing envelope and require EOF; verify `creationTime`/`endTime`, independent omission, retained completed/canceled deadlines, requested-empty/unrequested listener distinctions, unchanged request counts, and separate stdout/stderr. Use actual execution results rather than only view fixtures and retain unsupported-version coverage.
- [x] T023 [P] [US3] Extend public listener JSON assertions in `c8volt/element/client_test.go`, `c8volt/process/client_test.go`, and `c8volt/ops/client_test.go` for exact names, optional values, preserved instants/offsets, and retained deadlines; verify populated, nil, and requested-empty listener collections without changing their schemas.

### Implementation for User Story 3

- [x] T024 [US3] Add `CreationTime *time.Time` and `EndTime *time.Time` with the T003 JSON tags to `c8volt/job/model.go` and map both in `c8volt/job/client.go:fromDomainJob`, including existing list/page paths; enforce “Missing or null source values remain nil and are omitted from public JSON.” and “JSON preserves supplied deadlines in every state.” Leave standalone human formatting in `cmd/cmd_views_job.go` unchanged (depends on T021).
- [x] T025 [US3] Run `go test ./c8volt/job -run 'TestClient_GetJob|TestClient_SearchJobs|Timestamp' -count=1`, explicitly run page tests not matched, and run the relevant listener facade/command JSON tests from `specs/321-listener-timestamps/quickstart.md`; verify the complete programmatic contract and correct any mapping omissions in the owning converter identified by a failing regression (depends on T022–T024).

**Checkpoint**: Public job consumers and all four command JSON outputs preserve available timestamp facts with no fabricated times or deadline loss.

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T026 Audit the final diff against FR-001–FR-010 in `specs/321-listener-timestamps/spec.md` and the matrix in `specs/321-listener-timestamps/contracts/listener-timestamps.md`; review `README.md` and generated `docs/cli/` output for consistent definitions and intentional compatibility changes, ensure touched Go files are formatted, and run `git diff --check`. Do not regenerate documentation again unless source guidance changed after the last generation.
- [ ] T027 Reconcile actual test names and validation evidence with `specs/321-listener-timestamps/quickstart.md` and record checks, results, and material gaps there; run only missing or invalidated targeted checks. Use `make test` only if the actual diff or unresolved failures meet the broader-validation conditions in `specs/321-listener-timestamps/plan.md`; do not rerun tests merely to finish or commit. Live inspection remains optional and must not be reported as executed unless performed.

## Dependencies & Execution Order

### Phase dependencies

```text
T001 Setup
  -> T002 -> T003
       -> T004 / T005 / T006 -> T007 Foundation gate
           -> US1 T008–T013 (MVP)
               -> US2 T014–T020
                   -> US3 T021–T025
                       -> T026 -> T027
```

The graph shows the recommended sequential delivery order. US2 depends on the US1 helper. US3 standalone-job work (T021/T024) can begin after the foundation gate; its all-command contract tasks and completion gate wait for US1/US2 mappings and execution-test edits. Do not concurrently edit the same command/facade test files across story phases.

### Within-phase dependencies

- T004–T006 are independent once T003 completes; T007 waits for all three.
- T008/T009 can be authored concurrently after T007. T010 supplies the public element model; T011 waits for the US1 test tasks and model mapping. T012 follows integration and T013 validates the full slice.
- T014/T015 can be authored concurrently after US1. T016/T017 can run concurrently after their respective tests exist; they touch different files and consume the already-complete helper. T018 is independent of their file edits but must finish before T020. T019 runs after T014/T015 to avoid conflicting command-test edits.
- T021/T022/T023 can run concurrently after US2; they touch disjoint test files. T024 follows T021, and T025 waits for all story changes.
- T026/T027 wait for all three story checkpoints.

### Parallel examples per story

- **US1**: Author helper/element-row regression cases (T008) alongside element command execution cases (T009).
- **US2**: Author process-get/walk cases (T014) alongside slow-analysis cases (T015); after those finish, implement process facade/rendering (T016) alongside ops facade/rendering (T017).
- **US3**: Author public job cases (T021), command JSON cases (T022), and public listener JSON cases (T023) concurrently. Public job mapping (T024) can follow T021 while the other test tasks finish.

These are scheduling opportunities, not an instruction to launch agents or autonomous implementation during task generation.

## Requirement Traceability

| Requirement | Tasks |
| --- | --- |
| FR-001 Preserve timestamps through every representation | T002–T007, T010, T016–T018, T021, T023–T025 |
| FR-002 Creation/end semantics | T008–T013, T014–T017, T019–T020 |
| FR-003 Activated-only human deadline | T008–T011, T014–T017, T020 |
| FR-004 Independent absence/no substitution | T002–T006, T008–T011, T014–T017, T021–T025 |
| FR-005 All four commands/both listener kinds | T008–T020, T022 |
| FR-006 Formatting and tag order | T008, T011, T014–T017, T020 |
| FR-007 Additive JSON and preserved contracts | T010, T016–T017, T021–T025 |
| FR-008 Version-specific missing data | T004–T007, T020, T022 |
| FR-009 Unchanged durations/discovery/grouping | T009, T014–T018, T020, T022, T026 |
| FR-010 Documentation | T012, T019, T026 |

SC-001/SC-002 are demonstrated by the human execution and row tests; SC-003 by adapter/facade/JSON checks; SC-004 by T009/T018/T020; SC-005 by timezone regressions and T012/T019/T026.

## Implementation Strategy

### MVP first

Complete Setup, Foundation, and US1 (T001–T013). Demonstrate trustworthy element listener timestamps with targeted tests and matching docs. This delivers the smallest usable slice while leaving the other investigation commands explicitly unfinished.

### Incremental delivery

Complete US2 to extend the shared grammar to all human investigation views, then US3 to close public job and JSON verification. Existing listener timestamp mappings delivered in earlier stories are reused, not reimplemented in US3. Finish with the cross-cutting review and accurate validation record.

Keep commits small and Conventional Commits compliant, referencing #321 when committing. Do not require a full test suite solely for a commit. No new API client generation, dependencies, polling, mutation behavior, timestamp format, or public shared-type refactoring is needed.
