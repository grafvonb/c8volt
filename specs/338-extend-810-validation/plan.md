# Implementation Plan: Extend Release Validation to Camunda 8.10

**Branch**: `338-extend-810-validation` | **Date**: 2026-10-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/338-extend-810-validation/spec.md`, issue #338.

## Summary

Extend the existing integration harness with explicit release-version selection, complete per-version evidence, 8.10 real-state execution, batch and tenant scenarios, and aligned supporting guards. Preserve existing family targets and older coverage. Use a small integration-only runner and pure coverage reducer to prevent skipped tests, missing evidence, or overwritten reports from appearing successful. Reuse existing fixture discovery, CLI subprocess helpers, and report records.

Research resolved two important boundaries: DRAINING is a process-definition state, and physical routing is separate from logical tenant filtering. The current v810 resource deletion adapter rejects a legitimate null batch response. Record this as a separate product dependency; this feature must expose the failure rather than change production semantics or waive it.

## Technical Context

**Language/Version**: Go 1.26, toolchain go1.26.2; existing Bash and Python guard scripts.

**Primary Dependencies**: Existing Cobra CLI, Go testing/subprocess facilities, repository YAML support; standard library for new coverage policy and runner logic. No new external runtime dependencies.

**Storage**: JSON evidence and Markdown summaries in a unique operator-selected or temporary run directory; checked-in coverage/gap catalogs. Credentials stay in existing local configuration.

**Testing**: Pure reducer/configuration tests without the integration build tag; existing adapter controlled tests; explicitly invoked live tagged CLI tests; isolated generator guards. Documentation-only planning uses link, placeholder, and diff checks only.

**Target Platform**: Existing macOS/Linux local developer environments with Go, Make, and explicitly selected disposable Camunda clusters.

**Project Type**: CLI release-validation tooling, not a new product command.

**Performance Goals**: Build the CLI once per aggregate run, reuse compilation cache, keep existing baseline/volume/real-state timeout controls, and avoid combinatorial flag permutations. Aggregate evaluation is linear in declared scenarios and evidence records.

**Constraints**: No cluster mutation during ordinary unit tests or planning; verify version before mutation; no generated credential configuration; preserve stdout/stderr and existing CLI behavior. No physical infrastructure provisioning. No generated-client hand edits or unrelated production fixes.

**Scale/Scope**: Existing 11 baseline and 11 volume families, seven live real-state families plus gap validation, five common check categories, focused batch and tenant slices. Baseline/volume support 8.8/8.9/8.10; real-state support 8.9/8.10. Initial 8.10 live baseline is 8.10.3.

## Constitution Check

| Principle | Before research | After design |
| --- | --- | --- |
| Operational proof over intent | Pass: no exit-code-only certification | Pass: required evidence levels and post-state checks; gaps remain visible |
| CLI-first, script-safe interfaces | Pass: preserve existing CLI contracts | Pass: test-only controls; existing target names and profile configuration retained |
| Proportionate validation | Pass: planning runs no runtime tests | Pass: pure policy checks first, focused adapters next, live suites only on explicit disposable targets |
| Documentation matches behavior | Pass: integration tooling documentation in scope | Pass: integration README, targets, guards, and separate shell runner aligned; no generated product CLI docs needed unless product metadata changes in separate work |
| Small, compatible changes | Pass: reuse fixture and report helpers | Pass: one shared reducer/catalog; independent work units below; no duplicated version suites |

No constitution violations. The known resource-deletion dependency prevents a clean release certificate until separately resolved, but does not require an unjustified production change in this plan.

## Project Structure

### Documentation (this feature)

```text
specs/338-extend-810-validation/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/requirements.md
└── contracts/validation.md
```

`tasks.md` is intentionally deferred to the tasks phase.

### Source Code (repository root)

```text
Makefile                              # preserve slices; aggregate runner and focused new targets
integration/
├── runner/                           # proposed thin aggregate entry point
├── internal/validation/              # proposed shared catalog, config, evidence reducer + pure tests
├── cli/                              # existing tagged execution, fixtures, record writers
│   ├── harness_test.go
│   ├── config_test.go
│   ├── real_state_*_test.go
│   └── [focused batch/tenant/common coverage files]
├── assets/                           # scenario/gap catalog and tenant-role example
├── scripts/                          # separate shell runner and C810 entry point
└── README.md
api/tests/                            # provenance and isolated generation boundary checks
internal/services/batchoperation/v810/ # controlled compatibility tests only
internal/services/resource/v810/       # controlled contract regression, production fix separate
config/                               # existing base URL normalization coverage
specs/257-c89-real-state-integration/   # preserve history; align version/gap extension references
```

**Structure Decision**: Keep runner mechanics and policy out of product commands, facades, and services. Pure shared policy tests must not import or trigger the tagged CLI TestMain. Centralize target selectors and required scenario identities so runner, focused targets, and validation tests cannot drift. Retain existing report files and add normalized coverage evidence rather than reinterpreting old `outcome` fields incompatibly.

## Phase 0: Research Decisions

Completed in [research.md](research.md). Key decisions cover unique run evidence, strict versus exploratory validation, scenario-level gaps, version-aware fixture selection, batch/draining distinction, tenant routing and authorization, and isolated provenance checks. No unresolved design clarifications remain.

## Phase 1: Design

### Aggregate lifecycle

1. Resolve request and checked-in required coverage before launching subprocesses. Reject invalid release minors, duplicate role assignments, unsafe evidence paths, and incompatible test overrides.
2. Create a fresh run directory and persist its manifest as `running`. Build the CLI once. Run common nonmutating profile/version preflight for every selected target before any mutation; fail the aggregate before destructive slices if any required target fails.
3. Execute existing family selectors sequentially in isolated slice/profile child directories. Paired tenant scenarios receive their explicit role group. Recheck target readiness within destructive slices.
4. Capture process exit and Go test events, and have each scenario write normalized evidence with stable IDs and expected proof levels. Atomic writes avoid treating truncated records as valid.
5. Continue independent checks after ordinary scenario failures where targets remain safe; abort further mutation after readiness loss. On interrupt, stop child work and finalize the available evidence. A hard kill leaves `running`, which offline evaluation treats as incomplete.
6. Reduce against the expected manifest, not only discovered report files. Persist JSON and human summaries. Preserve failures when a later check succeeds; never reuse old invocation evidence.

### Required coverage policy

All seven live families must execute for 8.9/8.10. Required minima include actual fixture creation and observable readback, plus existing supported post-state checks for jobs, incidents, listeners, retention, destructive actions, and cascades. The BPMN-error family must execute its current real-job preview and unchanged-job checks, explicitly labeled with those proof levels; catchable error completion remains a named gap until its setup exists. This does not certify full BPMN-error mutation support.

The checked-in catalog must enumerate actual scenario IDs and proof levels before implementation is declared complete. Test removal, unknown IDs, missing required scenarios, whole-family skips, and duplicate evidence fail catalog validation. Known nonrequired missing proofs may yield `passed_with_gaps`; `coverageComplete` remains false. No runtime wildcard waiver is provided. Required failures, including known product defects, remain failures.

### Tenant scenarios

Use a checked-in example of credential-free role metadata pointing at existing local profiles. Logical roles include A, B, dual-authorized, and A-restricted principals. Physical roles name two provisioned endpoints in one declared cluster and their default-route control. Verify actual logical identity and positive owned data before absence checks. Search-scoped mutations and explicit-key authorization are different scenarios. Missing infrastructure is reported; explicitly required tenant coverage cannot pass without it.

### Supporting contracts

[contracts/validation.md](contracts/validation.md) defines controls, evidence, and exit behavior. [data-model.md](data-model.md) defines records and transitions. [quickstart.md](quickstart.md) describes future verification commands; they have not been executed during planning.

## Delivery Units and Consolidation Review

These are cohesive implementation units for later task generation, not separate tasks for reading, formatting, or bookkeeping:

1. Shared request/catalog/evidence reducer and deterministic policy tests, including missing/duplicate/stale evidence and gap semantics.
2. Aggregate runner, Make integration, common preflight, isolated evidence lifecycle, and fake-subprocess orchestration tests. Keep control changes and their consumers together.
3. Version-neutral real-state selection, cascade fixture identities, scenario evidence integration, and existing gap-artifact alignment with tests.
4. Focused batch/draining compatibility and live post-state scenarios, reusing optional-type coverage; record production dependency without masking failures.
5. Logical/physical tenant role validation, path compatibility checks, scoped and authorization scenarios, and missing-environment evidence.
6. Provenance/boundary guards, separate C810 shell runner, documentation, and final cross-artifact validation.

Six units avoid repetitive story-by-story rewrites. Units 1–2 remain separate because pure policy can be tested independently of subprocess orchestration. Tenant and batch fixtures have distinct prerequisites and failure models. Every unit includes its necessary tests and documentation; do not create separate formatting or progress tasks.

## Validation Strategy and Dependencies

- Planning: inspect links, requirements traceability, placeholders, and whitespace only.
- Implementation: pure policy and runner tests first, then tagged harness checks with fake subprocesses or compile-only checks where suitable. Tagged TestMain currently builds a CLI even for helper tests; avoid calling it unintentionally from ordinary validation.
- Controlled compatibility: target affected batch/resource adapter and URL normalization tests. Existing omitted-operation-type coverage is reused. A regression that exposes the resource adapter defect must remain an explicit failing dependency, not an expected-pass characterization.
- Live: execute requested disposable profiles separately; retain actual server version, all gap rows, and before/after state. 8.10 release proof depends on resolving the resource-deletion defect and supplying required tenant environments.
- Generator reproducibility: disposable checkout only, current pinned tag, before/after older-client fingerprints. No refresh in the developer checkout.
- Broader Go race validation is warranted if shared production code is changed by separately accepted fixes; the harness changes themselves use focused race checks and appropriate build coverage.

## Complexity Tracking

No constitution exceptions. The small integration runner and pure reducer replace implicit Make/Go skip semantics with testable release policy. This complexity is justified by per-profile evidence isolation, failure finalization, and deterministic missing-coverage checks; a shell-only pass/fail wrapper cannot provide those guarantees reliably.
