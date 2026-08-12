# Implementation Plan: Camunda 8.10 Support

**Branch**: `273-camunda-v810-support` | **Date**: 2026-08-12 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/273-camunda-v810-support/spec.md`

## Summary

Add Camunda 8.10 as one ordinary c8volt compatibility line (`toolx.V810`, canonical value `8.10`) while keeping V88 as the default and disclosing that the initial generated source is Camunda `8.10.0-alpha4`. Prepare one isolated unified client under `internal/clients/camunda/v810/camunda`, record deterministic provenance, add native `v810` adapters and factory cases for all eleven version-aware service families, replace exact-v8.9 checks with named capability predicates, explicitly map v8.10 to compatible C89 production fixtures, and preserve version-neutral facade, CLI, polling, retry, confirmation, output, and error contracts. Generated clients for v8.7-v8.9 and everything under `integration/` remain unchanged.

## Technical Context

**Language/Version**: Go 1.26 with toolchain Go 1.26.2; Bash and Python 3 for generated-client preparation

**Primary Dependencies**: Cobra 1.10.2, Viper 1.21.0, oapi-codegen 2.5.0 and runtime 1.4.0, Redocly CLI, PyYAML

**Storage**: Checked-in generated Go client, deterministic JSON provenance, embedded BPMN fixtures, and Markdown documentation; no database or persistent runtime schema

**Testing**: Go `testing`, Testify 1.11.1, shell generation-guard tests, fake Camunda HTTP servers, source-boundary tests, targeted package tests, documentation generation tests, and race-enabled `make test`

**Target Platform**: Cross-platform CLI for Linux, macOS, and Windows connecting to Camunda 8; generation runs in a POSIX shell environment

**Project Type**: Single Go CLI with public facades, internal version-neutral services, version-specific adapters, and checked-in generated clients

**Performance Goals**: No material regression to existing request, paging, retry, polling, or worker behavior; generated-client preparation is deterministic and a second run produces no diff

**Constraints**: Initial upstream tag `8.10.0-alpha4` at commit `4a76f06c9df8ad0a6c64fe88b92c618c2ebf2ef6`; one `V810`/`v810` family only; `CurrentCamundaVersion` remains V88; no v8.10 Operate, Tasklist, or Administration SM clients; no changes to v8.7-v8.9 generated clients or `integration/`

**Scale/Scope**: One version identity, one unified generated client and provenance record, eleven service adapter families, two shared capability consumers, production fixture mapping, version/config/help/docs surfaces, and stable-version regression coverage

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

| Principle | Pre-Research Gate | Post-Design Check |
|-----------|-------------------|-------------------|
| I. Operational Proof Over Intent | PASS: v8.10 adapters must retain existing confirmation, polling, and observable completion rules. | PASS: service and command contracts require success, failure, and pre-mutation proof for representative workflows. |
| II. CLI-First, Script-Safe Interfaces | PASS: v8.10 extends existing version configuration without a parallel command path. | PASS: [version-selection.md](contracts/version-selection.md) preserves aliases, shared JSON envelope, exit behavior, and compact human output. |
| III. Tests and Validation Are Mandatory | PASS: focused version, generation, factory, adapter, command, boundary, docs, and stable regression tests are required. | PASS: [quickstart.md](quickstart.md) defines runnable focused checks followed by race-enabled `make test`. |
| IV. Documentation Matches User Behavior | PASS: prerelease baseline disclosure and generated command documentation are delivery requirements. | PASS: README, API guidance, command metadata, version output, docs generation, and tests are explicit design surfaces. |
| V. Small, Compatible, Repository-Native Changes | PASS: existing version types, factories, adapters, helpers, facades, and generation tooling are extended in place. | PASS: only a target-isolated generation path, deterministic provenance, and narrow named compatibility helpers are added within existing ownership. |

No constitution violation requires complexity tracking.

## Project Structure

### Documentation (this feature)

```text
specs/273-camunda-v810-support/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── generation-contract.md
│   ├── service-compatibility.md
│   └── version-selection.md
├── checklists/requirements.md
└── tasks.md                         # created by /speckit-tasks
```

### Source Code (repository root)

```text
toolx/
├── version.go                       # V810 identity, aliases, support sets
├── camunda_capabilities.go          # named release capabilities
└── fixture_compatibility.go         # explicit V810 -> C89_ mapping

api/
├── refresh-clients.sh               # existing all-client path plus isolated v810 target
├── 1-fetch-camunda-product-v2-spec.sh
├── 3-generate-clients-from-fetched-specs.sh
├── generate-go-client.sh
├── mutations/                       # ordered existing product-v2 transformations
├── tests/                           # target, provenance, determinism, isolation guards
└── README.md                        # v810 reproduction/update instructions

internal/clients/camunda/v810/camunda/
├── client.gen.go
└── provenance.json

internal/services/
├── batchoperation/{factory.go,v810/}
├── cluster/{factory.go,v810/}
├── element/{factory.go,v810/}
├── incident/{factory.go,v810/}
├── job/{factory.go,v810/}
├── processdefinition/{factory.go,v810/}
├── processinstance/{factory.go,v810/}
├── resource/{factory.go,v810/}
├── tenant/{factory.go,v810/}
├── usertask/{factory.go,v810/}      # unified client only; no Tasklist fallback
├── variable/{factory.go,v810/}
├── incidentfilter/                  # generated-enum dependency made version-neutral
└── ops/                             # capability-based PD purge gate

c8volt/client_test.go                 # full V810 facade construction
cmd/                                  # root/version/config/capability/fixture UX and tests
README.md
docsgen/{main.go,main_test.go}
docs/cli/                             # regenerated by make docs-content
docs/index.md                         # regenerated by make docs-content
```

**Structure Decision**: Preserve the established single-project layering. `toolx` owns version identity and narrow compatibility facts; `api/` owns reproducible generation; the generated client remains isolated by Camunda minor version; each internal service owns conversion and its `v810` adapter; factories remain the adapter-selection seam; public facades stay thin and version-neutral; `cmd` owns operator-facing selection and rendering. The live integration harness remains out of scope and unchanged.

## Implementation Strategy

### Phase A - Reproducible V810 Client Foundation

1. Extend `api/refresh-clients.sh` with a validated `--target v810` mode while preserving the existing no-target all-client workflow.
2. Require the tuple `v810` / selected 8.10 tag / `internal/clients/camunda/v810/camunda`; reject unknown targets, invalid tags, commit mismatches, missing tools, and output escapes before repository writes.
3. Resolve the initial tag to the exact pinned commit, fetch the product v2 spec into a temporary checkout, bundle it, apply the existing five mutations in order, and generate package `camunda` to temporary output.
4. Validate generated syntax and required symbols, build deterministic provenance, fingerprint protected v8.7-v8.9 trees, then publish the v810 client and provenance as one replacement unit.
5. Prove output allowlisting, protected-tree stability, provenance completeness, mutation effectiveness, and second-run determinism.

### Phase B - Version, Gateway, Capability, and Fixture Model

1. Add `toolx.V810`, the four required aliases, and supported/implemented membership; retain V88 as default and reject prerelease-tag aliases.
2. Represent baseline metadata separately from the configured identity so later prerelease/final sources do not change operator configuration or package names.
3. Add an explicit parsed release-line compatibility result. Gateway `8.10`, `8.10.x`, and `8.10.0-alpha4` match configured `8.10`; a different, empty, or unparseable line is not accepted as a match and uses the existing script-visible diagnostic contract.
4. Add named capability predicates for features shared by V89 and V810, beginning with full process-definition history deletion; consume it in direct deletion and all-process-definitions purge without generic version ordering.
5. Add an explicit production-fixture mapping from V810 to `C89_`; retain reporting as `8.10`, add no C810 fixtures, and do not touch integration selection.
6. Remove the generated v89 enum dependency from version-neutral incident filter validation.

### Phase C - Eleven Native V810 Service Adapters

1. Add explicit V810 factory selection and tests for batch operations, cluster, elements, incidents, jobs, process definitions, process instances, resources, tenants, user tasks, and variables.
2. Start from corresponding v89 domain behavior, then adapt local contracts/conversions to v810 generated shapes; never import/delegate to v87-v89 adapters or generated clients.
3. Keep variable construction inside the v810 process-instance service on the v810 variable implementation.
4. Use only the unified v810 client; removed Tasklist, Operate, and Administration SM fallbacks are forbidden. Return existing explicit domain outcomes when the baseline cannot fulfill a workflow.
5. Preserve version-neutral domain, facade, paging, traversal, waiter, retry, polling, error conversion, and confirmation helpers.
6. Add compile-time assertions, representative success/error/malformed-response tests, fake-server mutation proof, and source-boundary scans for every family.

### Phase D - CLI, Documentation, and Regression Proof

1. Update root help and human `version` output to list `8.10` and disclose `8.10.0-alpha4 (prerelease)` separately.
2. Preserve the version JSON envelope and fields; add string baseline metadata fields.
3. Replace hard-coded “8.8 and 8.9” or exact-v8.9 wording with capability-accurate “or newer” descriptions where V810 is verified.
4. Add top-level client, configuration, gateway, fixture, command output/prompt/activity, and stable-default tests.
5. Update README/API guidance, regenerate docs using `make docs-content`, and verify authored/generated wording.
6. Confirm zero new differences in protected clients and `integration/`, then run targeted checks and `make test`.

## Risk Controls

- **Stable-client collateral changes**: validate before writes, use temporary storage, enforce output allowlists, fingerprint v8.7-v8.9, and keep the legacy refresh path unchanged.
- **Prerelease drift**: pin tag/commit; hash source, transformations, prepared spec, and output; explicitly update provenance for every in-place advance.
- **Mutation no-ops**: assert expected schema changes and required generated symbols.
- **Removed APIs**: source tests reject Operate, Tasklist, Administration SM, older generated clients, and cross-version service imports.
- **Type-shape changes**: localize conversion in each v810 package and compile all interfaces/factories before command work.
- **Misleading stability claims**: keep canonical `8.10` separate from baseline tag/status and test human/JSON disclosure.
- **Capability drift**: use explicit named sets, not string/ordinal comparison.
- **Fixture ambiguity**: use a named mapping instead of treating fixture prefix as identity.
- **Integration expansion**: require an unchanged `git diff -- integration`; use fake servers until separate stable certification.

## Complexity Tracking

No constitution violations or new architectural layers are introduced.
