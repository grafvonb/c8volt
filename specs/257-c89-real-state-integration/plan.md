# Implementation Plan: C89 Real-State Semantic Integration Coverage

**Branch**: `257-c89-real-state-integration` | **Date**: 2026-07-25 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/257-c89-real-state-integration/spec.md`

## Summary

Add a third integration coverage layer for c8volt that proves the highest-risk command semantics against real Camunda 8.9 cluster state. Feature 255 remains the all-command command-path and flag breadth baseline. Feature 256 remains the volume, progress, report, pipeline, and critical-flag semantics layer. This feature closes the remaining real-state gaps by creating or discovering actual jobs, incidents with related jobs, listener state, BPMN error job paths, retention candidates, purge/delete candidates, and mixed failure targets using the operator's default local c8volt configuration. Where c8volt commands or embedded BPMN fixtures cannot create a required state, runtime tests must record skipped-prerequisite or dry-run-covered evidence while spec-owned gap artifacts describe the missing capability.

## Technical Context

**Language/Version**: Go, repository current module toolchain

**Primary Dependencies**: Go `testing`, subprocess execution against the built `c8volt` binary, the existing `integration/cli` harness, Make targets, default local c8volt configuration, existing embedded BPMN fixtures, command manifest metadata, filesystem evidence reports, and Camunda 8.9 profiles selected from the default config

**Storage**: Filesystem evidence directories under the integration workdir; feature planning artifacts under `specs/257-c89-real-state-integration`; no generated CLI documentation output for this feature unless product command behavior changes

**Testing**: Real-state targets will use `go test -tags=integration ./integration/cli -run '<RealStateTestName>' -count=1 -timeout=<duration>` through Make targets; normal non-integration guard remains `go test ./integration/cli -count=1`; normal repository validation remains `make test`

**Target Platform**: Developer and release-validator machines that can reach disposable Camunda 8.9 clusters configured in the default local c8volt config

**Project Type**: Go CLI integration harness extension

**Performance Goals**: Keep 255 baseline targets and 256 volume targets separate; keep real-state slices focused enough to run independently; avoid timing-sensitive assertions; prefer deterministic state checks over sleep-heavy polling

**Constraints**: Do not pass `--config`; do not generate private config files; tolerate clean and dirty clusters; destructive mutation against the selected disposable cluster is allowed; prefer c8volt commands for setup; prefer embedded BPMN models; direct Camunda API setup is allowed for controlled dirty-state or worker-state simulation when c8volt intentionally prevents that state; keep runtime evidence separate from spec-owned setup and fixture gaps; keep reusable context outside `docs/`; focus current implementation on Camunda 8.9 while preserving version fields for future minor releases

**Scale/Scope**: Real-state coverage for active jobs, job mutations, incidents with related jobs, listener state, BPMN error jobs, deterministic retention candidates, confirmed purge/delete/cancel/resolve/repair outcomes, fail-fast and partial-failure behavior, and spec-owned gap tracking

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The feature is explicitly about observable real cluster post-state and truthful final outcomes.
- **CLI-First, Script-Safe Interfaces**: PASS. Validation goes through the built CLI, Make targets, stdin/stdout, reports, and default profile selection.
- **Tests and Validation Are Mandatory**: PASS. The plan adds build-tagged integration tests plus non-integration guard checks; `make test` remains required for normal validation before final merge.
- **Documentation Matches User Behavior**: PASS. Integration README and feature-local quickstart updates are planned; generated CLI docs change only if command examples or behavior change.
- **Small, Compatible, Repository-Native Changes**: PASS. Work extends the existing `integration/cli` harness, evidence model, gap tracking artifacts, and Make target style.

## Project Structure

### Documentation (this feature)

```text
specs/257-c89-real-state-integration/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── coverage-matrix.md
├── gaps.md
├── contracts/
│   ├── real-state-targets.md
│   └── real-state-evidence.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
Makefile
integration/
├── README.md
└── cli/
    ├── harness_test.go
    ├── deploy_embed_run_test.go
    ├── volume_seed_test.go
    ├── real_state_harness_test.go
    ├── real_state_data_test.go
    ├── real_state_jobs_test.go
    ├── real_state_incidents_test.go
    ├── real_state_listeners_test.go
    ├── real_state_bpmn_error_test.go
    ├── real_state_retention_test.go
    ├── real_state_destructive_test.go
    └── real_state_gap_validation_test.go
```

**Structure Decision**: Extend the existing build-tagged `integration/cli` package and Make target conventions. Keep real-state tests separate from all-command and volume tests so release validators can run the three layers independently.

## Architecture Grounding

- Feature 255 supplies command inventory, family target style, selected-profile handling, default local config rules, and scenario evidence.
- Feature 256 supplies volume datasets, progress/report/pipeline evidence, critical-flag semantics, and destructive target conventions.
- This feature adds real-state fixtures and assertions where the current suite is too shallow: non-empty jobs, observable job mutations, incidents with related jobs, listener jobs, BPMN error job paths, retention/purge/delete candidates, and fail-fast/partial-failure reporting.
- The harness must assert suite-owned identifiers and stable containment rather than global exact counts, because the selected cluster may already be dirty.
- Direct Camunda API setup is allowed only when c8volt commands cannot create the required state; each usage must be visible in runtime setup evidence, and missing c8volt command capability must be maintained in `gaps.md` when it is a safe product capability.
- Some direct Camunda API setup is intentionally test-only, such as deleting a call-activity parent below c8volt's cascade-safe deletion path to create orphan children. These cases must be marked as controlled corruption setup, not as product command proposals.
- Missing embedded BPMN behavior must remain visible in `gaps.md`; runtime tests should record skipped-prerequisite evidence instead of silently passing.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

- Data model: [data-model.md](./data-model.md)
- Real-state target contract: [contracts/real-state-targets.md](./contracts/real-state-targets.md)
- Real-state evidence contract: [contracts/real-state-evidence.md](./contracts/real-state-evidence.md)
- Quickstart validation guide: [quickstart.md](./quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The design requires pre-state and post-state evidence, accepted/no-wait distinctions, and skipped-prerequisite evidence when live proof is blocked.
- **CLI-First, Script-Safe Interfaces**: PASS. The contracts define Make targets, subprocess execution, default config usage, stdout cleanliness, and report evidence.
- **Tests and Validation Are Mandatory**: PASS. The quickstart defines local compile checks, gap artifact checks, and staged destructive Camunda 8.9 validation.
- **Documentation Matches User Behavior**: PASS. The quickstart and contracts cover operator-facing integration behavior; generated docs remain untouched unless command examples change.
- **Small, Compatible, Repository-Native Changes**: PASS. Work is scoped to the existing integration harness, Make targets, integration README, and feature-local specs.

## Complexity Tracking

No constitution violations or complexity exceptions are required.
