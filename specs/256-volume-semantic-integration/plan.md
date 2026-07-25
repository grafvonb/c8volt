# Implementation Plan: Volume And Semantic CLI Integration Coverage

**Branch**: `256-volume-semantic-integration` | **Date**: 2026-07-25 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/256-volume-semantic-integration/spec.md`

## Summary

Add a second, deliberately slower integration coverage layer for c8volt that proves the product promise behind `done is done` under larger datasets and longer-running workflows. The baseline all-command suite from feature 255 remains the command-path and representative-behavior gate. This feature adds separate volume targets that validate paging, filtering, critical flag semantics, stdin pipelines, human progress visibility, clean machine output, and ops audit report correctness against real disposable Camunda clusters using the operator's default local configuration.

## Technical Context

**Language/Version**: Go, repository current module toolchain
**Primary Dependencies**: Go `testing`, subprocess execution against the built `c8volt` binary, the existing `integration/cli` harness, default local c8volt configuration, embedded BPMN fixtures, current command contract metadata, and filesystem evidence reports
**Storage**: Filesystem evidence directories outside generated docs; default temporary workdir with optional stable workdir for reruns
**Testing**: Volume targets will use `go test -tags=integration ./integration/cli -run '<VolumeTestName>' -count=1 -timeout=<duration>` through Make targets; `IT_VOLUME_TIMEOUT` controls the slower volume target default while `IT_TIMEOUT` remains available for baseline family targets; normal repository validation remains `make test`
**Target Platform**: Developer/release-validator machines that can reach disposable Camunda 8.7/8.8/8.9 profiles configured in the default local c8volt config
**Project Type**: Go CLI integration harness extension
**Performance Goals**: Keep baseline family targets unchanged; keep each default volume family target bounded by configurable dataset count and a practical volume timeout; avoid timing-sensitive concurrency assertions
**Constraints**: Do not pass `--config`; do not generate private configs; do not override auth mode; tolerate clean and dirty clusters; mutation against selected disposable clusters is allowed; keep evidence out of `docs/`; keep volume targets separate from baseline targets; prefer c8volt commands for setup; record command or embedded BPMN proposals for setup gaps
**Scale/Scope**: Volume coverage for selected command families where deeper data shape matters: read/search, update, cancel, delete, expect/resolve, deploy/embed/run, walk, ops analyse, ops execute, ops purge, and ops repair

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The feature exists to prove visible progress, final outcomes, and observable post-condition evidence for long-running and destructive workflows.
- **CLI-First, Script-Safe Interfaces**: PASS. The plan validates behavior through the built CLI, stdin/stdout, Make targets, root flags, and machine output contracts.
- **Tests and Validation Are Mandatory**: PASS. The implementation will add integration tests and focused validation targets; normal `make test` remains required for non-integration safety.
- **Documentation Matches User Behavior**: PASS. New Make targets and volume-suite behavior require updates to `integration/README.md`; generated CLI docs are not changed unless product command behavior changes.
- **Small, Compatible, Repository-Native Changes**: PASS. The plan extends the existing `integration/cli` harness and evidence model rather than adding a separate framework.

## Project Structure

### Documentation (this feature)

```text
specs/256-volume-semantic-integration/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── volume-targets.md
│   └── evidence-progress-reporting.md
├── checklists/
│   └── requirements.md
├── issue-draft.md
└── tasks.md
```

### Source Code (repository root)

```text
Makefile
integration/
├── README.md
└── cli/
    ├── harness_test.go
    ├── volume_harness_test.go
    ├── volume_data_test.go
    ├── volume_get_test.go
    ├── volume_walk_test.go
    ├── volume_update_test.go
    ├── volume_cancel_test.go
    ├── volume_delete_test.go
    ├── volume_expect_resolve_test.go
    ├── volume_deploy_embed_run_test.go
    ├── volume_ops_analyse_test.go
    ├── volume_ops_execute_test.go
    ├── volume_ops_purge_test.go
    └── volume_ops_repair_test.go
```

**Structure Decision**: Extend the existing build-tagged `integration/cli` package with volume-specific helpers and test files. Keep Make targets as the operator entry point. Do not move baseline tests or fold volume checks into baseline family targets.

## Architecture Grounding

- Feature 255 provides the baseline command inventory, profile selection, seeded data, behavioral scenario evidence, proposal reports, verbose harness logging, and family Make targets.
- This feature adds a volume layer that reuses the same binary-build and subprocess path so root configuration, prompts, stdout, stderr, exit codes, and machine modes are tested as operators see them.
- Volume scenario setup should prefer c8volt commands: embedded fixture discovery/deploy, process-instance start, variable update, cancel/delete/resolve/ops workflows. Direct Camunda setup remains a proposal-recorded fallback only.
- Volume assertions must be dirty-cluster tolerant: verify suite-owned data and bounded/contains behavior rather than exact global counts.
- Critical flag validation must prove semantics, not just acceptance: dry-run no mutation, limit scope caps, keys-only cleanliness, report-file behavior, stdin parsing, fail-fast behavior, and no-wait finality wording.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

- Data model: [data-model.md](./data-model.md)
- Volume target contract: [contracts/volume-targets.md](./contracts/volume-targets.md)
- Evidence/progress/reporting contract: [contracts/evidence-progress-reporting.md](./contracts/evidence-progress-reporting.md)
- Quickstart validation guide: [quickstart.md](./quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The design requires post-condition evidence, final outcome wording, no-wait/submitted distinctions, progress visibility, and ops report validation.
- **CLI-First, Script-Safe Interfaces**: PASS. The design uses subprocess commands, stdin pipelines, keys-only, JSON, quiet, automation, and Make targets.
- **Tests and Validation Are Mandatory**: PASS. The quickstart defines compile, targeted non-destructive, and family-specific volume validation paths; tasks must include destructive real-cluster validation.
- **Documentation Matches User Behavior**: PASS. Make target documentation and volume evidence layout are planned under `integration/README.md`; public/generated docs remain unchanged unless product examples or command behavior change.
- **Small, Compatible, Repository-Native Changes**: PASS. Work is isolated to `integration/cli`, `integration/README.md`, `Makefile`, and this feature's spec artifacts.

## Complexity Tracking

No constitution violations or complexity exceptions are required.
