# Implementation Plan: Complete Smoke-Test Definition Cleanup

**Branch**: `codex/333-fix-smoke-test-cleanup` | **Date**: 2026-10-10 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/333-fix-smoke-test-cleanup/spec.md`, issue #333.

## Summary

Track every distinct process definition returned by the fixture deployment and clean up each eligible exact key after existing instance cleanup. Keep orchestration in the ops service and reuse existing eligibility, deletion, and confirmation helpers. Add per-definition results and an aggregate summary across the domain, public facade, human output, and audit reports. Preserve the existing singular fields as projections of the main fixture.

## Technical Context

**Language/Version**: Go 1.26; toolchain go1.26.2.
**Primary Dependencies**: Existing Cobra, testify, internal service interfaces, toolx helpers; no new dependencies.
**Storage**: In-memory workflow results and existing optional JSON/Markdown audit files; no database or migrations.
**Testing**: Go package tests, stubbed services, command fake-server/subprocess tests, existing terminal-input helpers.
**Target Platform**: Existing supported CLI platforms and configured Camunda compatibility versions.
**Project Type**: CLI with public Go facade.
**Performance Goals**: Linear bookkeeping in returned definitions; current fixtures deploy three. Reuse existing polling limits; no renderer discovery requests or new polling loops.
**Constraints**: Exact-key ownership, tenant isolation, sparse-page safety, truthful confirmation, existing flag/exit/stream behavior. No generated-client edits.
**Scale/Scope**: One smoke-test deployment per invocation; all supported fixture families (currently C88, C89, C810). No cleanup of earlier runs.

## Constitution Check

| Gate | Before research | After design |
| --- | --- | --- |
| Operational proof | Pass: existing wait semantics remain authoritative | Pass: per-key errors prevent false confirmation; no-wait remains explicitly submitted |
| CLI and script safety | Pass: no new flags or command paths | Pass: additive report fields; old fields retain main-fixture meanings; existing exits preserved |
| Proportionate validation | Pass: documentation checks for this phase | Pass: targeted service, facade, and command checks specified for implementation |
| Documentation parity | Pass: user-visible cleanup change identified | Pass: README, command metadata, and regenerated CLI docs included |
| Repository-native scope | Pass: existing ops owner and deletion helpers | Pass: no new service abstraction, dependency, or generated adapter change |

No gate violations or unresolved design questions. Published issue retrieval remains unverified as recorded in the specification; the conversation draft is the planning source.

## Project Structure

### Documentation (this feature)

```text
specs/333-fix-smoke-test-cleanup/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/requirements.md
└── contracts/smoke-test.md
```

`tasks.md` is the next phase and is not generated here.

### Source Code (repository root)

```text
internal/domain/ops_smoke_test_model.go
internal/services/ops/smoke_test_service.go
internal/services/ops/smoke_test_cleanup.go        # focused cleanup orchestration
internal/services/ops/smoke_test_test.go
internal/services/processdefinition/delete.go    # reuse, not planned redesign
c8volt/ops/{model.go,convert.go,client_test.go}
cmd/ops_execute_smoketest.go
cmd/ops_execute_smoketest_progress.go
cmd/cmd_views_ops_execute_smoketest.go
cmd/ops_execute_smoke_test_test.go
README.md
docs/cli/                                       # regenerate through make docs-content
```

**Structure Decision**: Move the cohesive smoke-test cleanup helpers into the focused internal service file as they are extended. Keep command construction, existing progress lifecycle, and final views in their current owners. Facade changes are mechanical mappings only.

## Phase 0: Research Decisions

See [research.md](research.md). Resolve deployment identity completeness, per-definition result attribution, additive report compatibility, no-wait/fail-fast behavior, and truthful aggregate outcomes before implementation.

## Phase 1: Design

See [data-model.md](data-model.md) and [contract](contracts/smoke-test.md).

1. Extract and deduplicate definition evidence from the deployment response in first-seen order. Select the main fixture by its exact BPMN ID. Preserve valid evidence even when some entries are malformed; flag incomplete or conflicting evidence. Do not infer keys by name or replace the main fixture with an arbitrary entry.
2. Preserve existing instance creation, family walking, and instance cleanup. Do not start definition mutations when that cleanup fails.
3. Initialize a result entry for every tracked definition. With no-cleanup, mark all skipped with the explicit reason. Otherwise evaluate each definition against the owned family keys using existing paged eligibility discovery.
4. For each eligible definition, invoke the existing single-key cleanup helper with current options and keep its response and error together. Use deterministic first-seen order; do not introduce a second worker pool. Worker options continue to govern existing nested operations and instance workflows.
5. Safety blockers retain a definition and do not count as errors. Continue on independent errors by default; fail-fast or cancellation stops further scheduling and leaves explicit unattempted entries. Keep completed results unchanged.
6. Aggregate from all entries plus evidence completeness and instance status. Return partial failure on errors, cleanup-skipped on intentional/safety retention, and successful submission with explicit no-wait status when accepted without waiting. Confirmation requires positive evidence for every required resource.
7. Map results through the facade. Render compact aggregate counts by default, per-definition evidence with verbose and audit output, and stable machine fields. Update semantic progress to use one frozen distinct-definition total rather than restarting a 1/1 counter per nested deletion.
8. Update README and command help/no-cleanup metadata to plural scope; regenerate docs. Do not add a new CLI execution mode.

## Validation and Work-Unit Guidance

Implementation must cover complete cleanup, blocked dependency with removable siblings, eligibility errors, accepted-but-unconfirmed deletion errors, fail-fast/cancellation, malformed/duplicate identities, unrelated versions/tenants, no-cleanup, no-wait, dry-run, and supported automation/output modes. Existing sparse-page safety tests remain authoritative and gain a dependency-cleanup consumer case.

Use the commands and scenarios in [quickstart.md](quickstart.md). Changes to domain/public fields, converters, and necessary test doubles belong together to preserve compilation. Service behavior and its fixtures/tests form a cohesive unit; command/report integration and its metadata/docs/tests form another. These are planning boundaries, not standalone context-reading or validation tasks. Do not rebuild a single-item design before adding collections. Respect Ralph story-scope rules when producing tasks; each work unit includes its own validation.

For this planning-only change, validate links, template completion, whitespace, and consistency without runtime tests. For implementation, start targeted. Run the full race suite only if shared runtime changes or unresolved failures justify the broader scope under the constitution.

## Complexity Tracking

No constitution exceptions. A small deterministic loop preserves precise per-definition errors and avoids introducing concurrency or modifying generic deletion contracts.
