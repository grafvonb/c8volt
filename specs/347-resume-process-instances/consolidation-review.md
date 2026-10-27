# Task Consolidation Review: Resume Process Instances

**Date**: 2026-10-27
**Feature**: #347 / `347-resume-process-instances`
**Artifacts**: [tasks.md](tasks.md), [spec.md](spec.md), [plan.md](plan.md), [data-model.md](data-model.md), [research.md](research.md), [CLI contract](contracts/cli.md), [service contract](contracts/service-api.md), [quickstart.md](quickstart.md).
**Outcome**: Consolidated a fresh draft from **7 to 5 tasks**, preserving all requirements and meaningful validation. No completed/started tasks or progress records existed. No historical consolidation section needed relocation.

## Evidence inspected

- `internal/services/processinstance/v810/suspend.go`: `SuspendProcessInstance` composes local validation, submission, and waiter observation with one result type; `suspensionFailure` retains evidence while classifying errors.
- `internal/services/processinstance/v810/suspension_submission.go`: `submitProcessInstanceSuspension` captures cumulative evidence outside `RetryCamundaMutation`; request-rate admission differs from dispatch; strict empty-204 validation and explicit rejection classification are substantial protocol obligations.
- `internal/services/processinstance/waiter/suspension.go` and `suspension_test.go`: a bounded, read-only, identity-aware state loop with last-observation/date replacement, terminal handling, read errors, context, and attempt limits. Resumption changes the success predicate without introducing another execution engine.
- `internal/services/processinstance/suspend.go` and `suspend_test.go`: `SuspendProcessInstances` normalizes/prevalidates all keys, initializes ordered unattempted slots, uses the shared pool, and retains partial outcomes under fail-fast/cancellation.
- `c8volt/process/suspend.go`, `suspend_test.go`: thin facade mapping and factory guards; no justification for a separate mapping-only task.
- `internal/services/processinstance/suspend_plan.go`: `PlanProcessInstanceSuspension` and `validateSuspensionTraversal` handle full scope validity, first-seen overlap handling, and classifications. Resumption uses different eligibility without new traversal.
- `c8volt/process/suspend_plan_test.go`: plan mapping/errors/factory guards/scope tests keep public-plan conversion with service planning.
- `cmd/suspend_processinstance_tree.go`: frozen clone, successful-empty early return, one approval, common collection call, activity cleanup, one-result/error-exit flow.
- `cmd/suspend_processinstance_logging_test.go`, `suspend_processinstance_terminal_test.go`, `suspend_processinstance_tree_terminal_test.go`: configured diagnostic/INFO contracts and actual terminal input require command execution; renderer-only tests cannot replace them.
- `integration/cli/suspended_fixture_test.go` and `get_processinstance_suspended_test.go`: the existing explicit resumption scenario is a direct transition followed by fresh reads; changing intended CLI coverage is a bounded source adjustment, with ownership and cleanup evidence preserved and no live validation.
- Repository instructions, constitution v2.0.0, the design artifacts, generated task template, hook configuration, and current worktree/progress state.

## Merges applied

### Original T001 + T002 → final T001: complete versioned lifecycle

The original waiter/model task and versioned-operation task shared the result contract and immediately handed state/timeout semantics into `SuspendProcessInstance`-style composition. Combining them removes a temporary prerequisite handoff and validates acceptance-to-observation evidence as one coherent operation. The existing waiter is a compact read-only loop; resumption uses the same timing/read primitives and differs chiefly in ACTIVE/COMPLETED versus cancellation handling. Its fixtures and protocol tests stay separate in code but belong to one completion criterion.

The combined unit still includes all domain field/enum/omission constraints, adapter contracts, all affected test doubles, capability/local guards, narrow normalization reuse, strict response/ambiguity/rate behavior, observation cases, formatting, and targeted validation. It does not add workers, public collection orchestration, CLI rendering, or tree discovery. This bounded scope fits a focused operation iteration; the architecture boundaries in code remain intact.

### Original T004 + T007 → final T003: exact-key command and intended fixture coverage

The integration task changed how an existing resumption scenario invokes the command and how its fallback guidance describes that capability. It introduced no independent production workflow. Fold it into the command unit, which already owns source help, README, generated docs, and fake-backed fresh-read/date coverage. This removes a documentation/fixture handoff and ensures intended CLI usage is aligned when the command lands.

Preserve source-only review for integration-tagged files, owned-key/cleanup tracking, independent observation evidence, and existing date-omission checks. No live execution is introduced or claimed. Remove the former `[P]` marker because its content is now part of the command unit.

## Adjacent-boundary evaluation

Original draft boundaries are recorded here to make the complete review auditable; only final IDs appear in task instructions.

| Original boundary | Decision / final ownership | Concrete reason |
| --- | --- | --- |
| T001 / T002 | Merge → final T001 | One adapter composes submission and a compact read-only waiter over the same evidence model; removes immediate state-contract handoff without adding collection/CLI complexity |
| T002 / T003 | Retain → final T001 / T002 | Protocol admission/retry ambiguity and bounded per-target state observation already form a substantial matrix. Adding pool scheduling, all-key prevalidation, stable ordering, fail-fast/in-flight/unattempted evidence, public factory mapping and all associated tests would combine two distinct state machines. Both can finish with useful isolated behavioral proof |
| T003 / T004 | Retain → final T002 / T003 | Collection/facade delivers reusable library behavior; CLI adds real-terminal confirmation, configured logger plain/structured/level routing, both DEBUG paths, writer-failure/exit precedence, activity lifecycle, and docs. These are substantial runtime contracts, not thin wiring |
| T004 / T005 | Retain → final T003 / T004 | The exact-key command is a useful complete MVP. Tree scope adds completeness/identity validation and traversal-before-filter semantics; it does not belong in command plumbing or need to delay exact-key delivery |
| T005 / T006 | Retain → final T004 / T005 | Whole-scope planning has nested/overlap/partial/malformed/read-permission failure semantics. Tree CLI additionally freezes approval, validates post-plan races, and proves no-prompt empty scopes with real PTY and every supported output combination. Merging both would add an extensive second interaction matrix to the discovery unit; keep each implemented and validated together |
| T006 / T007 | Do not merge adjacent pair; move T007 to original T004 → final T003 | Fixture resumption scenario depends on exact-key command, not tree discovery. Combining it with tree work would create an artificial dependency and a later docs/fixture repair |

## Nonadjacent sharing and remaining combinations

- Original T004/T007 merged as above; all implementation-owned documentation and fixtures stay with delivery.
- Final T002 includes the collection service plus its thin singular/collection facade, public models/converters, required stubs, and tests. There is no separate facade or single-item workflow to consolidate.
- Final T004 includes the planner plus thin facade and all plan models/validation; a mapping-only task would not be justified.
- Final T003/T005 share views/options/test infrastructure but not a second executor. Keep separate because exact-key delivery is independently useful and tree adds frozen approval, planning failure, and prompt-free empty-scope behavior. Reuse fixtures and renderer primitives without duplicating full cross-products or postponing logger compliance.
- Final T001/T004 share normalization or structural concepts only. Narrow helper reuse occurs where behavior is identical; do not turn this into a generic mutation framework or merge protocol/confirmation with hierarchy planning.
- US3, US4, and US5 are acceptance dimensions already owned by the backend/command units. Their phases explicitly map ownership; adding standalone state hardening, output repair, or docs tasks would repeat implementation or delay required behavior. Counts by primary ownership are two foundational tasks, one US1 task, and two US2 tasks; US3/US4/US5 have zero separate checkboxes, not zero coverage.
- No `[P]` remains: T003/T004 may affect overlapping public interfaces and command stubs, while all other task dependencies are explicit. No artificial work unit is created to manufacture concurrency.
- The command tasks are intentionally substantial because logger/PTY/envelope/exit/activity tests prove one delivered interaction contract. Reuse the existing isolated fixtures and avoid redundant state×format cross-products; do not split required checks into a later “hardening” task merely to shrink the checkbox.

## ID mapping

| Original ID | Final ID | Disposition |
| --- | --- | --- |
| T001 | T001 | Merged evidence/waiter with complete versioned operation |
| T002 | T001 | Adapter/contracts/guards/protocol composition in same unit |
| T003 | T002 | Collection and facade preserved together |
| T004 | T003 | Exact-key CLI plus fixture/guidance source |
| T005 | T004 | Tree planning and facade preserved together |
| T006 | T005 | Complete tree CLI interaction preserved |
| T007 | T003 | Fixture/guidance source folded into command delivery |

The list was fresh and wholly unchecked. A search of the feature artifacts found no external task-ID references or progress records before renumbering; the earlier chat reports did not publish these draft IDs. Dependencies, MVP boundary, acceptance indexes, coverage table, counts, and parallel examples were repaired. No task was checked off.

## Coverage and checks

- Reviewed original versus final acceptance text and preserved all implementation/test obligations; all original concrete path references remain present.
- Verified 27/27 functional requirements have owners, all nine success criteria map to owners, and all five stories have explicit goals and independent acceptance criteria.
- Verified verbatim data-model field/enum/array/ordering constraints, cumulative submission evidence, not-found versus other failed reads, and cancellation semantics remain present.
- Preserved generated-client reuse, no-wait/dry-run guards, singleton-through-collection, tenant behavior, traversal bounds, frozen scope, no-op/error distinction, worker controls, and unchanged other mutations.
- Preserved command-level logging/INFO suppression/DEBUG/redaction/partial-result/EOF/original-exit assertions, real-terminal prompt/no-op proof, successful and failed activity cleanup, and fresh-date inspection regressions.
- Checked sequential unique checkbox IDs, labels/paths, absence of stale task references or unjustified parallel markers, acyclic forward dependencies, and all-unchecked progress.
- Checked local artifact links, Markdown fences, unresolved template markers, trailing whitespace, and `git diff --check`.
- Rechecked hooks: consolidation is mandatory before the optional commit/Ralph hooks. Those optional actions are not dispatched by this review.

## Limitations

No unresolved design uncertainty was identified. This is a code-grounded planning review, not an implementation or performance result. Application tests, generated-doc execution, integration tests, live environment checks, and Ralph were not run. Planned test names/files may be created during implementation; verify matches before treating a test command as evidence. Real-engine behavior and unsupported-host PTY coverage remain subject to the documented validation limits.
