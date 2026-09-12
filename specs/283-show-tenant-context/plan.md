# Implementation Plan: Effective Tenant Context Before Mutations

**Branch**: `283-show-tenant-context` | **Date**: 2026-08-29 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/283-show-tenant-context/spec.md`

## Summary

Expose the meaning of the effective tenant before tenant-sensitive operations without changing tenant selection or authorization. Introduce one version-neutral tenant-context value, derive actual target tenants from resource data already present in frozen plans, and render that evidence consistently in human previews, confirmations, structured results, and operations audit reports. Command code owns configuration interpretation and presentation; internal services own aggregation from resolved resources; public facades map the shared value without adding backend calls.

## Technical Context

**Language/Version**: Go 1.26 (`toolchain go1.26.2`)

**Primary Dependencies**: Cobra 1.10.2, Viper, existing c8volt public facades and version-neutral internal services

**Storage**: N/A for command execution; existing JSON and Markdown audit report files remain the only persisted feature output

**Testing**: Go `testing`, Testify, command execution tests, facade/service unit tests, `make test` (`go test ./... -race -count=1`)

**Target Platform**: Cross-platform CLI targeting Camunda 8.7, 8.8, 8.9, and 8.10

**Project Type**: Go CLI with public facade packages and versioned backend adapters

**Performance Goals**: Tenant-context construction is linear in the already-resolved target set; no additional backend request or polling cycle is introduced

**Constraints**: Preserve tenant-selection behavior, explicit-key backend authorization, output envelope v1, quiet suppression, one-key-per-line output, and existing report formats; do not hand-edit generated clients

**Scale/Scope**: Shared contract plus representative configuration, process-instance, process-definition, job, deploy/run, and operations workflows; aggregation must remain deterministic for paged and bulk plans

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

| Principle | Gate result |
|---|---|
| I. Operational Proof Over Intent | PASS — the context is derived from the same frozen plan or target used for execution and is shown before mutation; no success or verification behavior changes. |
| II. CLI-First, Script-Safe Interfaces | PASS — human safety lines use existing render channels, JSON remains one valid document with the v1 envelope semantics, config YAML remains valid, quiet remains suppressed, and keys-only stdout remains one key per line. |
| III. Tests and Validation Are Mandatory | PASS — the design defines focused builder, service, facade, command, report, and contract coverage followed by `make test`. |
| IV. Documentation Matches User Behavior | PASS — command metadata, README safety guidance, affected ops guides, and generated CLI documentation are in scope. |
| V. Small, Compatible, Repository-Native Changes | PASS — the design reuses existing plan data, facade mappings, result envelopes, preflight renderers, and report writers; it adds no dependency, command hierarchy, flag, or backend call. |

**Post-design re-check**: PASS. The common value type, service-owned aggregation, additive structured fields, and explicit compatibility rules in [data-model.md](./data-model.md) and [contracts/tenant-context.md](./contracts/tenant-context.md) introduce no constitution exception.

## Project Structure

### Documentation (this feature)

```text
specs/283-show-tenant-context/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── tenant-context.md
└── tasks.md                 # generated later by $speckit-tasks
```

### Source Code (repository root)

```text
config/
├── app.go                         # raw tenant, default-target, and display semantics
└── config.go                      # sanitized config YAML production

internal/domain/
├── tenant_context.go              # version-neutral tenant-context value and enums
├── processinstance_traversal.go   # resolved PI plan evidence
└── ops_*.go                       # audit/report tenant context

internal/services/
├── common/tenant_context.go       # deterministic known/unknown tenant aggregation
├── processinstance/dryrun.go      # aggregate traversal data before it is discarded
├── processdefinition/...          # preserve plan-item tenant evidence
└── ops/...                        # aggregate from frozen workflow plans

c8volt/
├── tenant/
│   ├── context.go                 # public tenant-context model
│   └── convert.go                 # domain/public conversion
├── process/...                    # PI plan model/conversion
├── resource/...                   # process-definition plan model/conversion
├── job/...                        # job plan/result mapping where required
├── incident/...                   # repair/resolve mapping where required
└── ops/
    ├── model.go                   # nested context in audit models
    └── convert.go                 # audit model conversion

cmd/
├── cmd_tenant_context.go          # attach execution semantics to command context
├── cmd_views_tenant_context.go    # human labels/warnings and structured view helper
├── command_contract.go            # optional context on shared result envelope
├── cmd_views_contract.go          # propagate attached context into JSON results
├── config_*.go                    # validation/show/connection diagnostics
├── *_processinstance*.go          # cancel/delete/resolve/update plans and prompts
├── delete_processdefinition.go    # definition impact and confirmation
├── update_job*.go                 # job plan and confirmation
├── deploy_processdefinition.go    # creation target before deployment
├── run_processinstance.go         # creation target before process creation
├── ops_*.go                       # preflight, confirmation, and audit enrichment
└── cmd_views_*.go                 # final human/structured rendering

README.md
docs/ops/*.md
docs/cli/*                         # regenerated with make docs-content
```

**Structure Decision**: Keep the established CLI/facade/domain/service layering. Domain and public packages carry data only; `internal/services/common` aggregates already-resolved tenant metadata; command helpers interpret effective configuration, combine it with resolved evidence, and route it through existing views. No generated Camunda client or version-specific adapter change is planned.

## Phase 0: Research

Research decisions and rejected alternatives are recorded in [research.md](./research.md). The phase resolves model ownership, operation-specific empty-tenant semantics, aggregation without enrichment calls, machine-output placement, audit compatibility, YAML scope, and Camunda-version behavior. All planning questions are resolved.

## Phase 1: Design and Contracts

- [data-model.md](./data-model.md) defines the shared context, warning, aggregation rules, relationships, and lifecycle.
- [contracts/tenant-context.md](./contracts/tenant-context.md) defines exact human wording, structured fields, output-mode invariants, report compatibility, and command-family coverage.
- [quickstart.md](./quickstart.md) defines the implementation and validation sequence with representative acceptance checks.

## Implementation Strategy

1. Add the version-neutral and public tenant-context types, conversions, constructors, and deterministic accumulator tests.
2. Extend resolved mutation and operations plans to retain tenant evidence already obtained by existing requests. Aggregate per unique target key, propagate page/workflow summaries, and introduce no enrichment lookup.
3. Add command-context and view helpers. Attach the operation mode after effective configuration is resolved, merge frozen-plan evidence before preview/confirmation, and add the same optional object to structured output.
4. Adopt the shared contract by family: configuration diagnostics; process-instance mutations; process-definition deletion and job update; deploy/run creation; operations preflight and reports.
5. Update close tests for human order, confirmation prominence, JSON/YAML parsing, audit evidence, quiet suppression, and exact keys-only stdout.
6. Update source help/examples, README and ops guides, regenerate CLI docs, then run targeted and full validation.

## Compatibility and Risk Controls

- Empty tenant remains unfiltered discovery for 8.8–8.10 and reflects the normalized default-only behavior already used by 8.7; creation continues to use `TargetTenant()`.
- Explicit keys continue to set `IgnoreTenant`; tenant mismatch is evidence, never a new local rejection.
- Structured output receives an optional `tenantContext` object. Full-contract results add it alongside the existing envelope payload; raw structured views and audit reports use the same object and field names.
- Audit `tenantId` remains as a deprecated compatibility field only when it truthfully denotes a named filter or creation target. It is omitted for unfiltered work and must never be synthesized with `ViewTenant()`.
- Existing `*.v1` report schema identifiers remain because this is an additive optional object plus correction of an optional misleading field. Consumers are told to prefer `tenantContext`; a future removal of `tenantId` would require a schema revision.
- There is no new global YAML option. `config show` remains the existing YAML-producing command and gains a valid nested object in its sanitized document; operations reports remain Markdown or JSON.
- Warnings never alter exit status or confirmation policy. Unknown metadata is non-blocking, and cross-tenant status is based only on distinct known tenants.

## Validation Plan

Run `gofmt` on touched Go files, then focused tests for the common accumulator and each changed family. Verify representative human output occurs before mutation/confirmation, JSON is exactly one parseable document with an unchanged envelope payload, config YAML parses, quiet adds no context lines, and keys-only stdout remains exact. Regenerate docs with `make docs-content`, run `make vet`, then run the constitution-required `make test`. See [quickstart.md](./quickstart.md) for the concrete sequence.

## Complexity Tracking

No constitution violations require justification.
