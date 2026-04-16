# Tasks: Audit and Fix CLI Config Precedence

**Input**: Design documents from `/specs/107-flag-precedence-audit/`
**Prerequisites**: [plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/plan.md) (required), [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/spec.md) (required for user stories), [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/research.md), [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/data-model.md), [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/quickstart.md), [contracts/config-precedence.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/contracts/config-precedence.md)

**Tests**: Automated tests are REQUIRED for this feature because the specification explicitly requires exhaustive regression coverage across config-backed command paths, root persistent flags, command-local flags, mixed-source combinations, zero/empty-value cases, and explicit ambiguity failures.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Capture the current precedence surface and align the feature docs with the concrete command/config seams before shared implementation begins.

- [x] T001 Inventory config-backed command paths and current precedence seams in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_flagpacks.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/research.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/quickstart.md
- [x] T002 [P] Inspect existing config/bootstrap regression seams for tenant, profile, and shared failure handling in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/config_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/app_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/errors_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/*test.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish one authoritative precedence resolver and shared binding model before any story-specific coverage or documentation work begins.

**⚠️ CRITICAL**: No user story work should begin until this phase is complete.

- [x] T003 Define the authoritative precedence contract and effective-config resolver seams in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/contracts/config-precedence.md
- [x] T004 Refactor profile application into a lower-precedence field overlay in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/*.go
- [x] T005 [P] Normalize shared command-local config-backed bindings into the same resolver path in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_flagpacks.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run.go
- [x] T006 [P] Add or update shared config/bootstrap helper coverage for the normalized resolver in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/app_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/config_test.go

**Checkpoint**: The repository has one shared precedence resolver, profile overlay semantics are repository-native and field-aware, and command-local config-backed flags no longer rely on a conflicting precedence path.

---

## Phase 3: User Story 1 - Resolve Effective Values Consistently (Priority: P1) 🎯 MVP

**Goal**: Ensure every config-backed setting resolves with `flag > env > profile > base config > default` across the CLI, with the highest-risk baseline settings behaving consistently everywhere they appear.

**Independent Test**: Run representative commands that exercise `tenant`, active profile selection, API base URLs, auth mode, and auth credentials/scopes with flag, env, profile, base config, and default inputs, and confirm the effective winner always matches the shared contract.

### Tests for User Story 1

- [x] T007 [P] [US1] Add root bootstrap precedence tests for tenant, profile selection, and config-file loading in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/config_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/bootstrap_errors_test.go
- [x] T008 [P] [US1] Add config-level precedence and overlay tests for active profile, API base URLs, auth mode, and auth credentials/scopes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/app_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/errors_test.go
- [x] T009 [P] [US1] Add command regression tests that verify baseline settings resolve consistently in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go

### Implementation for User Story 1

- [x] T010 [US1] Implement the authoritative effective-config resolution flow in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config.go
- [x] T011 [US1] Align root persistent baseline setting bindings and defaults with the shared resolver in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/*.go
- [x] T012 [US1] Verify and normalize baseline setting consumption in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_cli.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_services.go, and config-backed command files under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/

**Checkpoint**: User Story 1 is independently testable: the shared precedence contract works for the named critical baseline settings across the audited command surface.

---

## Phase 4: User Story 2 - Preserve Correctness Across Command Types and Edge Cases (Priority: P2)

**Goal**: Keep precedence behavior correct for root persistent flags, command-local flags, zero/empty values, and ambiguous combinations, without silent fallbacks or command-specific drift.

**Independent Test**: Exercise commands that combine root persistent flags with command-local config-backed flags and edge-case values, then verify explicit higher-precedence empty/zero values are preserved when valid and ambiguity cases fail through the shared CLI error model.

### Tests for User Story 2

- [x] T013 [P] [US2] Add command-local precedence regression tests for shared backoff/config-backed flags in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go
- [x] T014 [P] [US2] Add edge-case tests for explicit empty/zero values and non-fallback behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/app_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/*test.go
- [x] T015 [P] [US2] Add subprocess or shared-failure-model tests for ambiguous-precedence validation failures in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_subprocess_scope_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/config_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/bootstrap_errors_test.go

### Implementation for User Story 2

- [x] T016 [US2] Implement shared command-local binding and precedence handling for config-backed flag packs in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_flagpacks.go and config-backed root command files under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/
- [x] T017 [US2] Implement explicit ambiguity and invalid-value failure handling in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_errors.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/bootstrap_errors.go
- [x] T018 [US2] Normalize zero/empty-value preservation and non-fallback behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/app.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/auth.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/api.go

**Checkpoint**: User Story 2 is independently testable: root and command-local config-backed flags follow the same rules, valid empty/zero winners are preserved, and unsafe ambiguity fails explicitly.

---

## Phase 5: User Story 3 - Trust the Contract Through Shared Coverage and Documentation (Priority: P3)

**Goal**: Make the precedence contract durable through shared internal coverage and operator-facing documentation that both describe the same behavior.

**Independent Test**: Review the updated tests and docs, then confirm maintainers can find one authoritative internal contract and operators can understand the same precedence rules in the CLI/config docs without reading source code.

### Tests for User Story 3

- [ ] T019 [P] [US3] Add or update shared regression coverage that proves the critical baseline settings are checked everywhere they appear in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/*test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/*test.go
- [ ] T020 [P] [US3] Add documentation-alignment validation notes and quickstart verification cases in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/quickstart.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/contracts/config-precedence.md

### Implementation for User Story 3

- [ ] T021 [US3] Update shared internal precedence guidance and traceability notes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/research.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/data-model.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/contracts/config-precedence.md
- [ ] T022 [US3] Update operator-facing precedence and override guidance in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/config_show.go
- [ ] T023 [US3] Regenerate affected CLI reference output under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/ from updated Cobra help text and docs commands

**Checkpoint**: User Story 3 is independently testable: the precedence contract is covered, documented, and reviewable from one internal and one operator-facing perspective.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish the full audit trail, validate the feature end to end, and leave the planning artifacts aligned with the shipped result.

- [ ] T024 [P] Refresh implementation and verification notes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/quickstart.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/plan.md
- [ ] T025 Run targeted precedence validation with `go test ./config -count=1` and focused `go test ./cmd ... -count=1` commands from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt
- [ ] T026 Run repository validation with `make test` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion and is the MVP slice.
- **User Story 2 (Phase 4)**: Depends on Foundational completion and is best implemented after User Story 1 because it builds on the same authoritative resolver.
- **User Story 3 (Phase 5)**: Depends on Foundational completion and is best implemented after User Stories 1 and 2 because it locks in audit coverage and docs against the final behavior.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: No functional dependency on later stories after Foundational work is complete.
- **User Story 2 (P2)**: Functionally independent for testing, but implementation should reuse the shared resolver established for User Story 1.
- **User Story 3 (P3)**: Functionally independent for testing, but implementation should target the final audited behavior after User Stories 1 and 2 settle the resolver and failure model.

### Within Each User Story

- Add or update regression tests before considering the story complete.
- Shared config/bootstrap changes before command-level cleanup that depends on them.
- Edge-case and ambiguity handling before documentation updates that describe the final contract.
- User-facing docs regeneration only after Cobra help text and effective behavior are stable.

### Parallel Opportunities

- `T001` and `T002` can run in parallel.
- `T005` and `T006` can run in parallel after the shared resolver shape is decided.
- `T007`, `T008`, and `T009` can run in parallel.
- `T013`, `T014`, and `T015` can run in parallel.
- `T019` and `T020` can run in parallel.
- `T024` can run in parallel with targeted validation once implementation behavior is stable.

---

## Parallel Example: User Story 1

```bash
# Prepare User Story 1 precedence coverage in parallel:
Task: "Add root bootstrap precedence tests for tenant, profile selection, and config-file loading in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/config_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/bootstrap_errors_test.go"
Task: "Add config-level precedence and overlay tests for active profile, API base URLs, auth mode, and auth credentials/scopes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/app_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/errors_test.go"
Task: "Add command regression tests that verify baseline settings resolve consistently in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go"
```

## Parallel Example: User Story 2

```bash
# Prepare User Story 2 edge-case coverage in parallel:
Task: "Add command-local precedence regression tests for shared backoff/config-backed flags in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go"
Task: "Add edge-case tests for explicit empty/zero values and non-fallback behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/config_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/app_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/*test.go"
Task: "Add subprocess or shared-failure-model tests for ambiguous-precedence validation failures in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_subprocess_scope_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/config_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/bootstrap_errors_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Stop and validate the shared precedence contract for the critical baseline settings before expanding to edge cases and documentation.

### Incremental Delivery

1. Finish Setup + Foundational once.
2. Deliver User Story 1 as the MVP for the shared precedence contract.
3. Add User Story 2 to close root-vs-command-local drift, zero/empty-value handling, and explicit ambiguity failures.
4. Add User Story 3 to lock in the audit matrix and operator-facing documentation.
5. Finish with targeted validation and full `make test`.

### Parallel Team Strategy

1. One contributor handles Setup + Foundational work.
2. After Foundational is complete:
   - Contributor A: User Story 1 resolver and baseline precedence coverage.
   - Contributor B: User Story 2 edge-case and command-local drift coverage.
   - Contributor C: User Story 3 documentation and audit-traceability work once behavior stabilizes.
3. Finish with shared validation and repository-wide tests.

---

## Notes

- [P] tasks are limited to work on different files with no dependency on unfinished tasks.
- [US1], [US2], and [US3] map directly to the user stories in [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/107-flag-precedence-audit/spec.md).
- This feature’s commit subjects must keep Conventional Commit formatting and append `#107` as the final token.
- Run `make test` before committing, per repository rules.
