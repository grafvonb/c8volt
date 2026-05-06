# Implementation Plan: Validate Process Definition Selectors for Process-Instance Commands

**Branch**: `175-validate-pd-selectors` | **Date**: 2026-05-06 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/175-validate-pd-selectors/spec.md`

## Summary

Add a shared process-definition selector preflight for process-instance commands that use `--bpmn-process-id`. The implementation should validate visible process definitions through the existing process facade before searching, mutating, or starting process instances; preserve valid empty process-instance results when the process definition exists; fail clearly in automation modes; and optionally offer interactive users a visible process-definition listing using the existing process-definition output format.

## Technical Context

**Language/Version**: Go, repository current module toolchain  
**Primary Dependencies**: Cobra command tree, existing process facade, process-definition search services, process-instance search/mutation/start command paths, shared render/error/prompt helpers, docs generation path  
**Storage**: N/A  
**Testing**: Go tests through targeted `go test ./cmd`, `go test ./c8volt/process`, process-definition service tests where needed, docs generation checks, and final `make test`  
**Target Platform**: CLI on the repository's supported platforms  
**Project Type**: Go CLI  
**Performance Goals**: Add at most one process-definition visibility validation per distinct BPMN process ID selector before affected process-instance operations; avoid validation when `--bpmn-process-id` is absent; keep existing process-instance paging behavior unchanged after validation succeeds  
**Constraints**: Preserve `found: 0` for visible process definitions with zero matching process instances; avoid prompts in `--json`, `--automation`, `--keys-only`, and non-TTY contexts; validate all run/multi-ID selectors before any create or mutation request; reuse existing tenant/version/version-tag resolution and process-definition list rendering patterns  
**Scale/Scope**: `cmd/get_processinstance.go`, `cmd/cancel_processinstance.go`, `cmd/delete_processinstance.go`, `cmd/run_processinstance.go`, command tests, process facade tests, process-definition search contracts if gaps appear, README/generated docs, and validation scripts

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. Commands must confirm the selected process definition is visible before reporting an empty process-instance set or submitting mutations/starts.
- **CLI-First, Script-Safe Interfaces**: PASS. The behavior is exposed through existing flags, exit paths, prompt eligibility, and structured-output conventions.
- **Tests and Validation Are Mandatory**: PASS. The spec requires tests for missing selectors, valid empty results, selector context, multi-ID all-or-nothing behavior, and prompt suppression.
- **Documentation Matches User Behavior**: PASS. User-visible diagnostics and command examples require README/generated docs updates.
- **Small, Compatible, Repository-Native Changes**: PASS. The plan reuses the current process facade, shared PI selector flags, and process-definition output renderer rather than introducing a separate validator command.

## Project Structure

### Documentation (this feature)

```text
specs/175-validate-pd-selectors/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-process-definition-selector-validation.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_processinstance.go
├── get_processinstance_test.go
├── cancel_processinstance.go
├── cancel_test.go
├── delete_processinstance.go
├── delete_test.go
├── run_processinstance.go
├── run_test.go
├── get_processdefinition.go
├── cmd_views_get.go
├── command_contract_test.go
└── process_api_stub_test.go

c8volt/
└── process/
    ├── api.go
    ├── client.go
    ├── client_test.go
    ├── convert.go
    └── model.go

internal/
├── domain/
│   └── processdefinition.go
└── services/
    └── processdefinition/
        ├── api.go
        ├── v87/
        ├── v88/
        └── v89/

README.md
docs/
docsgen/
```

**Structure Decision**: Keep command-level selector construction and prompt policy in `cmd/`; reuse `process.API.SearchProcessDefinitions` and `SearchProcessDefinitionsLatest` for visibility checks; use service/facade changes only if tests expose missing selector, tenant, version, or version-tag propagation.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

- Data model: [data-model.md](./data-model.md)
- CLI contract: [contracts/cli-process-definition-selector-validation.md](./contracts/cli-process-definition-selector-validation.md)
- Quickstart and verification scenarios: [quickstart.md](./quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The contract requires successful process-definition validation before process-instance search, mutation, or start work begins.
- **CLI-First, Script-Safe Interfaces**: PASS. The design defines human prompts, no-prompt automation behavior, structured-output compatibility, and preserved aliases.
- **Tests and Validation Are Mandatory**: PASS. The task list will include failing-first command and facade tests plus targeted and repository-wide validation.
- **Documentation Matches User Behavior**: PASS. The quickstart and task list include README/generated docs updates for the new diagnostic behavior.
- **Small, Compatible, Repository-Native Changes**: PASS. The planned helper centralizes behavior in the existing command package and keeps service/facade changes incremental.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
