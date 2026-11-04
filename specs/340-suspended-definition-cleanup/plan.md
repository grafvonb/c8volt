# Implementation Plan: Include Suspended Instances in Definition Cleanup

**Branch**: `340-suspended-definition-cleanup` | **Date**: 2026-10-23 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/340-suspended-definition-cleanup/spec.md`

**Status**: Phase 0 research and Phase 1 design complete. No implementation or runtime tests performed; no tasks generated.

## Summary

Extend existing definition deletion and purge from ACTIVE-only workload to ACTIVE plus SUSPENDED on Camunda 8.10. Add separate suspended statistics without changing active fields. Observe unfinished counts with one combined-state predicate, and use the same predicate for force discovery. Feed the resulting keys into current ancestry/root planning, deduplication, cancellation, drain verification, instance-history deletion, and definition deletion.

Preserve existing instance cancellation behavior, engine cascading, force rules, deletion order, opt-outs, capability restrictions, and completion handling. Do not add a new mutation engine or lifecycle model. General DRAINING display belongs to separate work.

## Technical Context

**Language/Version**: Go 1.26, toolchain go1.26.2 from the existing go.mod; no upgrade.

**Primary Dependencies**: Existing Cobra/pflag, domain/service/facade architecture, generated Camunda clients, and current polling/traversal helpers. No added dependencies or regenerated clients.

**Storage**: No persistent storage change. Existing result/report structures receive additive fields.

**Testing**: Existing Go unit tests, generated-client mocks, test-owned HTTP servers, isolated command/subprocess and real-terminal tests. No live build tags or integration targets.

**Target Platform**: Existing CLI/library platforms. Suspended cleanup applies to 8.10; 8.8/8.9 retain their existing behavior, including full-history deletion restrictions.

**Project Type**: CLI and public Go library.

**Performance Goals**: Preserve workers, batching, wait budgets, and no-stat/no-state-check read boundaries. v810 statistics add a suspended count and combined unfinished count, with existing exact-total optimization and capped-total paging. No caching or performance redesign.

**Constraints**: Existing cancellation and history workflows remain authoritative. Keep active fields active-only; do not infer authoritative zero by adding separate observations. Follow repository output/logging rules. Planning only in this invocation.

**Scale/Scope**: Definition statistics and their watch projection; definition-delete and purge preconditions, instance discovery, mandatory drain checks, and related reports/docs. Existing scopes can span multiple pages and overlapping trees; no new scalability or transaction guarantee.

## Constitution Check

| Principle | Before research | After design |
| --- | --- | --- |
| Operational proof | Required checks must recognize suspended work | Combined-state evidence drives gates; failed/unconfirmed prerequisites block dependent steps; current final confirmation is preserved |
| CLI-first/script-safe | Preserve commands, flags and active-field semantics | Additive fields and compact suspended statistics; existing envelopes, exit classes, prompts and opt-outs retained |
| Proportionate validation | Planning is documentation-only | Document checks now; targeted isolated regressions during implementation; no automatic live/full-suite gate |
| Documentation matches behavior | Active-only help/report wording becomes inaccurate | Update source metadata, README and generated CLI docs with the behavior change |
| Small compatible changes | Reuse existing service owners | Internal union selector and count, additive projections, current planner/cancellation; no new engine or public mutation API |

All gates pass before research and after design. No constitutional exception is requested. This is a design review, not evidence of passing implementation tests.

## Project Structure

### Documentation (this feature)

```text
specs/340-suspended-definition-cleanup/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/requirements.md
└── contracts/definition-cleanup.md
```

`tasks.md` is a later `$speckit-tasks` deliverable and is not created here.

### Source Code (repository root)

| Owner | Planned responsibility |
| --- | --- |
| `internal/domain/processdefinition.go` | Separate suspended statistics/support and internal authoritative unfinished observation |
| `internal/domain/processinstance.go` | Internal plural-state selector, preserving public filter shape |
| `internal/services/processdefinition/v810/service.go` | Suspended and union count requests using existing generated filter and count helpers |
| `internal/services/processinstance/search.go`, `v810/service.go`, `v810/convert.go`, older adapter validation | Validate singular/plural selection; v810 union mapping; prevent unsupported queries; preserve scalar behavior |
| `internal/services/processdefinition/delete.go` | Unfinished plan/check/discovery/drain behavior with unchanged cancellation and history calls |
| `internal/domain/resource_workflow.go`, `internal/domain/ops_all_process_definitions_purge.go` | Additive cleanup counts/keys and aggregation data |
| `internal/services/ops/all_process_definitions_purge.go` | Unfinished preconditions and affected-count fallback; retain existing candidate discovery and execution controls |
| `c8volt/process`, `c8volt/resource`, `c8volt/ops` models/converters | Mechanical additive projections, including duplicated ops mappings and existing plan totals |
| `cmd/cmd_views_processdefinition.go` | Compact suspended statistics and watch comparison/refresh projection |
| `cmd/delete_processdefinition.go`, relevant views, `cmd/ops_purge_all_processdefinitions.go` | Existing command dispatch/preflight with corrected count use and metadata |
| `cmd/cmd_views_ops_purge_all_processdefinitions.go`, `cmd/ops_report_purge_all_processdefinitions.go`, `cmd/ops_purge_all_processdefinitions_progress.go` | Correct blockers, fallback counts/keys, reports and aggregate progress wording |
| Adjacent unit tests, `README.md`, generated `docs/cli/*` and index | Focused evidence and aligned documentation |

**Structure Decision**: Keep mechanics in services, conversions in facades, presentation in views and mode-specific progress/watch files. Do not add a new package, public workflow interface, or generic reconciliation layer. Paths identify owners, not a requirement to edit every file. Keep generated clients/provenance and integration harness files untouched.

## Phase 0: Research Decisions

[research.md](research.md) resolves the technical unknowns from source inspection:

1. Separate active/suspended statistics cannot safely be summed as a destructive zero check during transitions.
2. The generated v810 advanced state filter already supports a combined predicate; a narrow internal plural selector is sufficient for discovery.
3. The current definition statistics owner can acquire an internal combined count without changing service signatures or opt-out ownership.
4. Existing cancellation, shared bulk root scope, and dependent-step ordering are reused.
5. Existing incident-support conventions supply the suspended availability/zero representation, and the watch projection must track both value and support.
6. Only completeness behavior affecting the new union searches is in scope; unrelated pagination and mutation hardening remain separate.

No external dependency choice, unresolved clarification, or new live prerequisite remains.

## Phase 1: Design

### Statistics and authoritative unfinished observations

Add `Suspended` and `SuspendedCountSupported` following existing incident conventions. Keep all active values unchanged. Add internal nullable `Unfinished` (`json:"-"`) to distinguish an authoritative zero from missing combined evidence. The v810 statistics acquisition performs separate display counts plus a combined ACTIVE/SUSPENDED count. Required query failure returns an error and cannot publish a partially successful statistics object.

`getProcessDefinitionDeletePlanBase` maps the combined observation into an additive unfinished plan count. On a supported result, nil is an error. Older versions retain active-only acquisition and use that active count for cleanup. Do not issue unsupported suspended queries, add a service method, or expose the internal count on ordinary public statistics. Plan/report unfinished fields are separately defined in [data-model.md](data-model.md).

This keeps the existing `WithStat` and skipped-state-check boundaries. It intentionally costs two extra v810 count queries per full statistics acquisition; exact totals avoid enumerating instances when available. No-stat reads and skipped previews remain unchanged.

### Combined discovery and completeness

Add internal `States` to the instance filter. Validate every member, reject simultaneous singular/plural selectors, normalize/deduplicate explicit values, and map the v810 union to the existing advanced filter. Update shared search and adapter validation together; keep public filter/CLI unchanged and older scalar queries intact. Older adapters must reject unsupported plural input rather than silently ignore it.

Definition force discovery chooses the union through existing side-effect-free state capability validation. Only an unsupported-capability result selects the established older-version ACTIVE path; other errors propagate. Use combined unfinished count rather than active count for the initial impact gate, and union keys for subsequent root planning. Preserve active keys separately according to their observed state.

Keep page advancement under services and retain existing tenant/options. Continue sparse pages when continuation is authoritative and advances. Fail closed on stalled or indeterminate enumeration rather than report complete cleanup. In the affected v810 page normalization/count path, distinguish capped totals from continuation and do not rely on an end cursor alone as proof of more data. Preserve genuine final empty pages and offset fallback. Add isolated request/metadata tests before widening any existing helper behavior; no general pagination rewrite.

A combined predicate protects membership across ACTIVE/SUSPENDED transitions. It cannot freeze a backend snapshot or prevent starts/completion during paging. Those limits remain documented, not solved with new cancellation expansion or retry protocols.

### Existing cleanup and purge flow

Change enabled safety decisions, required drain checks, `RequiresForce`, affected-count fallback, and relevant helper names/text from active-only to unfinished. Preserve the meaning of existing active accessors and fields; add separate unfinished accessors and projections.

Continue using `DryRunCancelOrDeletePlan`, existing unique-root scope, `CancelProcessInstances`, mandatory drain verification, `DeleteProcessInstances`, and existing resource deletion/confirmation. Do not move descendant-first or conflict-recovery logic, add child cancellations, resume suspended instances, or change instance behavior to meet this plan.

Preserve current per-item versus whole-plan preflight boundaries. No new transactional guarantee is introduced. No-wait still skips only supported final confirmations; it does not skip a drain prerequisite of a following destructive step. No-state-check retains its current bypass without added reads or implicit cancellation. Purge gains no new bypass flag. Failed or persistently unfinished observations stop dependent work through existing bounded failure handling; newly observed work is not silently added to cancellation.

### Output, reports and documentation

Map new fields mechanically through process, resource, and ops models/converters, including duplicated conversions. Preserve active fields and report schema version. Update plan totals, affected/blocked key fallback, Markdown reports, and JSON reports together so suspended work is not lost at a projection boundary.

Statistics show compact `su:` only when supported, using the established dash for zero. Update watch display identity, changed-field calculation, formatting, and support transitions together; do not add watch lifecycle/timing behavior. Default non-stat output remains unchanged. General DRAINING lifecycle representation is excluded.

Correct cleanup wording to unfinished with active/suspended detail when useful. Keep result rendering in view helpers and diagnostics on the configured logger. Apply the existing diagnostic, empty-result and terminal-prompt requirements to affected execution paths; do not turn a wording fix into a new output framework. Update command metadata and README, then regenerate CLI documentation with `make docs-content` during implementation.

## Validation Strategy

| Evidence group | Requirements | Expected result |
| --- | --- | --- |
| Stats union request, separate buckets, support/zero/errors, capped counts | FR-001–FR-003, FR-010 | Correct active/suspended display; authoritative union gate; older requests unchanged |
| Internal plural selector, ambiguity, capability and request mapping | FR-004, FR-010 | One combined predicate on v810; no silent broadening or unsupported request |
| Suspended-only/mixed force discovery, paging, roots and overlap | FR-004–FR-005 | Complete union scope through existing planner and one root dispatch in non-retried overlap cases |
| Remaining suspended work, transition, late work, failures, no-wait | FR-006–FR-008 | No false drain completion or dependent deletion after a failed prerequisite |
| Normal no-force, dry-run, bypass, empty/terminal-only | FR-003, FR-008–FR-009 | Correct existing eligibility, no extra skipped-mode reads, zero prohibited mutations |
| Facade/report mapping, human/JSON/keys/quiet, watch refresh | FR-002, FR-009 | Active semantics preserved; additive fields and accurate non-stale suspended output |
| Nullable deletion, existing batch/final confirmation, compatibility | FR-010–FR-011 | No lifecycle redesign or changed capability restrictions |
| Source help, README, generated references | FR-012 | Accurate unfinished-work and control descriptions |

Use existing fake-backed fixtures and tests; extend only missing scenarios. [quickstart.md](quickstart.md) gives safe targeted commands and concrete existing test anchors. Inspect targets and inherited environment before execution. No live tests, smoke workflows, profiles, deployments, or cluster probes are allowed as implementation validation.

Trace changed error paths from service through facade, command rendering/logger and exit. Use separate stdout/stderr, configured/inherited logger destination and severity, repeated distinctive HTTP errors, compact normal/verbose/quiet errors, one JSON envelope with EOF and preserved partial results/exit, redacted DEBUG via both flag and config, and plain/structured log formats. Reuse existing isolated diagnostic coverage; assertions must fail if logging is bypassed or raw transport details escape.

Where empty/prompt behavior is touched, verify real terminal stdin, separate streams, exact human text, zero-byte keys, one JSON result, no empty-scope prompt/mutation, and supported auto-confirm/automation/no-wait combinations. Preserve nonempty, abort, error and sparse-page regressions.

After relevant checks pass, broaden only for an actual shared-behavior risk or unresolved failure. `make test` is not a planning or automatic commit gate. Record the real reason if broader race coverage is selected. No runtime check has been performed in this planning step.

## Work-Unit Guidance for Task Generation

Keep each unit compiling, with related models/conversions, test doubles, necessary tests, formatting and targeted validation together. A practical grouping is:

1. **Statistics and presentation**: suspended support and internal union counts, affected converters, static/watch projection, query/completeness regressions and related docs. This independently makes paused workload visible without changing cancellation.
2. **Definition cleanup**: internal union selector/adapter validation and discovery, additive plan fields, existing preconditions/drain logic, facade projections and delete command output. Keep these coupled changes together so collection and single-item paths do not get different eligibility rules.
3. **Purge integration**: aggregate unfinished gates/counts/keys, report/progress/help projections and cross-layer isolated acceptance scenarios using the shared definition implementation.

These groups are guidance, not generated tasks. The first has an independently observable statistics result; the second owns shared cleanup; the third adapts the separately owned purge result/report flow. Consolidate adjacent groups if implementation is small, or split only at compiling independently verified boundaries. Do not split interface/model changes away from implementations and test doubles. Do not create standalone context-reading, progress-tracking, or live-validation tasks.

## Complexity Tracking

No constitutional violation or new architectural layer. Two focused internal additions are justified: a plural state selector because the backend already supports a union, and a nullable authoritative unfinished observation because separate bucket reads can produce a false zero. Public additions are limited to suspended statistics and cleanup impact. Simpler arithmetic was rejected for correctness; a new service API or mutation engine was rejected as unnecessary.

## Coverage Limitations

Isolated tests can prove c8volt request construction, scope handling, sequencing, rendering and failure behavior. They do not prove real engine cascading, exporter visibility, listener timing, or snapshot consistency. Existing #339/#341 completion is baseline context, not newly executed evidence. #342 remains separate real-environment validation only when explicitly requested for a target.
