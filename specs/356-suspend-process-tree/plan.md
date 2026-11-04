# Implementation Plan: Process-Instance Tree Suspension

**Branch**: `356-suspend-process-tree` | **Date**: 2026-10-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/356-suspend-process-tree/spec.md`, derived from issue [#356](https://github.com/grafvonb/c8volt/issues/356).

## Summary

Add opt-in --tree to the existing suspension command. Resolve the selected instances and descendants, merge overlapping scopes, classify observed states, freeze active target keys, preview or confirm once, then pass those keys to the existing collection suspension service. Already-suspended instances remain planning facts; terminal and other non-ACTIVE members are excluded without mutation. Discovery failures prevent all mutations. A successfully resolved scope with no active targets is a succeeded no-op.

Use one new read-only public planning method and a version-neutral internal helper built on existing guard/preview/DescendantsResult methods. Do not add tree behavior to the mutation executor, alter standalone walk or its retained 1,000-direct-children lookup limit, widen versioned interfaces, or introduce a generic framework. Keep the existing pool, wait/no-wait semantics, retry evidence, logger helpers, one-envelope contract, and simple execution spinner.

Tree-only output adds frozen plan metadata beside the existing mutation items; default exact-key JSON and behavior stay unchanged. Source help, README, metadata, and generated CLI docs accompany future implementation. This planning phase produces design artifacts only.

## Technical Context

**Language/Version**: Go 1.26.0, repository toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2, pflag 1.0.10, Viper 1.21.0, slog, generated Camunda v810 client, `toolx/pool`, existing service options/error conversion/activity writer, and testify 1.11.1. No new dependencies or client regeneration.

**Storage**: Per-invocation plan slices and existing result collections only. No plan registry, file persistence, or server-side transaction.

**Testing**: Existing unit/stub tests, test-owned HTTP servers, isolated command subprocesses, real PTY stdin tests, and activity sinks. Targeted race checks for concurrent command fixtures; all-package compile check for public interface additions. No runtime tests during planning and no live-environment gate.

**Target Platform**: Existing Linux/macOS CLI and public Go library; tree suspension requires configured Camunda 8.10. Existing 8.8/8.9 guards must run before discovery.

**Project Type**: Existing CLI plus public facade and internal service layers.

**Performance Goals**: Preserve existing mutation worker limits and bounded confirmation. Plan once before mutation and perform no post-approval tree discovery. Deduplicate logical targets; skip seeds already covered by an earlier complete traversal. No new latency/throughput promise or large-tree benchmark requirement.

**Constraints**: Selected-plus-descendant scope; no ancestor escalation or state-pruned traversal. Retain up to 1,000 direct children per lookup, with no beyond-limit completeness claim. Non-atomic frozen scope; no repeated discovery, rollback, native batch, resumption, or global helper changes. Preserve diagnostic routing, empty-result behavior, mode precedence, and exact-key contracts.

**Scale/Scope**: One new boolean CLI flag, one public planner method, corresponding domain/public plan models, a focused command mode/view, and tests/docs. Existing single-instance and collection mutation paths remain the only executors.

## Constitution Check

Initial gate passed before research: this is an opt-in extension using existing ownership and contracts. Phase 0 resolved service/API, model, output, option-validation, and empty-scope questions. Post-design evaluation confirms no constitutional or operator UX exception is needed.

| Principle/gate | Initial result | Post-design evidence |
| --- | --- | --- |
| I. Operational proof over intent | Pass | Plan observations do not establish accepted/confirmed mutation. Existing default waiter and no-wait evidence remain authoritative. |
| II. CLI-first, script-safe interfaces | Pass | --tree is explicit; exact-key schema and semantics unchanged. Tree payload retains items plus plan, one envelope, accepted keys, plain prompts, original exit classes. |
| III. Proportionate validation | Pass | Planning uses document checks only. Future work starts with targeted isolated tests; compile-check widened public API and use targeted race tests for relevant concurrent fixtures. No blanket repeat/full-suite gate. |
| IV. Documentation matches behavior | Pass | Future source help/metadata, README and generated CLI docs cover scope, permissions, classifications, retained limits, and non-atomic execution. |
| V. Small compatible repository-native changes | Pass | One planner, existing traversal/collection execution, no new dependency, pool, waiter, workflow framework, or global progress change. |
| Architecture boundaries | Pass | Internal service owns traversal/classification; facade only maps/delegates/copies; cmd owns approval/rendering/activity; existing adapters own capability and wire differences. |
| Logger/operator UX | Pass | Human scope/preview/outcome facts use configured INFO helpers. Safe aggregate errors and independent redacted DEBUG remain. Shared help gate prevents stream/logger implementation vocabulary in help. |
| Empty/no-op/prompt rules | Pass | Complete plan plus zero TargetKeys yields succeeded with items array empty and retained plan, no prompt/call/activity; failed/invalid planning never becomes no-op. |
| Implementation testing boundary | Pass | Unit/fake HTTP/subprocess/PTY validation only; no live environment discovery, provisioning, mutation, cleanup, or target approval gate. |

**Compatibility note**: Adding `PlanProcessInstanceSuspension` to `process.API` requires external custom implementations and affected repository concrete stubs to implement it. Update all in one compiling unit. Existing consumers of exact-key suspension results and existing versioned/internal interfaces are unchanged. The opt-in tree payload is specified separately in [contracts/cli.md](contracts/cli.md).

The older #346 artifacts describe the previous exact-key feature; this issue adds explicit tree selection without changing that default. The constitution's proportionate-validation rule takes precedence over older blanket full-suite language.

## Project Structure

### Documentation (this feature)

```text
specs/356-suspend-process-tree/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── cli.md
│   └── facade-service.md
├── checklists/requirements.md
└── tasks.md                    # Future speckit-tasks output; not created here
```

### Source Code (repository root)

Planned ownership, not an instruction to create every listed test file if an existing cohesive file suffices:

```text
internal/domain/
└── processinstance_suspension_plan.go       # New frozen planning facts
internal/services/processinstance/
├── suspend_plan.go                         # New version-neutral planner
├── suspend_plan_test.go                    # Isolated planning behavior
├── suspend.go                              # Reused collection executor
├── traversal/                              # Reused structured descendant results
├── walker/                                 # Reused traversal, no paging changes
└── v810/suspend.go                          # Existing local preflight and mutation/wait
c8volt/process/
├── api.go                                  # One public planning method
├── model.go                                # Public planning types
├── suspend_plan.go                         # Thin planning facade
├── convert.go                              # Mechanical/copying plan conversion
└── suspend_plan_test.go                    # Facade/guard/copy tests
cmd/
├── suspend_processinstance.go               # --tree registration/dispatch
├── suspend_processinstance_tree.go          # Tree approval/execution lifecycle
├── cmd_views_processinstance_suspend_tree.go # Tree-only final views/payload
├── cmd_views_processinstance_suspend.go      # Reused result/failure formatters
├── suspend_processinstance_*test.go          # Tree cases and exact-key controls
├── process_api_stub_test.go                 # Public-interface stub alignment
├── command_contract_test.go                 # Flag/help capability coverage
└── command_help_focus_test.go               # Existing whole-command help gate
c8volt/resource/client_test.go               # Concrete public-interface stub alignment
README.md                                   # User-facing tree contract
docs/cli/                                  # Regenerated from source help
docs/index.md                              # Regenerated README mirror
```

**Structure Decision**: Existing packages own all work. Put the new tree mode in a focused command file rather than growing the ordinary batch path; keep final view formatting in the focused view file. No new service interface is needed below the public facade, and no generated client or shared pool/activity changes are planned.

## Phase 0 — Research decisions

See [research.md](research.md) for decisions, rationale, and alternatives. Research resolved:

- Existing DescendantsResult includes the selected key and walks mixed-state descendants; FamilyResult would incorrectly escalate scope.
- Version-neutral planning can reuse the existing guard, first-seed local preview, explicit-key discovery authorization, and option suppression.
- First-seed preview validates confirmation timing before discovery even when no active target is later found; it never establishes observed eligibility.
- First-seen scope classification, copied frozen keys, and fail-closed planning avoid race reconciliation, executable partial plans, and redundant discovery.
- A tree-only typed payload keeps plan facts separate from unchanged mutation evidence.
- The simple existing execution activity is sufficient; completion progress machinery is unnecessary.

No unresolved design questions or clarification markers remain.

## Phase 1 — Design and contracts

- [data-model.md](data-model.md): RequestedKeys, classified Instances, TargetKeys, copy/ordering/error invariants, separate tree payload, and lifecycle.
- [contracts/facade-service.md](contracts/facade-service.md): Public planner signature, neutral service algorithm, local preflight/options, guards and execution delegation.
- [contracts/cli.md](contracts/cli.md): --tree grammar, focused mode, frozen confirmation, preview/no-op/failure envelopes, logger messages, accepted-key semantics, and simple activity lifecycle.
- [quickstart.md](quickstart.md): Runnable isolated implementation validation, logger/empty/PTY matrix, metadata/help/doc checks, and proportional broadening rules.

### Cohesive implementation handoff

Generate dependency-ordered work units next; do not create separate tasks merely for reading context, adding test doubles, formatting, or hypothetical parallelism.

1. Deliver the domain/public plan, neutral planning service, thin facade, concrete stub alignment, and meaningful service/facade tests together. This leaves a compiling, independently usable planning capability.
2. Integrate the opt-in CLI lifecycle/views, output/empty/activity/terminal regressions, metadata/help, README and generated docs together, reusing the first unit's plan and existing executor. Preserve all four story acceptance areas without implementing a separate executor per story.

These are planning boundaries, not a generated tasks list. No tasks.md or runtime implementation is produced by this command.

### Acceptance and validation allocation

| Coverage | Owning design | Validation evidence expected during implementation |
| --- | --- | --- |
| US1 / FR-001–FR-005, FR-020 | Planner plus opt-in dispatch | Nested/overlapping/mixed-state scope, no ancestors, one eligible target, exact-key zero-discovery controls. |
| US2 / FR-004, FR-007–FR-011 | Complete plan and command lifecycle | No mutation on discovery failure, frozen counts/keys, one approval, dry-run versus actual empty/no-wait results, invalid input. |
| US3 / FR-006, FR-012–FR-018 | Existing executor plus focused views | Real partial exits/evidence, accepted keys, one envelope with EOF, configured logger matrix, worker/activity/PTY checks. |
| US4 / FR-019, FR-021–FR-023 | Guard/documentation contract | 8.8/8.9 no-I/O guards, action-focused help gate, permissions/limits, generated docs, retained neighboring semantics. |

## Complexity Tracking

No constitution violations or new abstractions require an exception. The small plan type and dedicated tree mode are necessary to freeze operator-approved scope and keep discovery classifications out of mutation evidence. Existing collection execution, traversal, helper logging, and activity mechanisms are reused.
