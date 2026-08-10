# Quickstart: Command Mode And Concern Reorganization Validation

This guide describes how to validate the planned reorganization once implementation tasks are generated and completed. It is a validation guide, not an implementation recipe.

## Prerequisites

- Work from branch `270-cmd-mode-reorg`.
- Read `specs/270-cmd-mode-reorg/spec.md`, `specs/270-cmd-mode-reorg/plan.md`, `specs/270-cmd-mode-reorg/research.md`, `specs/270-cmd-mode-reorg/data-model.md`, and `specs/270-cmd-mode-reorg/contracts/command-mode-reorg-contract.md`.
- Ralph-driven implementation iterations must also read `specs/ralph-implementation-rules.md` and be launched with `--implementation-context specs/ralph-implementation-rules.md`.
- Use focused validation first, then broaden based on the changed slice.

## Scenario 1: Process-Definition Watch Ownership Is Focused

Review the process-definition command files and tests after the watch slice is complete.

```bash
go test ./cmd -run 'TestGetProcessDefinition.*Watch|TestProcessDefinition.*Watch' -count=1
```

Expected outcome: watch lifecycle behavior is located in focused watch ownership, ordinary process-definition command behavior remains separate, and existing watch/repaint behavior is unchanged.

## Scenario 2: Resource Views Remain Presentation-Only

Run renderer-focused tests after splitting resource views.

```bash
go test ./cmd -run 'TestRender|Test.*View|Test.*JSON|Test.*KeysOnly' -count=1
```

Expected outcome: process-instance, process-definition, incident, resource, tenant, and flat-row rendering stays output-compatible, and renderer files do not perform facade calls or backend orchestration.

## Scenario 3: Process-Instance Support Files Preserve Behavior

Run process-instance command and view tests after moving dry-run, paging, progress, search, and mutation-result concerns.

```bash
go test ./cmd -run 'TestGetProcessInstance|TestCancelProcessInstances|TestDeleteProcessInstances|Test.*ProcessInstance.*DryRun|Test.*ProcessInstance.*Paging|Test.*ProcessInstance.*Progress' -count=1
```

Expected outcome: search paging, dry-run output, cancellation, deletion, prompts, JSON/keys-only cleanliness, and progress behavior remain compatible with the baseline.

## Scenario 4: Large Command Workflows Stay Reviewable

Run targeted tests for job update, root command behavior, slow-process analysis, and ops progress/report slices as those areas are reorganized.

```bash
go test ./cmd -run 'TestUpdateJob|TestRoot|TestSlowProcess|TestOps.*Progress|TestOps.*Report|TestRenderOps' -count=1
```

Expected outcome: command wiring, request parsing, worker outcomes, validation, progress, rendering, and report serialization are independently identifiable without behavior changes.

## Scenario 5: Destructive Workflow Safety Is Preserved

For cancel/delete and ops mutation slices, run destructive workflow coverage.

```bash
go test ./cmd -run 'Test.*(Cancel|Delete|Purge|Repair|Retention).*' -count=1
```

Expected outcome: confirmation, auto-confirm, automation, force, dry-run, fail-fast, worker controls, partial-completion reporting, and deterministic exit behavior remain covered and unchanged.

## Scenario 6: Command Package Validation Passes

After each major reorganization slice, run the command package tests.

```bash
go test ./cmd -count=1
```

Expected outcome: the full command package remains behavior-compatible after the slice.

## Scenario 7: Documentation Has No Unintended Diff

Check generated CLI documentation after all behavior-preserving moves.

```bash
make docs-content
git diff -- docs README.md
```

Expected outcome: generated docs and README show no unintended user-facing changes. Any intended diff has a separate compatibility note and matching tests.

## Scenario 8: Full Repository Validation Passes

Run final formatting, whitespace, and full validation.

```bash
gofmt -w $(rg --files cmd c8volt internal/services internal/domain toolx | rg '\.go$')
git diff --check
make test
```

Expected outcome: formatting is stable, whitespace checks pass, and the full repository validation target passes before completion.

## Validation Log

- 2026-08-10 13:45 Iteration 3 foundational baseline: `go test ./cmd -run 'TestCommandContract|Test.*View' -count=1` passed (`ok github.com/grafvonb/c8volt/cmd 0.524s`).
- 2026-08-10 14:10 Iteration 3 US1 watch checkpoint: `go test ./cmd -run 'TestGetProcessDefinition.*Watch|TestProcessDefinition.*Watch|TestValidateGetProcessDefinitionWatch|TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata' -count=1` passed (`ok github.com/grafvonb/c8volt/cmd 0.623s`).
- 2026-08-10 14:10 Iteration 3 US1 non-watch checkpoint: `go test ./cmd -run 'TestGetProcessDefinition|TestProcessDefinitionSelectorValidationHelpContract' -count=1` passed (`ok github.com/grafvonb/c8volt/cmd 5.001s`).
- 2026-08-10 15:05 Iteration 15 US2 renderer ownership guard: `go test ./cmd -run '^TestGetViewFilesAvoidBackendOwnership$' -count=1` passed (`ok github.com/grafvonb/c8volt/cmd 0.540s`).
- 2026-08-10 15:05 Iteration 15 US2 renderer compatibility checkpoint: `go test ./cmd -run 'Test.*View|TestRender|Test.*JSON|Test.*KeysOnly|Test.*Flat' -count=1` passed (`ok github.com/grafvonb/c8volt/cmd 1.101s`).
- 2026-08-10 16:02 Iteration 24 US3 dry-run renderer ownership audit: `go test ./cmd -run 'Test(GetViewFilesAvoidBackendOwnership|.*DryRun)' -count=1` passed (`ok github.com/grafvonb/c8volt/cmd 0.872s`).
