# Implementation Plan: Align Camunda 8.10 Guidance

**Branch**: `277-align-v810-guidance` | **Date**: 2026-08-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/277-align-v810-guidance/spec.md`

## Summary

Align every current maintainer and operator guidance source with the Camunda 8.10 support model delivered by issue #273 and finalized by issue #275, while intentionally promoting the omitted-version fallback from V88 to the current stable V89 line. Update version inventories, default-routing regressions, root help, README/config guidance, and generated documentation; preserve historical #273 records, generated clients, embedded definitions, and integration assets.

## Technical Context

**Language/Version**: Go 1.26.2 for existing documentation assertions; Markdown and YAML for guidance artifacts

**Primary Dependencies**: Go standard library, existing Cobra/config rendering paths, existing documentation generator, and repository-native shell/search tooling; no new dependency

**Storage**: N/A; one runtime version constant plus checked-in guidance, tests, generated documentation, and configuration-template files

**Testing**: Focused Go tests for version/default routing, configuration-template and documentation metadata, repository content scans, documentation regeneration review, `git diff --check`, protected-scope diff checks, and `make test`

**Target Platform**: Maintainers and operators of the existing cross-platform c8volt CLI

**Project Type**: Go CLI default-version promotion and guidance refinement

**Performance Goals**: No runtime or CLI performance regression; guidance review and validation remain bounded to the checked-in repository

**Constraints**: Preserve command behavior and output contracts other than intentionally changing the fallback runtime from V88 to V89; keep one V810 identity; retain the current pinned baseline; select only C810 for V810 embedded workflows; preserve historical #273 records; do not touch generated clients, BPMN definitions, or live integration assets

**Scale/Scope**: Two core maintainer guides, current architecture fact/synthesis documents, one shipped configuration template and its focused assertion, two normative #273 gateway requirements, authored gateway help plus its generated page, review-only operator documentation, and the #277 design artifacts

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

- **I. Operational Proof Over Intent — PASS**: The feature changes only omitted-version selection from V88 to V89. Validation proves the new default through configuration and every version-aware factory while preserving gateway-diagnostic and embedded-selection behavior.
- **II. CLI-First, Script-Safe Interfaces — PASS**: Commands, flags, exit codes, and structured output remain unchanged; root help accurately discloses the new default. The plan preserves diagnostic behavior instead of turning warnings into failures.
- **III. Tests and Validation Are Mandatory — PASS**: The design adds or extends the closest useful documentation assertions, runs focused package checks, regenerates authored CLI documentation, guards protected runtime paths, and finishes with `make test`.
- **IV. Documentation Matches User Behavior — PASS**: This feature exists to reconcile maintainer and operator guidance with shipped V810 behavior. Generated CLI pages are updated through command metadata and the repository generator rather than by hand.
- **V. Small, Compatible, Repository-Native Changes — PASS**: The design updates existing guidance and tests in place, introduces no new abstraction or dependency, and preserves historical delivery records.

**Pre-design gate result**: PASS. No constitutional exception or unresolved clarification exists.

## Project Structure

### Documentation (this feature)

```text
specs/277-align-v810-guidance/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── v810-guidance-contract.md
└── tasks.md                              # created later by /speckit-tasks
```

### Source Code (repository root)

```text
AGENTS.md                                 # include V810 in durable repository guidance

specs/
├── ralph-implementation-rules.md         # include V810 in factories, adapters, clients, and tests
└── 273-camunda-v810-support/
    ├── spec.md                           # clarify diagnostic non-match semantics
    ├── contracts/version-selection.md    # authoritative gateway matrix; expected to remain stable
    ├── tasks.md                          # historical C89 tasks and #275 note; preserve
    ├── progress.md                       # historical record; preserve
    └── ralph-memory.md                   # historical record; preserve

.specify/memory/
├── architecture.md                       # update current support synthesis to V810
└── architecture-repo-facts.md            # update observable runtime/client facts to V810

config/templates/
└── config.example.yaml                   # advertise all supported versions and V89 default

toolx/
├── version.go                            # promote omitted-version fallback to V89
└── version_test.go                       # assert V89 as current default

internal/services/*/factory_test.go       # assert CurrentCamundaVersion routes to v89

cmd/
├── root.go                               # disclose V89 default
├── config_test_connection.go             # clarify authored gateway diagnostic help
└── config_test.go                        # protect template/help guidance

README.md                                 # review authoritative operator contract; change only if drift exists
docs/index.md                              # generated README mirror; do not hand-edit
docs/cli/index.md                          # authored CLI support matrix; include V810
docs/cli/c8volt.md                         # generated root contract; do not hand-edit
docs/cli/c8volt_version.md                 # generated baseline disclosure; do not hand-edit
docs/cli/c8volt_config_test-connection.md  # regenerate from authored help
```

**Structure Decision**: Keep every correction with the existing source of truth. Durable agent guidance stays in `AGENTS.md` and Ralph rules, architecture facts stay in `.specify/memory`, gateway requirements stay in the #273 normative specification, operator template/help stays under `config/templates` and `cmd`, and generated CLI documentation continues to come from command metadata and the existing docs generator. Historical #273 records are validation inputs, not edit targets.

## Design Sequence

1. Establish an explicit guidance classification from [data-model.md](data-model.md): active normative guidance may be corrected; generated derivatives follow their source; historical completed records remain unchanged and rely on the existing #275 supersession context.
2. Update the current maintainer and architecture inventories to include V810 wherever all supported adapters, generated clients, capability reviews, or newest-runtime guidance are enumerated. Set V89 as the default, retain version-neutral layering rules, and leave integration-only 8.7–8.9 matrices unchanged because live V810 integration remains out of scope.
3. Correct the shipped configuration template to list 8.10 and distinguish the supported set from the V89 default. Add a focused assertion against rendered template output for the V89 fallback and verify the source-only supported-version comment with a content scan.
4. Refine #273 FR-006 and SC-002 to use the gateway result language already defined by its version-selection contract: same release line is a match; different release line is a diagnostic non-match; empty or unparseable output is an unverifiable diagnostic; neither creates a new hard-failure contract.
5. Clarify the authored `config test-connection` help with the same gateway outcomes, update its nearby test, and regenerate its CLI reference through `make docs-content` rather than editing generated Markdown.
6. Audit all active #273 fixture guidance against #275. Leave its C810 statements unchanged when correct, retain the existing supersession note, and do not rewrite historical checked tasks, progress, or Ralph memory that record the former C89 implementation.
7. Review README, API guidance, the authored CLI landing-page support matrix, and generated root/version documentation against the contract. Change source documentation only for a concrete mismatch and avoid unrelated generated churn.
8. Run focused default-routing and guidance checks from [quickstart.md](quickstart.md), verify production service implementations, generated clients, BPMN, and integration paths have no unintended diff, then complete `make test` and final diff review.

## Post-Design Constitution Check

- **I. Operational Proof Over Intent — PASS**: The gateway and fixture contracts are tied to observable established outcomes, and the validation guide proves documentation alignment without a live mutation.
- **II. CLI-First, Script-Safe Interfaces — PASS**: The contract limits default behavior to the explicit V89 promotion and forbids changes to commands, output contracts, aliases, and diagnostics.
- **III. Tests and Validation Are Mandatory — PASS**: Phase 1 defines focused assertions, content audits, protected-scope checks, documentation regeneration, and the full race-enabled repository test gate.
- **IV. Documentation Matches User Behavior — PASS**: The design assigns one source of truth per guidance topic and treats generated docs as derived artifacts.
- **V. Small, Compatible, Repository-Native Changes — PASS**: The design touches only existing guidance seams and nearby documentation tests; there is no new dependency, generator, or abstraction.

**Post-design gate result**: PASS. Phase 1 introduces no constitutional violation or complexity exception.

## Complexity Tracking

No constitution violations or additional complexity require justification.
