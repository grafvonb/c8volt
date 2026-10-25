# Implementation Plan: Element Scope Variables

**Branch**: `329-element-scope-variables` | **Date**: 2026-10-25 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/329-element-scope-variables/spec.md`, backed by issue #329.

## Summary

Add local variable inspection to `get element/ei --with-vars` and frozen, explicit-scope variable updates to `update element/ei`. Resolve each selected element's owning process separately from its local scope, read all variable pages, and plan the entire update selection before confirmation or mutation. Reuse the exact-local writer and shared request pacing, retries, worker pool, value formatting, envelopes, and prompts. Confirm approved requested values unless no-wait is selected.

Introduce a narrow arbitrary-scope reader underneath existing root-variable APIs, service-owned element enrichment and update orchestration, and thin facade methods. Correct the missing configured-tenant filter in element discovery and retain per-attempt submission evidence across writer retries. Keep version-specific differences in v88/v89/v810 adapters and preserve existing PI root, UT effective-scope, and task-completion contracts.

## Technical Context

**Language/Version**: Go 1.26.0, toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2, Viper 1.21.0, generated Camunda clients, standard JSON/context libraries, `toolx/pool`, shared HTTP/request-rate and retry helpers; no new dependencies.

**Storage**: No local persistence or migration. Frozen plans are in-memory snapshots; Camunda owns runtime variables.

**Testing**: Go tests, testify, generated-client doubles, test-owned HTTP servers, isolated subprocess and real-terminal tests through `testx`. No live integration/smoke targets.

**Target Platform**: Existing Linux, macOS, and Windows CLI/library targets; Camunda adapters 8.8, 8.9, 8.10. Preserve existing configured-8.7 bootstrap failure and exit 1.

**Project Type**: Go CLI and public facade library in the existing single repository.

**Performance Goals**: No extra element discovery for enrichment or rendering; no variable requests for effective keys-only/total inspection; one logical write per changed unique element; no write for unchanged elements. Element limits bound selection only. Complete variable traversal has work proportional to selected scopes and their pages. Execution honors configured worker and invocation-wide request-rate bounds; no new latency SLA is introduced.

**Constraints**: Complete all planning before approval; immutable approved destinations; exact-local confirmation; preserve partial outcomes, error classes, and submission uncertainty. Human diagnostics use configured logging and bounded reasons; prompts use configured stderr; selected-format results use result renderers. Fake-only implementation validation. No generated-client hand edits or generic workflow framework.

**Scale/Scope**: One read enrichment, one new update command, three version adapters, shared variable traversal/writer evidence, element facade composition, command views/tests/docs. Memory holds the selected elements and complete local observations required for a frozen plan. No recursive scope discovery or atomic transaction guarantee.

## Constitution Check

Both reviews use constitution 2.0.0 and current `AGENTS.md`.

| Gate | Before research | After design | Evidence |
|------|-----------------|--------------|----------|
| I. Operational proof | Pass: local read-back required | Pass | Complete scoped confirmation; no-wait explicitly submitted; accepted evidence survives confirmation failure |
| II. CLI/script compatibility | Pass: preserve existing contracts | Pass with documented correction | Existing selection/paging/aliases and envelopes retained; search tenant correction documented below; [CLI contract](contracts/cli.md) |
| III. Proportionate validation | Pass: docs-only now, isolated checks later | Pass | [Quickstart](quickstart.md) targets behavioral risk, real-terminal routing, and shared-runtime race regressions; no live gates |
| IV. Documentation | Pass: user-facing changes identified | Pass | README, command/parent metadata and examples, generated `docs/cli/*` via `make docs-content` in implementation work |
| V. Small repository-native design | Pass: existing layers/helpers | Pass | Shared collector and small evidence accumulator; existing constructors/root/UT semantics retained; no new dependencies/framework |
| AGENTS layering and file cohesion | Pass | Pass | Internal services own loops/pools/wait; facade maps; batch runner separated from command construction and views |
| AGENTS diagnostics/no-op/prompts | Pass: known unsafe precedent excluded | Pass | Single logger ERROR, JSON once, explicit succeeded no-op, complete discovery, actual terminal tests |
| AGENTS implementation boundary | Pass | Pass | No live discovery, provisioning, mutation, or cleanup requirements |

No unresolved gate or clarification remains. These are design gates, not claims that implementation tests have passed.

## Project Structure

### Documentation (this feature)

```text
specs/329-element-scope-variables/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── cli.md
│   └── facade-service.md
└── checklists/requirements.md
```

`tasks.md` is a later `$speckit-tasks` output and is not created by this command.

### Source Code (repository root)

The following is the intended edit map; new files are design targets, not implemented files.

```text
cmd/
├── get_element.go, get_element_search.go
├── update.go, update_element.go                  # registration, construction, validation
├── update_element_variables.go                  # batch execution/approval lifecycle
├── cmd_views_element.go
├── cmd_views_element_update.go                  # preview/results and bounded failures
├── update_variables_payload.go                  # reuse established parser
└── adjacent *_test.go                           # execution, output, terminal contracts
c8volt/
├── client.go                                    # shared dependency composition
└── element/
    ├── api.go, client.go, model.go, convert.go
    ├── enrichment.go, variable_updates.go
    └── adjacent *_test.go
internal/
├── domain/
│   ├── element.go, element_variable.go          # query, plans, outcomes
│   └── usertask_update.go                       # additive internal scope-writer evidence
└── services/
    ├── element/
    │   ├── enrichment_variables.go
    │   ├── update.go, update_validation.go, update_wait.go
    │   ├── v88/service.go, v89/service.go, v810/service.go
    │   └── adjacent *_test.go
    ├── variable/
    │   ├── api.go, search/search.go
    │   ├── v88/, v89/, v810/                    # APIs, filters, raw decoders, writer evidence
    │   └── adjacent *_test.go
    └── common/                                 # focused shared evidence helper if needed
README.md
docs/cli/                                       # regenerated from command metadata
```

Reuse `toolx`, `toolx/pool`, `testx`, `internal/services/httpc` attempt observations, and existing common helpers before adding another helper. Keep existing PI/UT facade/public JSON shapes intact. Interface additions and all affected adapters/test doubles belong in the same compiling unit.

**Structure Decision**: Extend existing packages; element orchestration depends on narrow element/variable interfaces. Version adapters must not import their parent element orchestration package. A new facade composition constructor preserves `New` and `NewWithListeners`; old constructors continue to support old operations and report an ordinary precondition error for unwired new operations.

## Phase 0: Resolved Research

[Research](research.md) records nine decisions and alternatives. The material findings are:

1. Root-variable filters cannot be reused by substituting an element key; owner and scope must be independent.
2. Paged element aggregation copies structs before the visitor; the composed service must retain enriched items in the final aggregate.
3. Element discovery currently omits configured tenant filters across all adapters. Apply the existing generated `TenantId` field server-side. This intentionally narrows configured-tenant searches and totals to their intended tenant; explicit-key lookup continues to bypass the discovery filter. Document this correction in release-facing guidance.
4. Retry cancellation and local pacing can erase prior write evidence. Preserve it outside the retry callback, including client-timeout substitution, without changing retry policy or existing PI/UT public schemas.
5. PI/UT raw-error human renderers are not a compliant template. Use configured logging and bounded aggregate reasons, preserving machine evidence separately.

## Phase 1: Design

[Data model](data-model.md) defines selected identities, observations, frozen classifications, submission evidence, and state transitions. [Facade/service contract](contracts/facade-service.md) defines additive interfaces and adapter boundaries. [CLI contract](contracts/cli.md) defines mode behavior, JSON payloads, rendering ownership, and error flow.

The execution sequence is local validation → all-target planning → dry-run/no-op handling → confirmation where eligible → validated snapshot execution → local verification unless no-wait → one mode-aware result path and original exit classification. An empty object still completes authoritative planning. No renderer makes service requests.

Variable-only search retains existing paged continuation policy. Combined listener/variable search retains collected listener policy. Keys-only/total suppress variable enrichment before dispatch while retaining existing conflicts. Every selected element remains visible after an empty local read.

## Delivery and Task-Generation Guidance

Use independently verifiable work units, with necessary implementation, tests, formatting, and relevant checks together:

1. Shared scoped reading and root compatibility across all adapters, including every affected interface/test double.
2. Element discovery tenant correction and enrichment composition, including facade wiring and enriched paged aggregates.
3. Frozen local update planning, execution, exact-local wait, and retry submission evidence with compatible PI/UT mappings.
4. Read CLI integration and output combinations, with adjacent command metadata/docs and execution-path tests.
5. Update CLI lifecycle, modes, diagnostic routing, real-terminal interactions, and user docs, with its fixtures/tests in the same unit.

This is sequencing guidance, not a generated task list. The shared writer evidence and update service may be split only if each part compiles and has independent behavioral coverage. Group interface changes with their consumers. User stories define acceptance coverage, not a reason to rebuild shared behavior. Avoid standalone context-reading, fixture-only, or bookkeeping tasks. Review tightly coupled neighbors for consolidation; explain necessary separations. More than ten tasks requires a consolidation review.

Implementation validation follows [quickstart.md](quickstart.md). Run the full race suite after integrating shared traversal, writer evidence, and concurrency changes because they affect multiple packages. Do not rerun it merely to commit or after prose-only edits. Current planning validation is limited to changed-document review, local links, placeholders, and whitespace.

## Complexity Tracking

No constitution exception is required. The callback collector shares existing paging behavior; the evidence accumulator is restricted to scope-variable writes; the enrichment option struct selects two existing capabilities. None introduces a general workflow engine. Main compatibility risks are root pagination drift, lost enriched page results, configured-tenant search narrowing, and incorrect retry acceptance classification; each has explicit isolated regression coverage.
