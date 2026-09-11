# Implementation Plan: Tenant Context Before Ops Mutations

**Branch**: `295-tenant-context-before-mutations` | **Date**: 2026-09-10 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/295-tenant-context-before-mutations/spec.md`, issue #295.

## Summary

Show effective tenant selection before discovery for all six affected ops commands. Show validated affected tenants before mutation and before an interactive confirmation prompt. Add a synchronous, wording-free tenant-evidence event to existing service progress callbacks so auto-confirm can report this boundary without extra discovery. Render selection and affected context as separate, deduplicated stages on the existing durable stderr channel. Preserve final structured data, audit evidence, operational verification, and mutation behavior.

## Technical Context

**Language/Version**: Go 1.26; repository toolchain Go 1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2, pflag 1.0.10, Viper 1.21.0, testify 1.11.1, and repository progress/tenant helpers. No additions.

**Storage**: Existing optional JSON and Markdown audit files; no new persistence or schema migration.

**Testing**: Go tests, testify, existing command subprocess/fake-server helpers and activity sinks; full race suite via `make test`.

**Target Platform**: Existing supported CLI environments and Camunda configurations; version-neutral ops orchestration with unchanged version-specific adapters.

**Project Type**: CLI and public Go facade library.

**Performance Goals**: Zero additional discovery passes or metadata requests; one tenant-evidence notification per successful service invocation at the established scope boundary; existing tenant summary normalization cost only.

**Constraints**: No mutation/filter/auth changes; no backend mechanics in commands or facades; no tenant chatter in protected modes; preserve error/exit/confirmation policy and audit schemas. Callback completes before mutation starts. No new output mode, flag, or dependency.

**Scale/Scope**: Six commands, shared CLI tenant/progress support, public/internal progress models and conversion, five service workflow files, associated tests, README and generated CLI references.

## Constitution Check

| Principle / gate | Before research | After design |
| --- | --- | --- |
| Operational proof over intent | Pass: tenant visibility does not alter completion verification | Pass: no changes to cancellation, waits, repair outcomes, or success claims |
| CLI-first, script-safe interfaces | Pass: preserve existing mode and prompt policies | Pass: durable stderr stages are mode-gated; structured and audit contracts remain complete |
| Tests and validation mandatory | Pass: targeted tests plus full race suite required for implementation | Pass: service ordering, facade mapping, six command paths, report and mode regression coverage defined |
| Documentation matches behavior | Pass: output change requires source help and README updates | Pass: regenerate references with `make docs-content`; no hand edits to generated docs |
| Small, compatible, repository-native changes | Pass: existing callback and renderer patterns available | Pass: additive event, two-stage CLI state, no new subsystem or dependency |
| Layering and command file cohesion | Pass: backend boundary remains service-owned | Pass: public facade only maps; reporting state stays in focused support files; final formatting remains in views |

No unresolved gate violations or clarification questions. These gates assess the design; implementation tests have not yet been run. Planning and future task generation must preserve `specs/ralph-implementation-rules.md`. Any Ralph run must receive `--implementation-context specs/ralph-implementation-rules.md`.

## Project Structure

### Documentation (this feature)

```text
specs/295-tenant-context-before-mutations/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── tenant-reporting.md
└── tasks.md                    # Future speckit-tasks output
```

### Source Code (repository root)

```text
cmd/
├── cmd_tenant_context.go                 # Shared context and human emission state
├── cmd_views_tenant_context.go           # Shared rendering and final suppression
├── ops_tenant_context.go                 # Ops selection/evidence reporting support
├── ops_progress_mode.go                  # Existing channel policy, reused
├── ops_purge_all_processdefinitions.go
├── ops_purge_orphan_processinstances.go
├── ops_purge_processinstances_with_incidents.go
├── ops_execute_retention_policy.go
├── ops_repair_incident.go
├── ops_repair_processinstance.go
├── ops_purge_all_processdefinitions_progress.go
├── ops_processinstance_purge_progress.go
├── ops_repair_progress.go
├── cmd_views_ops_*.go                    # Preserve final result grammar
└── *_test.go                            # Adjacent command, renderer, audit tests
c8volt/ops/
├── progress_model.go                    # Additive public event/payload
├── convert.go                           # Mechanical callback payload mapping
└── client_test.go
internal/domain/ops_progress.go          # Internal event/payload
internal/services/ops/
├── all_process_definitions_purge.go
├── orphan_purge.go
├── incident_purge.go
├── retention_policy.go
├── repair.go
├── tenant_evidence.go                   # Existing evidence support
└── *_test.go
testx/                                  # Existing test helpers, reused
README.md
docs/cli/                               # Regenerate via make docs-content
```

**Structure Decision**: Extend the existing repository layers. Keep reporting lifecycle helpers in `ops_tenant_context.go` and callback wiring in existing focused progress files. Base command files only select context mode, initialize reporting, and retain ordinary dispatch/prompt logic. Do not place a new reporting lifecycle in all six base command files. No generated clients or version-specific adapters need changes.

## Phase 0: Research Outcome

[research.md](research.md) records verified service boundaries and decisions. Auto-confirm requires a notification from within the service invocation; the existing all-or-nothing rendering marker must become stage-aware for these ops executions. All technical unknowns are resolved from repository evidence.

## Phase 1: Design

### Service evidence notification

Add an optional `TenantScope` payload and `tenant_scope` kind to the existing internal/public progress envelopes. The payload carries a copied `TenantEvidence`; absence differs from a successfully validated empty scope. Reuse evidence already in delete plans or repair frozen sets. Invoke synchronously at each applicable service boundary described in research, before mutation workers or variable updates begin. Nil callbacks remain no-ops. Do not serialize callbacks or add events to final CLI envelopes/reports.

Publish only complete evidence after successful scope construction. Preserve dry-run, no-work, validation, and force-blocking order. Successful previews can report their validated scope while retaining existing no-mutation behavior. In real execution, existing blockers remain authoritative. Service events contain facts only; normalization and labels remain command responsibilities.

Extend both conversion directions and deep-copy tenant slices/targets using existing evidence conversions. No facade loops, planning, polling, or mutation orchestration are introduced.

### CLI reporting lifecycle

Initialize a fresh command-scoped reporting state after local validation, report-path checks, and request-mode resolution, before the first discovery activity wrapper or service call. Derive the base tenant context with existing discovery/explicit-key constructors and print only selection plus applicable override lines on the mode-derived durable stderr channel.

Handle the new tenant event in the existing callback chain before generic progress rendering. Attach full normalized context and emit only affected tenants and unknown warnings. Preserve the existing progress callback and all non-tenant event handling. A successful empty evidence stage is complete without an affected line or unknown warning.

Track `selectionRendered` and `affectedRendered` independently, marking only stages permitted and emitted by the selected output policy. Context data and emission state are separate. The shared final renderer for a staged ops execution emits only an applicable unrendered stage; it cannot treat selection-only output as fully rendered. Preserve the legacy full-context path for unrelated commands, especially smoke-test creation semantics. Reset state per command execution, including tests that reuse Cobra command objects.

Interactive preflight keeps its current planning and frozen-request reuse. The event reports before the prompt; the existing planned-result context attachment can remain as an idempotent fallback. Remove hardcoded human-channel printing. A second event during the execution call must not repeat human context for the same frozen command scope. Keep full final-result/report attachment independent of this suppression.

### Compatibility and failure handling

The intended compatibility change is the timing and single occurrence of tenant lines in permitted human output. CLI commands, flags, aliases, JSON envelopes, audit fields, and backend behavior remain unchanged. The public progress callback receives an additive event kind; existing event kinds retain their meanings, and consumers should ignore unhandled kinds. No new human text is sent to JSON, quiet, automation, or supported keys-only modes.

Selection can remain visible on later discovery failure. Incomplete evidence is never called validated. Preserve existing errors and report-write behavior on failures and declined confirmation. Suppression must not remove warnings from data models or audit serialization. Preserve warning severity exactly once and stable tenant ordering. Do not use string matching to deduplicate lines.

### Validation and delivery sequence

1. Extend progress models/conversion and service evidence emission with ordering and no-extra-request tests.
2. Add stage-aware CLI reporting and wire all six commands; cover early selection, earliest mutation, confirmation, warning uniqueness, and repeated command execution.
3. Complete protected-mode, audit, dry-run, empty-scope, and failure regression coverage; update source help and README; regenerate CLI docs.
4. Run targeted service, facade, and command tests, then `make test` before implementation completion or commit. Apply `gofmt` to touched Go files. See [quickstart.md](quickstart.md) for runnable validation commands.

Acceptance coverage must include each of the six auto-confirm paths and both repair discovery modes. The matrix in [contracts/tenant-reporting.md](contracts/tenant-reporting.md) covers tenant selection, actual evidence, output modes, and lifecycle cases. Use request counters and event ordering to prove zero extra discovery calls, accounting for existing interactive revalidation rather than assuming interactive execution performs only one service call.

Commit messages for subsequent issue-backed work use Conventional Commits and reference #295. This planning command does not implement or commit code.

## Complexity Tracking

No constitution exceptions. The additive service notification is necessary because the CLI cannot observe the pre-mutation boundary of auto-confirm calls; a second discovery pass is explicitly prohibited. Two stage flags are the smallest state needed to distinguish early selection from later affected scope.
