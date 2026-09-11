# Tasks: Confirmation Prompts on Stderr

**Input**: Design documents from `specs/298-confirmation-prompts-stderr/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/confirmation-streams.md](contracts/confirmation-streams.md), [quickstart.md](quickstart.md)

**Tests**: Required explicitly by FR-008–FR-009 and the constitution. Terminal-input acceptance must actually execute; pipe-only tests or mocked terminal checks are insufficient.

**Organization**: Shared setup and signature migration precede independently verifiable story increments. US2 reuses the default-no correction from US1. No persistent models, backend services, or generated clients need changes.

## Format: `[ID] [P?] [Story] Description`

All paths are repository-relative. `[P]` marks disjoint work that can run concurrently after its stated prerequisites. Story labels map to the three stories in the specification. All tasks start unchecked; none of the implementation or runtime validation has been performed by task generation.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish the migration inventory and terminal-test interface without changing behavior.

- [x] T001 Audit all references to `confirmCmdOrAbort`, `confirmCmdOrAbortFn`, and `confirmProcessDefinitionSelectorListVisibleFn` under `cmd/`, starting with `cmd/cmd_cli.go` and `cmd/process_definition_selector_validation.go`; record the exact production/test file inventory and baseline focused-test results in `specs/298-confirmation-prompts-stderr/quickstart.md`, using `go test ./cmd -run 'Confirm|Paging|Selector' -count=1`.
- [x] T002 Define the minimal PTY allocator contract and bounded subprocess runner interface in `testx/cmd_terminal_runner.go`, following `testx/cmd_subprocess_runner.go` helper-process environment and exact-test-selection conventions; specify separate stream capture, prompt/response sequencing, canonical EOF, timeout cleanup, and explicit unsupported-platform results before platform work starts.

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Provide actual terminal-input validation and a compile-safe writer interface for every story.

- [x] T003 [P] Implement Linux PTY allocation with the existing `golang.org/x/sys/unix` module in `testx/cmd_terminal_linux.go` and allocator validation in `testx/cmd_terminal_linux_test.go`, using the T002 contract; prove the slave is a terminal, configure canonical input/echo appropriately, and close descriptors on both success and partial failure.
- [ ] T004 [P] Implement Darwin PTY allocation in `testx/cmd_terminal_darwin.go` and allocator validation in `testx/cmd_terminal_darwin_test.go`, using the T002 contract and existing Unix module; add `testx/cmd_terminal_unsupported.go` with complementary build constraints so unsupported platforms compile and are explicitly reported without importing Unix-only code.
- [ ] T005 Complete the isolated child-process runner in `testx/cmd_terminal_runner.go` and validate it in `testx/cmd_terminal_runner_test.go` after T003–T004: attach only stdin to the PTY, capture outputs independently, wait for each complete prompt before sending its answer, use synchronized capture, enforce deadlines, drain/disable echo, and kill/reap/close on timeout; allocation failures on Linux/macOS must fail rather than skip.
- [ ] T006 Add the explicit first `io.Writer` argument to both helper declarations/seams in `cmd/cmd_cli.go` and `cmd/process_definition_selector_validation.go`, migrate every production caller in the migration inventory below to pass `cmd.ErrOrStderr()`, and update all matching test stub signatures found by T001 as one compile-safe change; retain the existing prompt writes for now so story tests can demonstrate the routing defect, and preserve every auto-confirm argument and caller abort/stop branch.
- [ ] T007 Run `go test ./testx -run 'Terminal' -count=1` and `go test ./cmd -run 'Confirm|Paging|Selector' -count=1` after T005–T006; confirm the terminal-support tests are actually discovered and record results and host platform in `specs/298-confirmation-prompts-stderr/quickstart.md` before beginning story work.

**Checkpoint**: Shared support works and all callers compile with the writer interface; production routing is intentionally still unchanged until the story corrections.

## Phase 3: User Story 1 - Confirm an operation while capturing results (Priority: P1) — MVP

**Goal**: Default-no prompts use stderr, preserving ordinary result output and confirmation decisions.

**Independent Test**: With real terminal stdin and captured stdout, accept or abort a default-no operation and compare exact prompt/result bytes. Explicit, inherited, and fallback stderr destinations must work.

### Tests for User Story 1

- [ ] T008 [P] [US1] Add `TestConfirmOrAbortTerminal` and its isolated helper in `cmd/cmd_confirmation_terminal_test.go` for accept, decline, empty answer, and canonical EOF; assert actual terminal stdin, exact literal prompt bytes and spacing, no logger prefixes, standard/nil-writer fallback, and no prompt in stdout; demonstrate the routing assertion fails before T010 without confusing a timeout with successful regression evidence.
- [ ] T009 [P] [US1] Add command-level configured and inherited stderr routing tests in `cmd/cmd_confirmation_command_test.go`, exercising an existing mutation command such as `cmd/delete_processdefinition.go` with fake backend evidence and the real helper; verify result output after acceptance, no mutation after decline, unchanged abort/exit behavior, and the invariant “Never result stdout; custom/inherited stderr is honored.” Demonstrate failure before T010.

### Implementation for User Story 1

- [ ] T010 [US1] Replace the default-no prompt write in `cmd/cmd_cli.go` with `fmt.Fprint` to the supplied writer, falling back to `os.Stderr` for nil; preserve “Preserve formatting and trailing spacing; no logger prefix.” Keep `formatConfirmationPrompt`, terminal checks, scanning, normalization, error conversion, and ignored write-error policy unchanged.
- [ ] T011 [US1] Run the new default-no and command-wiring tests plus existing formatting/mutation regressions with `go test ./cmd -run 'TestConfirmOrAbortTerminal|TestConfirmationCommand|FormatConfirmationPrompt|DeleteProcessDefinition' -count=1`; ensure T009 test names match the filter and record the US1 result in `specs/298-confirmation-prompts-stderr/quickstart.md`.

**Checkpoint**: US1 provides a usable default-no routing correction. Full issue completion still requires US2, US3, documentation, and release validation.

## Phase 4: User Story 2 - Page through keys without polluting the key stream (Priority: P1)

**Goal**: Existing paging questions stay off result stdout while continue/stop behavior remains unchanged.

**Independent Test**: Run real process-instance paging with terminal stdin, accept one continuation, then decline or send EOF. Only fetched keys appear on stdout and no additional page is requested after stopping.

### Tests and Integration for User Story 2

- [ ] T012 [P] [US2] Add `TestGetProcessInstanceKeysOnlyPagingTerminal` and helper scenarios in `cmd/get_processinstance_paging_terminal_test.go`, using the actual paging path in `cmd/get_processinstance_search.go` with a fake service and the T005 runner; verify “No prompt text; keys-only remains one key per line.” across repeated continuation, completion, decline, and EOF, including exact prompt counts, retained keys, and no extra request after stop. For red evidence, run this regression against the pre-T010 write or temporarily restore that write and then restore the correction before completion.
- [ ] T013 [P] [US2] Extend caller-writer and stop-policy regression coverage in `cmd/get_processinstance_paging_test.go`, `cmd/get_incident_test.go`, `cmd/get_job_test.go`, and `cmd/get_element_search_test.go` (create the last file if needed), covering the migrated calls in `cmd/get_processinstance_search.go`, `cmd/get_incident_search.go`, `cmd/get_job_search.go`, and `cmd/get_element_search.go`; preserve “Caller retains its own abort or normal paging-stop interpretation.” and existing JSON/auto-confirm continuation. Do not introduce new paging behavior if T006 and T010 already satisfy these assertions.
- [ ] T014 [US2] Run `go test ./cmd -run 'TestGetProcessInstanceKeysOnlyPagingTerminal|Paging|GetIncident|GetJob|GetElement' -count=1`, verify the real terminal parent test executes, and record independent US2 validation in `specs/298-confirmation-prompts-stderr/quickstart.md`.

**Checkpoint**: Keys-only paging is proven clean through the actual default-no caller path. No separate paging lifecycle redesign is needed.

## Phase 5: User Story 3 - Retain familiar confirmation decisions (Priority: P2)

**Goal**: Correct default-yes routing and preserve all answer, eligibility, and automation behavior.

**Independent Test**: Exercise both defaults with real terminal input across the answer matrix, then verify prompt-skipping and selector-recovery guards through command regressions.

### Tests for User Story 3

- [ ] T015 [P] [US3] Add `TestConfirmOrAbortDefaultYesTerminal` and complete both-default answer matrices in `cmd/cmd_confirmation_terminal_test.go`: `y`, `yes`, mixed case, surrounding whitespace, empty line, other answers, and EOF; enforce “Case and surrounding whitespace handled as before.”, empty-default distinctions, unchanged abort errors, literal `[Y/n]` prompt bytes, custom/fallback destinations, and separate result stdout; demonstrate the default-yes routing failure before T017.
- [ ] T016 [P] [US3] Extend `cmd/process_definition_selector_validation_test.go` and add skip-policy subprocess coverage in `cmd/cmd_confirmation_skip_test.go` for both recovery prompts, configured/inherited stderr wiring, auto-confirm, supported/unsupported automation, non-terminal stdin, JSON, and keys-only; prove “Writer selection does not alter eligibility or cause a skipped read.” with deadline-backed no-input scenarios. Keep redirected-stdout recovery suppressed; use the existing eligibility seam only for caller-wiring coverage, while T015 supplies real terminal proof. Verify accepted recovery lists the expected definitions and decline/EOF preserves the original diagnostic outcome.

### Implementation for User Story 3

- [ ] T017 [US3] Route `confirmCmdOrAbortDefaultYes` through its writer with nil fallback to `os.Stderr` in `cmd/process_definition_selector_validation.go`; retain the two recovery caller destinations migrated by T006, literal formatting, default-yes behavior, scanner/EOF semantics, ignored write-error policy, and all existing caller terminal/output-mode guards.
- [ ] T018 [US3] Run `go test ./cmd -run 'TestConfirmOrAbort.*Terminal|Confirmation.*Skip|ProcessDefinitionSelector|Automation' -count=1`, ensure T016 names match the filter, and record US3 decisions/skip results in `specs/298-confirmation-prompts-stderr/quickstart.md`; resolve any regression without changing existing confirmation policy.

**Checkpoint**: All three stories satisfy the stream contract and compatibility matrix independently.

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Document the intentional stream correction and establish completion evidence.

- [ ] T019 [P] Update the output-contract section in `README.md` to explain plain confirmation and continuation questions on stderr with results on stdout, including redirected output and unchanged selector-recovery/auto-confirm policies; state that prompt consumers must now capture stderr.
- [ ] T020 [P] Update shared command help metadata in `cmd/root.go` and corresponding help assertions in `cmd/root_test.go` to describe the same routing contract without changing flags, aliases, defaults, or examples unrelated to this correction.
- [ ] T021 Run `make docs-content` after T019–T020 and inspect the generated `docs/cli/c8volt.md` and all other changed generated artifacts for consistency with `README.md` and `cmd/root.go`; regenerate from source rather than hand-edit generated Markdown.
- [ ] T022 Format all touched Go files, run the targeted commands in `specs/298-confirmation-prompts-stderr/quickstart.md`, then run `make test` from `Makefile` and `git diff --check`; fix failures and record exact commands, results, and terminal-test execution evidence in `specs/298-confirmation-prompts-stderr/quickstart.md` before marking validation complete.
- [ ] T023 Validate platform isolation with `GOOS=windows GOARCH=amd64 go test -c -o /tmp/c8volt-cmd-298.test.exe ./cmd`, exercise terminal acceptance on Linux/macOS where available, and record executed versus unavailable platforms in `specs/298-confirmation-prompts-stderr/quickstart.md`; inspect `.github/workflows/go.yml` to ensure the new Linux tests are included by its existing race-enabled coverage invocation, without treating compilation as Windows runtime proof.
- [ ] T024 Review the final `cmd/cmd_cli.go` and `cmd/process_definition_selector_validation.go` call-site inventory against T001, confirm no new backend calls or unrelated output changes, and update the validation record in `specs/298-confirmation-prompts-stderr/quickstart.md` and completion checkboxes in `specs/298-confirmation-prompts-stderr/tasks.md` only for work actually completed; verify all FR-001–FR-010 evidence using the coverage table below.

## Dependencies & Execution Order

```text
T001 → T002 → (T003 || T004) → T005 → T006 → T007
                                              ↓
                         US1: (T008 || T009) → T010 → T011
                                              ↓
                         US2: (T012 || T013) → T014
                         US3: (T015 || T016) → T017 → T018
                                              ↓
                     (T019 || T020) → T021 → T022 → T023 → T024
```

- Foundation is required by every story. T006 is deliberately one atomic signature/stub migration to prevent broken intermediate package compilation.
- US1 starts after T007. US2 starts after US1 because it depends on the default-no routing correction; it has its own command-level proof.
- US3 can start after US1 concurrently with US2, because their test files and implementation targets are disjoint. Sequential priority order is US1, US2, US3.
- Every test task precedes its associated behavior change. For the reused US1 behavior, US2 explicitly records regression evidence against the old write; do not fabricate a new production change merely to make a story contain code edits.
- Commands updating the shared quickstart validation record are serialized. Tests that replace global seams must retain existing isolation/cleanup and must not gain unsafe `t.Parallel` calls.
- Polish starts after all story validations. Full-suite success and platform limitations must be reported before completion or any commit claiming implementation completion.

## Parallel Examples

### User Story 1

After foundation, T008 writes terminal helper acceptance in `cmd/cmd_confirmation_terminal_test.go` while T009 writes command-level wiring tests in `cmd/cmd_confirmation_command_test.go`. Both finish before T010.

### User Story 2

After US1, T012 writes real terminal paging acceptance in `cmd/get_processinstance_paging_terminal_test.go` while T013 extends existing resource paging tests in separate files. T014 waits for both.

### User Story 3

After US1, T015 adds the default-yes matrix in `cmd/cmd_confirmation_terminal_test.go` while T016 covers recovery and skip behavior in `cmd/process_definition_selector_validation_test.go` and `cmd/cmd_confirmation_skip_test.go`. T017 follows their regression evidence; T018 follows T017.

Platform allocators T003/T004 and documentation T019/T020 offer additional disjoint work. These are scheduling opportunities, not instructions to launch extra agents automatically.

## Migration Inventory

T006 must cover the exact files below and any additional references found by T001. Test stubs are discovered with `rg -l 'confirmCmdOrAbort|confirmProcessDefinitionSelectorListVisibleFn' cmd --glob '*test.go'` and migrated in the same change.

- Helper ownership: `cmd/cmd_cli.go`, `cmd/process_definition_selector_validation.go`.
- Ordinary mutation callers: `cmd/delete_processdefinition.go`, `cmd/delete_processinstance.go`, `cmd/delete_processinstance_selector.go`, `cmd/cancel_processinstance.go`, `cmd/cancel_processinstance_selector.go`, `cmd/resolve_processinstance.go`, `cmd/update_processinstance.go`, `cmd/update_job.go`.
- Paging callers: `cmd/get_processinstance_search.go`, `cmd/get_incident_search.go`, `cmd/get_job_search.go`, `cmd/get_element_search.go`.
- Ops callers: `cmd/ops_purge_all_processdefinitions.go`, `cmd/ops_purge_orphan_processinstances.go`, `cmd/ops_purge_processinstances_with_incidents.go`, `cmd/ops_execute_retention_policy.go`, `cmd/ops_repair_incident.go`, `cmd/ops_repair_processinstance.go`, `cmd/ops_analyse_slow_process_instances_progress.go`.
- Also inspect `cmd/ops_execute_smoketest.go` references to distinguish related comments from actual calls; update only if the signature migration requires it.

## Requirement Coverage

| Requirement | Tasks |
|---|---|
| FR-001: both helpers and paging use stderr | T006, T008–T010, T012–T013, T015–T017 |
| FR-002: configured stderr and fallback | T008–T010, T013, T015–T017 |
| FR-003: exact plain prompt formatting | T008, T010, T012, T015, T017 |
| FR-004: stdout results and keys only | T008–T009, T012–T014 |
| FR-005: answer/default/EOF decisions | T008, T012, T015–T018 |
| FR-006: guards and skip policies | T006, T013, T016–T018 |
| FR-007: caller outcomes preserved | T009, T012–T014, T016, T024 |
| FR-008: actual terminal input | T002–T005, T008–T009, T012, T015 |
| FR-009: targeted and full race validation | T007, T011, T014, T018, T022–T023 |
| FR-010: aligned user guidance | T019–T021 |

The data model contains transient concepts only; no model structs or migrations are requested. Its observable invariants are quoted in T009, T010, T012, T013, T015, and T016.

## Implementation Strategy

1. Complete setup and foundation, preserving production behavior through the signature migration.
2. Deliver US1 as the first reviewable MVP: default-no terminal confirmation and configured stderr wiring. Validate it before relying on it for paging.
3. Prove US2 paging behavior, then complete US3 default-yes routing and the full compatibility matrix. These are required for issue #298, even if US1 is demonstrated first.
4. Update source guidance, regenerate docs, run targeted and full race-enabled tests, and report platform evidence honestly. Use small Conventional Commits with issue #298 when committing, subject to repository validation rules.
5. Do not mark tasks complete because artifacts exist or a filtered command matched zero tests. Runtime assertions and final checks supply completion evidence.
