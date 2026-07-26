# Implementation Plan: All-Command Integration Suite

**Branch**: `255-all-command-integration` | **Date**: 2026-07-25 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/255-all-command-integration/spec.md`

## Summary

Add a destructive all-command integration suite under `integration/` that validates the built `c8volt` CLI against real disposable Camunda clusters using the operator's default local configuration. The suite will use `capabilities --json` as the live command inventory oracle, require explicit coverage for all 55 current command nodes, seed data through c8volt commands first, tolerate clean or dirty clusters, record API/fixture gaps as proposals, and validate help/generated CLI examples without making the rules file part of normal Speckit or Ralph implementation context.

## Technical Context

**Language/Version**: Go, repository current module toolchain
**Primary Dependencies**: Go `testing`, subprocess execution against built `c8volt` binary, Cobra command contract through `c8volt capabilities --json`, existing `integration/` evidence conventions, embedded BPMN fixtures, default local c8volt configuration
**Storage**: Filesystem evidence directory outside `docs/`, defaulting to a temporary work directory and overrideable by suite environment
**Testing**: `go test -tags=integration ./integration/cli -count=1 -timeout=60m` for the suite; normal repository validation remains `make test`
**Target Platform**: Developer/release-validator machines that can run the Go test suite and reach disposable local Camunda 8.7/8.8/8.9 profiles
**Project Type**: Go CLI integration harness
**Performance Goals**: Build the binary once per test run; keep command inventory and profile gates under 30 seconds on a healthy local setup; allow full destructive suite runtime up to 60 minutes
**Constraints**: Do not pass `--config`; do not generate private config; do not override auth mode; use existing local profiles only; tolerate clean and dirty clusters; broad mutation against selected disposable clusters is allowed; keep suite artifacts under `integration/`; do not edit generated `docs/cli/*` as part of harness work
**Scale/Scope**: 55 current command nodes, every command-local flag in the coverage manifest, aliases, output modes, grouping commands, version behavior for configured 8.7/8.8/8.9 profiles, all high-level ops workflows, help and generated CLI example validation

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The suite exists to prove observable command outcomes against real clusters and must verify preview, mutation, wait, and output behavior.
- **CLI-First, Script-Safe Interfaces**: PASS. Coverage is driven through the real CLI binary and the machine-readable command contract, not internal service shortcuts.
- **Tests and Validation Are Mandatory**: PASS. This feature is test infrastructure; tasks must include runnable integration checks and normal unit validation for harness code.
- **Documentation Matches User Behavior**: PASS. The suite validates help/generated examples but does not change public docs unless separate product issues are accepted.
- **Small, Compatible, Repository-Native Changes**: PASS. The suite reuses `integration/`, embedded fixtures, command contract metadata, and existing CLI behavior instead of adding product command surfaces.

## Project Structure

### Documentation (this feature)

```text
specs/255-all-command-integration/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── command-coverage.md
│   └── evidence-reporting.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
integration/
├── AGENTS.md
├── README.md
├── assets/
│   ├── all-command-go-integration-rules.md
│   └── command-matrix.md
├── cli/
│   ├── harness_test.go
│   ├── all_commands_test.go
│   ├── config_test.go
│   ├── get_test.go
│   ├── deploy_embed_run_test.go
│   ├── update_test.go
│   ├── cancel_test.go
│   ├── delete_test.go
│   ├── expect_resolve_test.go
│   ├── walk_test.go
│   ├── ops_analyse_test.go
│   ├── ops_execute_test.go
│   ├── ops_purge_test.go
│   ├── ops_repair_test.go
│   └── examples_test.go
└── scripts/
    └── existing release-suite scripts remain separate
```

**Structure Decision**: Add a dedicated Go integration package under `integration/cli/` and keep reusable suite guidance under `integration/assets/`. Do not place suite rules, generated evidence, or harness docs under public/generated `docs/`. Existing release scripts remain separate; the Go suite may reuse concepts but should not depend on script internals unless a task explicitly adds a shared helper.

## Architecture Grounding

- The suite must run the built CLI as an operator would, so command behavior is validated through root config resolution, command flags, stdout/stderr, prompts, exit codes, and output modes.
- `capabilities --json` is the authoritative command inventory source. Hand-maintained coverage entries are validated against that inventory.
- Data creation must prefer existing c8volt command behavior and embedded BPMN fixtures. Direct Camunda setup is a recorded fallback, not the normal path.
- Reports and generated evidence belong under the suite work directory, not `docs/` or feature specs.

## Ralph / Speckit Isolation

- `integration/assets/all-command-go-integration-rules.md` applies only to this integration-suite feature and future explicit work on the suite.
- Normal Speckit or Ralph implementation runs must not load the rules file unless the user explicitly asks to work on the all-command integration suite or its scripts.
- If Ralph is later used to implement this feature, it must still receive `specs/ralph-implementation-rules.md` for repository layering plus this feature's artifacts for suite scope.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

- Data model: [data-model.md](./data-model.md)
- Command coverage contract: [contracts/command-coverage.md](./contracts/command-coverage.md)
- Evidence/reporting contract: [contracts/evidence-reporting.md](./contracts/evidence-reporting.md)
- Quickstart and validation guide: [quickstart.md](./quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. Design requires subprocess execution, real profile gates, seeded/dirty-cluster proof, and evidence capture for command outcomes.
- **CLI-First, Script-Safe Interfaces**: PASS. The suite consumes public CLI contracts and records CLI output; any direct API fallback must be reported as missing command support.
- **Tests and Validation Are Mandatory**: PASS. Tasks must begin with manifest/profile/read-only harness validation before mutating command families.
- **Documentation Matches User Behavior**: PASS. Example validation is part of the suite; public docs changes remain product work only when accepted separately.
- **Small, Compatible, Repository-Native Changes**: PASS. New work is isolated to `integration/cli/` and feature artifacts, with no production code required for the initial harness.

## Complexity Tracking

No constitution violations or complexity exceptions are required.
