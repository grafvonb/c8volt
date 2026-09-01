# Implementation Plan: Semantic Progress Milestones for Long-Running Commands

**Branch**: `285-semantic-progress-milestones` | **Date**: 2026-08-31 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification for [GitHub issue #285](https://github.com/grafvonb/c8volt/issues/285)

## Summary

Add exact, operator-facing completion milestones to long-running commands by extending the existing typed progress callback with one wording-free completion fact and routing it through one concurrency-safe command reporter. The reporter owns the stable workflow activity, a completion-driven 10-second durable cadence, immediate failures, verbose per-item detail, final flushing, and existing output-mode suppression. Services emit facts at real worker or stage completion boundaries; commands retain all human wording. No CLI flags, worker behavior, backend requests, result schemas, or mutation safety rules change.

## Technical Context

**Language/Version**: Go 1.26 (`toolchain go1.26.2`)

**Primary Dependencies**: Cobra/pflag, standard-library `context`, `sync`, `time`, and `log/slog`; existing `toolx/logging`, `toolx/pool`, `internal/services`, `c8volt/foptions`, and `c8volt/ops` progress abstractions

**Storage**: N/A; progress is in-memory command state and existing audit/result persistence is unchanged

**Testing**: Go `testing` with Testify; `testx/activitysink`, fake HTTP services, and injected clocks; targeted package tests with `-race`, then `make test`

**Target Platform**: Cross-platform c8volt CLI against supported Camunda 8.7, 8.8, 8.9, and 8.10 clusters

**Project Type**: Go CLI with public facades and version-neutral/versioned internal service layers

**Performance Goals**: O(1) work and bounded state per completion; no additional Camunda requests; transient activity updates on every completion; at most six default informational milestones per minute per active scope

**Constraints**: Completion-driven 10-second cadence; no timer goroutine; exact monotonic totals under concurrent callbacks; no stdout progress; clean sub-10-second runs stay durably silent; quiet retains only failure warnings; automation, JSON, and keys-only suppress all human progress

**Scale/Scope**: One shared reporter reused by process-definition deletion/purge/deployment, process-instance cancel/delete, retention/orphan/incident purge, repair, smoke-test, bulk start, slow analysis, and justified multi-target waits; command-family adapters remain small and independently testable

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

| Principle | Pre-design evidence | Post-design evidence | Result |
| --- | --- | --- | --- |
| I. Operational Proof Over Intent | The spec distinguishes submitted, confirmed, and failed outcomes and preserves existing waits. | Completion facts are emitted only at the owning service's actual acceptance/confirmation boundary; command wording cannot promote submitted work to confirmed. | PASS |
| II. CLI-First, Script-Safe Interfaces | JSON, keys-only, quiet, automation, confirmation, dry-run, and exit contracts are explicit requirements. | The output-policy contract keeps progress on activity-aware stderr and leaves machine stdout/result envelopes byte-parseable. | PASS |
| III. Tests and Validation Are Mandatory | Concurrency, pacing, failures, activity arbitration, lifecycle wording, and suppression all have acceptance criteria. | Design includes fake-clock, race, service-boundary, command-mode, documentation, and full `make test` validation. | PASS |
| IV. Documentation Matches User Behavior | The feature changes visible progress behavior. | Command help/README source is updated and generated CLI docs are refreshed with `make docs-content`; generated docs are not hand-edited. | PASS |
| V. Small, Compatible, Repository-Native Changes | Existing progress events, activity priority, pacing, and output gates were assessed first. | The design adds one event fact and one focused reporter, reusing existing callbacks and worker seams without a new subsystem or dependency. | PASS |

No constitution violations or complexity exceptions are required.

## Project Structure

### Documentation (this feature)

```text
specs/285-semantic-progress-milestones/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── semantic-progress-contract.md
└── tasks.md                         # generated later by $speckit-tasks
```

### Source Code (repository root)

```text
internal/domain/ops_progress.go      # canonical wording-free progress facts
internal/services/calloption.go      # callback propagation
internal/services/
├── processinstance/                 # bulk start/cancel/delete/wait completions
├── processdefinition/               # basic and purge deletion completions
├── resource/                        # deployment visibility completions
├── incident/                        # incident work completions
└── ops/                             # purge, repair, analysis, and smoke stages

c8volt/foptions/options.go           # public process/resource callback mirror
c8volt/ops/                          # public ops callback mirror and conversion

cmd/
├── ops_progress_mode.go             # output policy
├── ops_progress_milestones.go       # 10-second completion-driven pacing
├── ops_semantic_progress.go         # focused reporter/aggregate lifecycle
├── ops_progress_render.go           # compact command-owned wording
├── processinstance_mutation_progress.go
├── ops_processinstance_purge_progress.go
├── ops_repair_progress.go
├── ops_explicit_large_work_progress.go
├── cancel_processinstance*.go       # direct/stdin/search mutation wiring
├── delete_processinstance*.go       # direct/stdin/search mutation wiring
├── delete_processdefinition.go      # basic definition deletion scope
├── deploy_processdefinition.go      # deployment/start phase scopes
└── ops_*purge*, ops_*repair*, and ops_execute_smoketest.go

toolx/logging/activity.go            # reused activity arbitration and safe writes
testx/activitysink/                  # reused activity test capture
README.md and command metadata       # operator documentation sources
docs/cli/                            # regenerated with make docs-content
```

**Structure Decision**: Keep service-owned facts in the existing canonical progress envelope, facade conversion mechanical, and all wording/output decisions in `cmd`. Add the stateful reporter as a focused command file because timing, aggregation, activity ownership, and finish semantics form one distinct lifecycle. Do not add worker orchestration to commands or facades.

## Design and Delivery Strategy

1. Extend the canonical progress event and both public mirrors with a completion fact carrying scope metadata, identity, disposition, optional trustworthy affected delta, and failure detail. Keep strings descriptive facts, not rendered sentences.
2. Implement one mutex-protected semantic reporter with an injected clock, explicit workflow activity start/stop, output policy, monotonic aggregate, affected-count validity, immediate failure path, and idempotent finish.
3. Emit one completion fact at each service worker/stage return. Fail-fast work that was never scheduled emits nothing. Preserve existing pool ordering and suppress legacy timer progress whenever the structured callback is active.
4. Wire the same reporter into direct, stdin, and search paths. End planning activity before prompts and begin a fresh workflow scope only after confirmation. Keep nested HTTP/wait/batch activity below workflow priority.
5. Deliver in operator-verifiable slices: shared contract/reporter; process-instance cancel/delete; process-definition delete/purge/deploy; ops cleanup/repair/smoke; assessed secondary workflows; documentation and broad regression validation.

## Risk Controls

- Treat affected counts as all-or-nothing for a scope. A producer must declare complete coverage and provide a trustworthy value, including explicit zero, for every completion; otherwise the reporter never renders the aggregate.
- Do not infer success from elapsed time, response slice length, or final rendering. The service completion boundary determines submitted, confirmed, or failed disposition.
- Keep discovery page events separate from mutation completion. Plain page advancement is never labeled as completed mutation work.
- Use no timer or background goroutine in the reporter. The next real completion checks elapsed time, which prevents idle output and simplifies shutdown.
- Preserve legacy progress only for callers without the structured callback to prevent duplicate human streams.

## Complexity Tracking

No constitution violations require justification.
