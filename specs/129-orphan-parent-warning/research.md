# Research: Graceful Orphan Parent Traversal

## Decision 1: Make the shared walker return structured partial results instead of only an orphan error

- **Decision**: Refactor the shared traversal contract around `internal/services/processinstance/walker/walker.go` so a missing non-start parent yields structured partial ancestry/family data plus machine-readable missing ancestor keys rather than only `services.ErrOrphanedInstance`.
- **Implementation seam**: add `TraversalResult` and facade `DryRunPIKeyExpansion` types first, then migrate command callers from legacy tuple returns to the structured contract story by story.
- **Rationale**: The current walker already accumulates `chain` state before returning the orphan error, so the repository-native way to unlock partial rendering and preflight continuation is to preserve and expose that accumulated state instead of re-running or reconstructing traversal higher up.
- **Alternatives considered**:
  - Catch `ErrOrphanedInstance` only in commands and reconstruct partial output there: rejected because `cancel`, `delete`, and indirect cleanup preflight would each need to duplicate traversal recovery rules.
  - Leave the walker unchanged and special-case only `walk pi`: rejected because the issue explicitly includes destructive preflight and indirect dependency expansion, not just read-only rendering.

## Decision 2: Keep the new warning contract limited to traversal and dependency-expansion callers

- **Decision**: Apply the new behavior only to callers built on `Ancestry`, `Descendants`, `Family`, and facade dry-run expansion; keep direct `GetProcessInstance`, `GetProcessInstanceStateByKey`, and waiter absent/deleted behavior strict.
- **Rationale**: The spec and clarification session explicitly preserve single-resource strictness and waiter semantics. The codebase already separates these seams: direct lookup lives in the versioned services, while traversal behavior is concentrated in the walker.
- **Alternatives considered**:
  - Make direct key lookup warning-based too: rejected because it would contradict the issue's must-keep behavior and weaken operator expectations for `get process-instance --key`.
  - Loosen waiter not-found handling beyond absent/deleted success: rejected because `internal/services/processinstance/waiter/waiter.go` intentionally treats absence as success only for the desired absent/deleted states.

## Decision 3: Use one structured missing-ancestor contract across `v87`, `v88`, and `v89`

- **Decision**: Keep the orphan-parent partial-result contract version-neutral because all three versioned services already delegate ancestry/family traversal through the shared walker.
- **Rationale**: `internal/services/processinstance/factory.go` constructs `v87`, `v88`, and `v89`, and each service forwards `Ancestry`, `Descendants`, and `Family` to `walker.Ancestry`, `walker.Descendants`, and `walker.Family`. A version-specific fork would add unnecessary complexity unless tests prove a concrete adapter need.
- **Alternatives considered**:
  - Plan only for `v88`/`v89` and leave `v87` as-is: rejected because the shared walker already sits under all versions and the spec requires regression coverage for all three.
  - Introduce separate walker logic per version: rejected because it would violate the repository-native layering and multiply maintenance cost without evidence of different upstream traversal semantics.

## Decision 4: Define success around actionable resolved results, not around whether any ancestor was missing

- **Decision**: Affected traversal and preflight flows should succeed with warnings only when at least one actionable process-instance result was resolved; fully unresolved traversals should still fail normally.
- **Rationale**: This preserves operator usefulness for orphan-child cleanup while preventing empty or misleading success states. The clarification session made this the authoritative success/failure boundary.
- **Alternatives considered**:
  - Fail on any missing ancestor even when partial results exist: rejected because it recreates the current operator dead-end the issue is trying to remove.
  - Always succeed when a missing ancestor is detected, even with zero resolved results: rejected because an empty success would be hard to trust and harder to automate safely.

## Decision 5: Keep dry-run expansion as the main mutation-preflight integration seam

- **Decision**: Route destructive preflight behavior through `c8volt/process/dryrun.go` and its callers instead of embedding orphan recovery separately in `cancel` and `delete` command handlers.
- **Rationale**: Both keyed and paged `cancel`/`delete` flows call `DryRunCancelOrDeleteGetPIKeys`, and indirect resource cleanup uses the same facade path. A single integration here keeps impact calculation and warning propagation consistent.
- **Alternatives considered**:
  - Patch `cmd/cancel_processinstance.go` and `cmd/delete_processinstance.go` independently: rejected because paged and keyed modes would drift and indirect resource deletion could still fail hard.
  - Hide missing ancestors entirely and return only collected keys: rejected because the spec requires callers to distinguish resolved keys from missing ancestor keys.

## Decision 6: Preserve rendering and scripting value by carrying both human and machine signals

- **Decision**: The feature should expose both a user-facing warning and machine-readable missing ancestor metadata through one shared contract.
- **Rationale**: `walk` needs readable warnings for operators, while preflight and JSON/automation-compatible flows need structured data that downstream callers can test and reason about consistently.
- **Alternatives considered**:
  - Warning text only: rejected because scripting and tests would need brittle text matching.
  - Metadata only with no warning text: rejected because human-oriented traversal output should remain immediately understandable without extra decoding.

## Decision 7: Use walker, facade, and command tests as the primary regression seams

- **Decision**: Anchor regression coverage in `internal/services/processinstance/walker/walker_test.go`, `c8volt/process/client_test.go`, and command tests under `cmd/` for `walk`, `cancel`, and `delete`, with explicit non-regression coverage for strict `get` and waiter behavior.
- **Rationale**: Those layers map directly to the shared helper contract, the dependency-expansion facade, and the user-facing commands affected by the issue. They are the closest tests to the changed behavior and match repository guidance to validate near the changed area first.
- **Alternatives considered**:
  - Test only the final commands: rejected because helper/facade regressions would be harder to isolate.
  - Test only the walker: rejected because the issue’s value is in command/preflight behavior, not just in the helper return shape.

## Audit Inventory: Current Failure Chain

- [`internal/services/processinstance/walker/walker.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker.go) returns `services.ErrOrphanedInstance` when a non-start parent lookup yields `d.ErrNotFound`, even though `chain` already contains the resolved descendants/ancestors gathered so far.
- [`cmd/walk_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_processinstance.go) treats any fetch error from `cli.Ancestry`, `cli.Descendants`, or `cli.Family` as a normal command failure, so partial tree/list rendering never runs.
- [`c8volt/process/dryrun.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/dryrun.go) aborts delete/cancel preflight as soon as `c.Ancestry(...)` or `c.Descendants(...)` returns an error, so orphan children stop cleanup work even when other keys were resolved.
- [`cmd/cancel_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go) and [`cmd/delete_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go) both depend on that dry-run expansion for keyed and paged flows.
- [`c8volt/resource/client.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/client.go) also depends on the same dry-run family expansion for indirect process-definition deletion impact.

## Audit Inventory: Strict Seams That Must Not Change

- Direct key retrieval stays strict in the versioned services and command surface, including [`cmd/get_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go) and the service-level `GetProcessInstance` implementations.
- Waiter absent/deleted success behavior remains localized to [`internal/services/processinstance/waiter/waiter.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter.go), which already treats `ErrNotFound` as success only when absent/deleted is the desired end state.
- The shared domain response helpers such as [`internal/services/common/response.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/common/response.go) should remain the place where strict single-resource lookup semantics are normalized.

## Audit Inventory: Version Support Surface

- [`internal/services/processinstance/factory.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory.go) now constructs `v87`, `v88`, and `v89`, so the feature’s regression and contract scope must include all three.
- `v87` still treats direct-key lookup and state-by-key lookup as tenant-unsafe unsupported operations, but its `Ancestry`, `Descendants`, and `Family` methods still delegate to the shared walker.
- `v88` and `v89` both implement tenant-safe `GetProcessInstanceStateByKey` via `GetProcessInstance`, and they also route traversal through the shared walker.
- Because traversal is shared across versions, the orphan-parent partial-result behavior should be driven by the walker contract rather than by per-version command branching.

## Regression Anchors

- [`internal/services/processinstance/walker/walker_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker_test.go) already has an orphan-parent test that currently expects `ErrOrphanedInstance`; it is the primary seam for updating the helper contract.
- [`c8volt/process/client_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go) already exercises `DryRunCancelOrDeleteGetPIKeys`, making it the best seam for proving resolved roots/collected keys vs missing ancestor handling.
- [`cmd/walk_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go) is the user-visible seam for partial ancestry/family rendering and warning behavior.
- [`cmd/cancel_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go) and [`cmd/delete_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go) are the best seams for actionable preflight continuation with orphan children.
- Strict non-regression coverage belongs in [`cmd/get_processinstance_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go) and [`internal/services/processinstance/waiter/waiter_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/waiter/waiter_test.go).
