# Implementation Plan: Native Camunda 8.10 Embedded Process Definitions

**Branch**: `275-native-c810-definitions` | **Date**: 2026-08-17 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/275-native-c810-definitions/spec.md`

## Summary

Add eight native `C810_` BPMN definitions derived from the existing `C89_` production family, changing only the permitted version identity and execution-platform metadata. Replace the temporary V810-to-C89 production fixture mapping with V810-to-C810, preserve all stable fixtures and integration assets, verify normalized C89/C810 equivalence and command selection, and correct the normative issue #273 artifacts that describe the superseded fallback.

## Technical Context

**Language/Version**: Go 1.26 with checked-in BPMN 2.0 XML assets

**Primary Dependencies**: Go standard library (`embed`, `io/fs`, XML/text comparison helpers), existing Cobra CLI and Testify test stack; no new dependency

**Storage**: Checked-in BPMN files embedded into the binary; no database or runtime schema

**Testing**: Go package and command tests, normalized fixture-equivalence checks, protected-file diff guards, and `make test`

**Target Platform**: Existing cross-platform c8volt CLI targeting the Camunda 8.10 compatibility line

**Project Type**: Go CLI and public library with embedded static resources

**Performance Goals**: No material runtime change; fixture filtering remains a small in-memory scan of bundled filenames

**Constraints**: Exactly eight C810 definitions; preserve C89 behavior and stable fixture bytes; no live 8.10 integration assets; no new generator; unknown versions continue to fail

**Scale/Scope**: Eight BPMN assets, one production-prefix mapping, focused embed/smoke verification, and corrections to the normative issue #273 design artifacts

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

- **I. Operational Proof Over Intent — PASS**: Existing deploy and smoke workflows retain their confirmation behavior. Focused tests prove V810 selects and reports C810 resources before the full repository gate.
- **II. CLI-First, Script-Safe Interfaces — PASS**: Command names, flags, exit codes, envelopes, and rendering stay unchanged. Only version-matched fixture names change from the temporary C89 fallback to C810.
- **III. Tests and Validation Are Mandatory — PASS**: The design requires close package tests for BPMN invariants and mapping, one shared embed-selector command test, smoke selection/output tests, protected-file and integration-boundary guards, then `make test`.
- **IV. Documentation Matches User Behavior — PASS**: Generic README and generated CLI wording does not name C89 and needs no change. The normative #273 artifacts that explicitly document C89 reuse must be corrected.
- **V. Small, Compatible, Repository-Native Changes — PASS**: The design reuses the existing embedded filesystem, production-prefix selector, C89 fixture pattern, and nearby tests without adding an abstraction, dependency, generator, or integration environment.

**Post-design re-check**: PASS. Phase 1 adds only a fixture-selection contract and validation guide; it introduces no constitutional exception or unresolved clarification.

## Project Structure

### Documentation (this feature)

```text
specs/275-native-c810-definitions/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── embedded-definition-selection.md
└── tasks.md                              # created later by /speckit-tasks
```

### Source Code (repository root)

```text
embedded/
├── fs.go
├── fs_test.go                            # C810 inventory and normalized parity checks
└── processdefinitions/
    ├── C87_*.bpmn                        # protected, unchanged
    ├── C88_*.bpmn                        # protected, unchanged
    ├── C89_*.bpmn                        # behavioral source, unchanged
    └── C810_*.bpmn                       # eight new native definitions

toolx/
├── fixture_compatibility.go              # V810 production prefix becomes C810_
└── fixture_compatibility_test.go

cmd/
├── embed_test.go                         # exact V810 family selected by shared embed helper
└── ops_execute_smoke_test_test.go        # one V810 command-output identity check

internal/services/ops/
├── smoke_test_service.go                 # unchanged consumer of shared prefix mapping
└── smoke_test_test.go                    # V810 fixture and dependency-closure selection

specs/273-camunda-v810-support/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
└── contracts/service-compatibility.md    # replace superseded C89-reuse design
```

**Structure Decision**: Keep assets in the repository's existing embedded BPMN family, keep compatibility ownership in `toolx.ProductionFixturePrefix`, and test behavior at the embedded, selector, service, and CLI boundaries already used by the project. Historical progress, Ralph memory, and completed #273 task descriptions remain historical records; `tasks.md` receives a supersession note instead of rewritten history.

## Design Sequence

1. Add the eight C810 files as controlled counterparts of the C89 family. Replace `C89_` identity references with `C810_`, set the execution-platform version to `8.10.0`, preserve all workflow and diagram content, and do not invent a new Modeler exporter version.
2. Add one embedded-resource test that enumerates the exact family and proves each C810 file equals its C89 source after normalizing only C810/C89 identity and the 8.10/8.9 platform version.
3. Change only the V810 case of `ProductionFixturePrefix` from `C89_` to `C810_`. Leave stable mappings, unknown-version rejection, and integration-only behavior unchanged.
4. Update one embed selection test to assert the exact C810 family, the smoke service tests to assert the C810 parent and dependency closure, and one CLI smoke test to assert the surfaced C810 identity. Retain V89 assertions for C89.
5. Correct the active #273 specification, plan, research, data model, quickstart, and service-compatibility contract to describe the final native C810 decision. Add a supersession note to #273 `tasks.md` without changing its completed task descriptions.
6. Run focused tests, protected-tree and integration-boundary diff checks, then `make test`.

## Complexity Tracking

No constitution violations or additional complexity are required.
