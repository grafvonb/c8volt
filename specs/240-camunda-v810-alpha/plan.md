# Implementation Plan: Experimental Camunda 8.10 Alpha Support

**Branch**: `240-camunda-v810-alpha` | **Date**: 2026-08-12 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/240-camunda-v810-alpha/spec.md`

## Summary

Add `8.10-alpha` as an explicit experimental runtime target pinned to Camunda `8.10.0-alpha4` while preserving the current default and all Camunda 8.7-8.9 behavior. Generate one isolated product v2 client with machine-readable provenance, add native alpha adapters for every version-aware service family, remove legacy Tasklist fallback from the alpha path, centralize capability decisions, explicitly reuse the C89 embedded fixtures, and keep stable and experimental support visibly distinct in CLI and documentation output.

## Technical Context

**Language/Version**: Go 1.26 with toolchain Go 1.26.2; Bash and Python 3 for client generation
**Primary Dependencies**: Cobra 1.10.2, Viper 1.21.0, oapi-codegen 2.5.0, oapi-codegen runtime 1.4.0, Redocly CLI, PyYAML
**Storage**: Checked-in generated Go client, machine-readable provenance, embedded BPMN fixtures, and Markdown documentation; no database changes
**Testing**: Go `testing`, Testify 1.11.1, targeted package tests, generation guard tests, integration-tag CLI tests, live alpha smoke script, and repository gate `make test`
**Target Platform**: Cross-platform CLI for Linux, macOS, and Windows clients connecting to Camunda 8; generation and release validation use a POSIX shell environment
**Project Type**: Single Go CLI with public facades, internal version-neutral services, version-specific Camunda adapters, and checked-in generated clients
**Performance Goals**: No material regression for existing commands; alpha request, paging, retry, polling, and worker behavior remain within existing command contracts
**Constraints**: Exact upstream tag `8.10.0-alpha4` and commit `4a76f06c9df8ad0a6c64fe88b92c618c2ebf2ef6`; `CurrentCamundaVersion` remains `V88`; plain `8.10` is rejected; no alpha generation may modify v8.7-v8.9 clients; no alpha dependency on removed Operate or Tasklist component APIs
**Scale/Scope**: One runtime identity, one generated product v2 client, eleven versioned service families, two exact-v8.9 capability gates, shared fixtures, version/help/docs surfaces, and one focused live smoke path

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

| Principle | Pre-Research Gate | Post-Design Check |
|-----------|-------------------|-------------------|
| I. Operational Proof Over Intent | PASS: existing mutation confirmation and polling contracts remain mandatory for alpha adapters. | PASS: service contract requires alpha adapters to preserve existing post-mutation confirmation; live smoke includes a supported mutation and observed result. |
| II. CLI-First, Script-Safe Interfaces | PASS: the feature extends the existing version selector and command tree without new command-local runtime routing. | PASS: version identity, help, JSON metadata, error behavior, and unsupported capability handling are defined in `contracts/version-selection.md`. |
| III. Tests and Validation Are Mandatory | PASS: the design includes focused generation, version, factory, adapter, command, docs, and integration checks plus `make test`. | PASS: `quickstart.md` provides runnable focused-to-broad validation and a live pinned-alpha smoke procedure. |
| IV. Documentation Matches User Behavior | PASS: stable and experimental wording is a release gate; generated CLI docs will originate from command metadata. | PASS: README, version output, root help, docs build metadata, API generation guide, and generated CLI docs are explicit design surfaces. |
| V. Small, Compatible, Repository-Native Changes | PASS: existing version packages, factories, facades, helpers, mutations, and integration harness are reused. | PASS: new work stays in existing ownership boundaries; the only new shared abstractions are target-aware generation guardrails and a narrow version-capability helper. |

No constitution violation requires complexity tracking.

## Project Structure

### Documentation (this feature)

```text
specs/240-camunda-v810-alpha/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── generation-contract.md
│   ├── service-compatibility.md
│   └── version-selection.md
├── checklists/
│   └── requirements.md
└── tasks.md                         # created by /speckit-tasks
```

### Source Code (repository root)

```text
toolx/
├── version.go                       # V810Alpha identity, normalization, support sets, C89 fixture mapping
├── version_test.go
├── camunda_capabilities.go          # named capability predicates, including history-safe PD deletion
└── camunda_capabilities_test.go

api/
├── refresh-clients.sh               # target-aware all vs v810alpha workflow
├── 1-fetch-camunda-product-v2-spec.sh
├── 3-generate-clients-from-fetched-specs.sh
├── generate-go-client.sh
├── mutations/                       # existing verified mutation chain
└── README.md                         # pinned alpha reproduction and provenance contract

internal/clients/camunda/v810alpha/camunda/
├── client.gen.go                    # generated only from pinned product v2 source
└── provenance.json                  # source, tag, commit, spec, mutations, generator, command

internal/services/
├── batchoperation/{factory.go,v810alpha/}
├── cluster/{factory.go,v810alpha/}
├── element/{factory.go,v810alpha/}
├── incident/{factory.go,v810alpha/}
├── job/{factory.go,v810alpha/}
├── processdefinition/{factory.go,v810alpha/}
├── processinstance/{factory.go,v810alpha/}
├── resource/{factory.go,v810alpha/}
├── tenant/{factory.go,v810alpha/}
├── usertask/{factory.go,v810alpha/} # unified API only; no Tasklist client
└── variable/{factory.go,v810alpha/}

cmd/
├── root.go                           # stable vs experimental support wording
├── version.go                       # additive machine-readable experimental metadata
├── delete_processdefinition.go      # centralized capability predicate
└── *_test.go                        # help, version, command, fixture, and capability regressions

internal/services/ops/
├── all_process_definitions_purge.go # centralized capability predicate
└── *_test.go

integration/
├── cli/                             # alpha normalization, fixture, and command coverage
└── scripts/
    └── run-c810alpha-smoke.sh        # pinned live smoke and report

README.md
docsgen/
└── main.go                           # stable and experimental generated-doc metadata
docs/cli/                             # regenerated by make docs-content
```

**Structure Decision**: Keep the established single-project layering. `toolx` owns runtime identity and version capabilities; `api/` owns reproducible generated-client preparation; generated types remain under a dedicated client version; each service family owns its alpha adapter; factories remain the only service-selection seam; `cmd` owns user-facing metadata; integration scripts own live release proof. Public facades and version-neutral domain contracts remain unchanged unless an alpha compile difference proves a domain conversion adjustment is necessary.

## Implementation Strategy

### Phase A - Isolated Generated Client Foundation

1. Add a target-aware alpha path to the generation workflow while preserving the existing full stable-client path.
2. Require the tuple `v810alpha` / `8.10.0-alpha4` / `internal/clients/camunda/v810alpha/camunda/client.gen.go`; reject all mismatches before generation.
3. Fetch alpha sources into a temporary target-specific checkout, resolve the annotated tag to commit `4a76f06c9df8ad0a6c64fe88b92c618c2ebf2ef6`, bundle `v2/rest-api.yaml`, and apply the existing five product-v2 mutations in their current order.
4. Generate to a temporary file, verify that stable generated-client tree fingerprints are unchanged, then publish only the alpha client and `provenance.json`.
5. Add shell-level guard tests for wrong tag, wrong target, wrong output, stable-tree protection, provenance completeness, and deterministic regeneration.

### Phase B - Runtime Identity and Capability Model

1. Add `V810Alpha` with canonical string `8.10-alpha`; accept only the explicit aliases in `contracts/version-selection.md` and reject `8.10` and full upstream prerelease tags.
2. Include alpha in supported and implemented discovery while introducing stable/experimental groupings so user-facing output cannot imply stable 8.10 support.
3. Keep `CurrentCamundaVersion = V88` and map `V810Alpha.FilePrefix()` explicitly to `C89_`.
4. Add named capability predicates instead of ordering or exact-version comparisons. Use the history-safe process-definition deletion capability from both command validation and all-definition purge validation.
5. Update connection diagnostics so configured `8.10-alpha` matches gateway `8.10.x` while still warning for another major/minor runtime.

### Phase C - Native Alpha Service Boundary

1. Create `v810alpha` adapters for all eleven version-aware service families and register them explicitly in API assertions and factories.
2. Start from the corresponding v8.9 domain behavior, then compile and adapt against alpha-generated types; do not import any v8.9 generated package from an alpha adapter.
3. Keep alpha user-task lookup on the unified API. Use direct keyed user-task retrieval and tenant-result validation; do not construct or call a Tasklist client after unified lookup misses.
4. Preserve resource deployment confirmation and add alpha-aware retry behavior where the 8.10 eventual-consistency contract can temporarily hide newly deployed resources.
5. Record every family's final support result in the matrix from `contracts/service-compatibility.md`; unsupported behavior must return the shared domain unsupported error before mutation.

### Phase D - CLI, Fixtures, Documentation, and Live Proof

1. Update authored root help, version output, command descriptions, and tests so stable versions and the pinned experimental target are separate.
2. Add experimental metadata additively to version JSON rather than changing existing field types or the command envelope.
3. Reuse C89 fixtures explicitly for alpha in both production fixture filtering and integration fixture selection; do not duplicate BPMN files unless alpha verification exposes an incompatibility.
4. Add the focused alpha smoke script covering connection/authentication, topology, one read, one confirmed mutation, and one deterministic unsupported-target or unsupported-capability probe.
5. Update README and API generation guidance, regenerate `docs/cli/*` and `docs/index.md` with `make docs-content`, then run focused tests followed by `make test`.

## Risk Controls

- **Generated-client collateral changes**: validate target/tag/output before writes, generate through temporary files, fingerprint v8.7-v8.9 trees before and after, and fail on any difference.
- **Prerelease drift**: hard-pin both tag and resolved commit in machine-readable provenance; later alphas require an explicit issue/spec update.
- **Removed legacy APIs**: alpha adapters may import only the alpha unified client; source-level import checks cover Operate, Tasklist, and Administration SM exclusions.
- **Type-shape changes**: compile every adapter against the alpha-generated client and keep generated-to-domain conversion local to each alpha package.
- **Misleading stability claims**: maintain separate stable and experimental metadata and test all authored/generated version wording.
- **Eventual consistency**: retain existing pollers and retries and add focused alpha resource visibility coverage where Camunda 8.10 changed retrieval behavior.
- **Integration environment availability**: automated fake-server and generation tests remain required; the live alpha smoke report is an additional release gate and must be recorded as not run rather than silently skipped.

## Complexity Tracking

No constitution violations or new architectural layers are introduced.
