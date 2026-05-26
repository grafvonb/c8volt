# Implementation Plan: BPMN Selector Validation for Operational Commands

**Branch**: `207-bpmn-selector-validation` | **Date**: 2026-05-23 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/207-bpmn-selector-validation/spec.md`

## Summary

Extend the shared visible process-definition selector validation from issue #175 to every direct operational `--bpmn-process-id` path that can otherwise make a missing BPMN value look like a valid empty result. The implementation should validate `cancel pi`, `delete pi`, and `get incident` before resource search or mutation planning; audit `get pd` and `delete pd` for explicit direct process-definition selector behavior; preserve valid empty results after successful validation; keep machine modes non-interactive; and update tests and user-facing docs.

## Technical Context

**Language/Version**: Go, repository current module toolchain

**Primary Dependencies**: Cobra command tree, existing process and incident facades, shared process-definition selector validation helpers, process-definition search services, command render/error/prompt helpers, docs generation path

**Storage**: N/A

**Testing**: Go tests through targeted `go test ./cmd`, process and incident command tests where needed, docs generation checks, and final `make test`

**Target Platform**: CLI on the repository's supported platforms

**Project Type**: Go CLI

**Performance Goals**: Add at most one process-definition visibility validation per distinct direct BPMN selector before affected operational searches or mutations; avoid validation when `--bpmn-process-id` is absent; preserve existing paging and valid empty-result behavior after validation succeeds

**Constraints**: Preserve `found: 0` or empty incident output when the process definition is visible; avoid prompts in `--json`, `--automation`, `--keys-only`, `--pi-keys-only`, piped, and non-TTY contexts; include tenant, `--pd-version`, and `--pd-version-tag` where the command supports them; reuse shared diagnostics from issue #175

**Scale/Scope**: `cmd/cancel_processinstance.go`, `cmd/delete_processinstance.go`, `cmd/get_incident.go`, `cmd/get_processdefinition.go`, `cmd/delete_processdefinition.go`, `cmd/process_definition_selector_validation.go`, command tests, command contract tests, README, generated docs, and feature verification docs

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. Operational commands must verify the BPMN process-definition selector before claiming an empty search, dry-run plan, or mutation outcome.
- **CLI-First, Script-Safe Interfaces**: PASS. The feature preserves existing commands and flags while tightening diagnostics and prompt policy for human and machine modes.
- **Tests and Validation Are Mandatory**: PASS. The specification requires missing-selector and valid-empty-result tests for every aligned command.
- **Documentation Matches User Behavior**: PASS. Help, README, and generated CLI docs must describe missing BPMN selectors as validation failures where behavior changes.
- **Small, Compatible, Repository-Native Changes**: PASS. The plan reuses the shared selector validator, process facade, incident command search path, and generated docs flow from existing repository patterns.

## Project Structure

### Documentation (this feature)

```text
specs/207-bpmn-selector-validation/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-bpmn-selector-validation.md
└── tasks.md
```

### Source Code (repository root)
```text
cmd/
├── process_definition_selector_validation.go
├── process_definition_selector_validation_test.go
├── cancel_processinstance.go
├── cancel_test.go
├── delete_processinstance.go
├── delete_test.go
├── get_incident.go
├── get_incident_search.go
├── get_incident_test.go
├── get_processdefinition.go
├── get_processdefinition_test.go
├── delete_processdefinition.go
├── command_contract.go
└── command_contract_test.go

c8volt/
├── process/
└── incident/

internal/
└── services/
    ├── processdefinition/
    └── incident/

README.md
docs/
docsgen/
```

**Structure Decision**: Keep selector validation orchestration in `cmd/`, where output mode, prompt eligibility, and command flow are already owned. Reuse the existing process facade for visibility checks, keep incident resource search behind the incident facade, and update service/facade code only if tests expose missing selector propagation.

## Phase 0: Research

See [research.md](./research.md).

## Phase 1: Design & Contracts

- Data model: [data-model.md](./data-model.md)
- CLI contract: [contracts/cli-bpmn-selector-validation.md](./contracts/cli-bpmn-selector-validation.md)
- Quickstart and verification scenarios: [quickstart.md](./quickstart.md)

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. The contract requires selector visibility validation before protected searches, dry-run plans, confirmations, or mutations.
- **CLI-First, Script-Safe Interfaces**: PASS. Human recovery prompts, structured output, key-only output, automation, and pipeline boundaries are explicitly captured.
- **Tests and Validation Are Mandatory**: PASS. The task list will include failing-first command tests, valid-empty-result tests, docs generation, and repository validation.
- **Documentation Matches User Behavior**: PASS. The design includes README and generated CLI documentation updates for every aligned command.
- **Small, Compatible, Repository-Native Changes**: PASS. The design extends the existing shared validation helper instead of adding a parallel selector framework.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
