# Quickstart: CLI Debt Refactor Validation

This guide describes how to validate the planned refactor once implementation tasks are generated and completed. It is intentionally a validation guide, not an implementation recipe.

## Prerequisites

- Work from branch `254-cli-debt-refactor`.
- Read `specs/254-cli-debt-refactor/spec.md`, `specs/254-cli-debt-refactor/plan.md`, `specs/254-cli-debt-refactor/research.md`, `specs/254-cli-debt-refactor/data-model.md`, and `specs/254-cli-debt-refactor/contracts/cli-debt-refactor-contract.md`.
- Ralph-driven implementation iterations must also read `specs/ralph-implementation-rules.md` and be launched with `--implementation-context specs/ralph-implementation-rules.md`.
- Use focused validation first, then broaden based on the changed slice.

## Scenario 1: Assessment Baseline Exists

1. Open the checked-in assessment artifact created by implementation tasks.
2. Confirm all 55 command nodes are present.
3. Confirm each command record includes family, mutation behavior, automation support, output modes, paging behavior, progress behavior, ownership, execution style, and high-volume performance risk.

Expected outcome: the assessment is complete enough to order refactor slices without inspecting unrelated command implementations.

## Scenario 2: Basic Search Paging Behavior Remains Compatible

Run targeted tests for the first changed basic search command, then repeat for every additional changed command area.

```bash
go test ./cmd -run 'TestGet(Job|Element|Incident|ProcessInstance)' -count=1
go test ./c8volt/job ./c8volt/element ./c8volt/process -count=1
```

Expected outcome: paged human output, JSON output, keys-only output, totals, prompt behavior, limit trimming, and zero-result behavior remain compatible with the documented baseline.

## Scenario 3: Machine Output Stays Clean

Run or add targeted command tests covering JSON, keys-only, quiet, and automation modes for every changed command.

```bash
go test ./cmd -run 'Test.*(JSON|KeysOnly|Quiet|Automation|NoIndicator|Prompt)' -count=1
```

Expected outcome: JSON emits one valid document, keys-only prints one key per line, quiet mode suppresses nonessential text, and automation mode never prompts unexpectedly.

## Scenario 4: Destructive Workflow Safety Is Preserved

For changed cancel/delete or ops mutation workflows, run targeted command and service tests.

```bash
go test ./cmd -run 'Test.*(Cancel|Delete|Purge|Repair).*' -count=1
go test ./internal/services/ops ./c8volt/ops ./c8volt/process -count=1
```

Expected outcome: dry-run, confirmation, auto-confirm, automation refusal or acceptance, fail-fast, worker controls, partial-completion, and deterministic exit behavior remain covered.

## Scenario 5: High-Volume Performance Is Characterized

Run the fake-latency, benchmark-style, or targeted smoke validation created for the changed slice.

```bash
go test ./cmd ./internal/services/ops ./c8volt/process -run 'Test.*(Latency|Concurrent|Performance|HighVolume|Workers)' -count=1
```

Expected outcome: changed workflows either improve throughput or remain within the documented baseline; any slowdown has a written accepted tradeoff.

## Scenario 6: Help, Docs, And Capabilities Match Behavior

When command metadata, help, examples, flags, aliases, or output contracts change, regenerate docs and run docs/capability checks.

```bash
make docs-content
go test ./cmd ./docsgen -count=1
```

Expected outcome: `--batch-size` and `--limit` wording matches the contract, generated CLI docs are updated, and `capabilities --json` accurately reports changed automation and output support.

## Scenario 7: Broad Validation Before Completion

After targeted tests pass for the changed slice, run the repository validation target.

```bash
make test
```

Expected outcome: full repository validation passes with race-enabled test coverage, generated docs remain in sync, and no unrelated command behavior regresses.

## Validation Log

- 2026-07-24 06:37: Phase 2 T012 passed with `go test ./cmd -run 'TestCommandContract|TestCapability' -count=1`.
- 2026-07-24 06:37: Phase 2 artifact checks passed with `go test ./cmd -run 'TestCapabilityDocumentForRoot_CoversCLIDebtAssessment' -count=1` and `go test ./docsgen -run 'TestCLIDebtRefactorAssessmentArtifactDocumentsBaseline' -count=1`.
- 2026-07-24 06:47: US1 focused output checks passed with `go test ./cmd ./toolx/logging -run 'TestGet(Job|Element|Incident|ProcessInstance).*Progress|TestPagedSearchMachineOutputCleanliness|TestOpsPurge.*Discovery|TestRenderOpsRepair.*Discovery|TestIndicatorEnabled|TestActivityWriter_DisabledSuppressesActivityOutput' -count=1`.
- 2026-07-24 06:47: US1 T025 passed with `go test ./cmd -run 'TestGet(Job|Element|Incident|ProcessInstance)|Test.*JSON|Test.*KeysOnly|Test.*Automation|Test.*NoIndicator|Test.*Prompt' -count=1`.
- 2026-07-24 06:47: US1 activity package validation passed with `go test ./toolx/logging -count=1`; whitespace validation passed with `git diff --check`.
- 2026-07-24 15:15: US2 T045 passed with `go test ./cmd ./c8volt/job ./c8volt/element ./c8volt/incident ./c8volt/process -count=1`.
