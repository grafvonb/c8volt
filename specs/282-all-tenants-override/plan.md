# Implementation Plan: All-Tenants Tenant Override

**Branch**: `282-all-tenants-override` | **Date**: 2026-08-30 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/282-all-tenants-override/spec.md`

## Summary

Add a root persistent `--all-tenants` boolean flag that deliberately clears the resolved tenant filter for commands that can operate across the authenticated operator's visible tenants. Apply the override centrally after existing configuration normalization and before command context or service installation. Reuse the existing private tenant-provenance and warning renderer, and use one command annotation as the source of truth for both runtime rejection and capability metadata. The four concrete-destination commands reject the flag before side effects. No facade, service, adapter, generated-client, or authorization behavior changes are required.

## Technical Context

**Language/Version**: Go 1.26 (`go 1.26.0`, toolchain `go1.26.2`)

**Primary Dependencies**: Cobra 1.10.2, pflag 1.0.10, Viper 1.21.0, and the existing `cmd` tenant-context/capability infrastructure

**Storage**: N/A; the feature changes transient CLI configuration and output metadata only

**Testing**: Go `testing`, Testify 1.11.1, command/subprocess tests, mock-backed request assertions, integration example-contract tests, and the repository `make test` race-enabled suite

**Target Platform**: Cross-platform Go CLI targeting Camunda 8.7, 8.8, 8.9, and 8.10

**Project Type**: Go CLI with public facade and versioned internal service adapters

**Performance Goals**: Constant-time local flag validation and tenant resolution; no additional network request, tenant enumeration, retry, or polling step

**Constraints**: Preserve current behavior when the flag is absent; preserve explicit-empty tenant semantics; never let concrete-destination commands reinterpret an empty tenant as `<default>`; preserve backend authorization; preserve existing structured command envelopes and tenant-context schemas; add no config/profile/environment alias for the flag; do not edit generated clients

**Scale/Scope**: One inherited root flag, centralized root validation/resolution, one private provenance extension, one additive command-capability field, four rejected executable leaves, representative discovery/mixed/ops command coverage, generated CLI documentation, README, and operator guides

## Constitution Check

### Pre-design gate

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. Operational Proof Over Intent | PASS | The design requires request-shape assertions, no-side-effect rejection tests, config-precedence coverage, and repository validation rather than relying on flag/help presence. |
| II. CLI-First, Script-Safe Interfaces | PASS | The flag is explicit, invalid combinations remain invalid-input errors, protected output modes stay clean, and capability metadata makes per-command support machine-discoverable. |
| III. Tests and Validation Are Mandatory | PASS | Targeted command/config/rendering tests precede `make test`; documentation regeneration and diff checks are part of completion. |
| IV. Documentation Matches User Behavior | PASS | Root and command source metadata, README, operator guides, generated CLI pages, and integration example validation are all in scope. |
| V. Small, Compatible, Repository-Native Changes | PASS | Existing configuration resolution, empty-filter semantics, warning pipeline, capability document, and command annotations are extended; backend layers are unchanged. |

Project constraints and delivery workflow also pass: the issue-backed branch/folder uses authoritative number `282`; the change remains in established package ownership; generated docs come from source; and no dependency or external service is added.

### Post-design gate

PASS. Phase 1 keeps all behavior inside `cmd` and documentation/test owners. The transient data model adds no persistence or public API. The CLI contract preserves existing output schemas and authorization. The quickstart includes targeted and full validation. No constitution exception or complexity justification is required.

## Project Structure

### Documentation (this feature)

```text
specs/282-all-tenants-override/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── all-tenants.md
└── tasks.md                 # Created later by $speckit-tasks
```

### Source Code (repository root)

```text
cmd/
├── root.go                         # persistent flag and early validation hook
├── root_config.go                  # post-normalization effective tenant override
├── cmd_tenant_context.go           # private override provenance
├── cmd_views_tenant_context.go     # exact human warning and ordering
├── command_contract.go             # command annotation and capability field
├── capabilities.go                 # capability resolution/rendering
├── deploy_processdefinition.go     # concrete-destination annotation
├── embed_deploy.go                 # concrete-destination annotation
├── run_processinstance.go          # concrete-destination annotation
├── ops_execute_smoketest.go        # concrete-destination annotation
└── *_test.go                       # close unit, command, request, and output tests

integration/cli/
└── examples_test.go                # inherited boolean root-flag recognition

README.md                           # root flag and safety contract
docs/ops/                           # central and command-specific operator guidance
docs/cli/                           # regenerated from command source metadata
docs/index.md                       # regenerated from README
```

**Structure Decision**: This is a CLI-only feature. Flag declaration, validation, tenant resolution, capability metadata, and rendering remain under `cmd/` as required by repository layering. Existing public facades and internal services already implement an empty tenant as unfiltered discovery and therefore remain untouched. Documentation is updated at its source and regenerated through the repository workflow.

## Phase 0: Research

Research is complete in [research.md](./research.md). It resolves configuration timing, conflict validation, destination safety, shared capability metadata, warning reuse, authorization boundaries, compatibility, and validation scope. No technical context remains undecided.

## Phase 1: Design and Contracts

- [data-model.md](./data-model.md) defines the transient flag selection, effective tenant filter, private provenance, command support classification, invariants, and state transitions.
- [contracts/all-tenants.md](./contracts/all-tenants.md) defines CLI grammar, precedence, validation, command support, exact warning behavior, output isolation, capability representation, authorization, and compatibility.
- [quickstart.md](./quickstart.md) gives the implementation sequence and targeted/full validation matrix.

## Implementation Strategy

1. Add the command support annotation and enum-like capability type, resolve it through one helper, serialize it in capabilities, and mark the four concrete-destination leaves.
2. Add the inherited boolean flag and run early root validation before config, service, input, prompt, activity, or request work.
3. After current config/profile/environment/`--tenant` resolution and normalization, save private provenance and clear the effective tenant when active.
4. Extend the existing tenant-context renderer for the exact all-tenants warning while preserving once-only behavior and protected output modes.
5. Add root/config/capability/rendering tests, representative discovery request tests, and no-side-effect rejection tests for all four destination commands.
6. Update source help/examples, README, operator guides, and integration flag parsing; regenerate CLI documentation.

## Compatibility and Risk Controls

- `--all-tenants=false` is inactive; absence of the flag leaves every command unchanged.
- An explicitly changed `--tenant`, including `--tenant ""`, conflicts with active `--all-tenants` before configuration loading.
- Override application occurs after normalization so Camunda 8.7 cannot restore `<default>`, but before config enters context or services.
- The four destination commands reject through the same annotation advertised by capabilities, preventing documentation/runtime drift.
- Direct resource-key commands keep existing backend authorization and never receive an authorization-bypass option.
- Public tenant context and result envelopes remain unchanged; command capability gains only an additive field under document version `v1`.
- Protected output modes never receive the human warning; machine-readable output remains parseable and provenance-free.
- A focused inventory test should fail if a future concrete-destination command is added without the rejecting annotation.

## Validation Plan

Run focused `cmd` tests first for root resolution, conflict errors, destination rejection, capability metadata, warnings, output isolation, and representative request bodies. Run the integration example parser after registering the new boolean inherited flag. Regenerate docs with `make docs-content`, run `make vet`, then run `make test` and `git diff --check`. A live asymmetric-tenant authorization scenario is optional/gated because current profiles do not guarantee that identity; deterministic automated tests must still prove omitted discovery filters and unchanged backend error handling.

Complete the specified usability review with representative operators or a documented proxy study: at least 90% must identify and invoke `--all-tenants` within 30 seconds without being directed to `--tenant ""`. Record the participant count, result, and any discoverability correction made before completion.

## Complexity Tracking

No constitution violations require justification.
