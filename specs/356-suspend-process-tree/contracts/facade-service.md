# Facade and Service Contract: Tree Suspension Planning

**Feature**: [spec.md](../spec.md) | **Models**: [data-model.md](../data-model.md)

## Public surface

Add to `c8volt/process.API`:

```go
PlanProcessInstanceSuspension(ctx context.Context, keys typex.Keys, opts ...foptions.FacadeOption) (ProcessInstanceSuspensionPlan, error)
```

This method plans selected-plus-descendant scope only. It performs no suspension mutation and does not prompt. CLI `--tree` uses it; ordinary exact-key suspension never does.

Keep existing `SuspendProcessInstance` and `SuspendProcessInstances` signatures and behavior unchanged. Callers execute a successful plan by passing a copy of `plan.TargetKeys` to the existing collection method. The collection executor remains unaware of trees and does not rediscover scope.

The facade copies inputs, maps facade options to call options, delegates to the internal helper, mechanically copies/maps plan fields, and converts errors through `ferrors.FromDomain`. No traversal loop, classification, pool, waiting, or transport logic belongs in the facade.

Adding the method widens the public interface for custom implementers. Update all affected concrete test doubles in the same compiling unit, including `cmd/process_api_stub_test.go` and `c8volt/resource/client_test.go`. Interface-embedding fakes need an explicit planner callback in tree tests. No change is needed to the internal versioned API or generated client.

## Internal planning helper

Add a version-neutral helper under `internal/services/processinstance/suspend_plan.go`, accepting the existing `API`, explicit seeds, context, and service options. Introduce the plan types in `internal/domain` and reuse existing `ValidateProcessInstanceSuspension`, `SuspendProcessInstance`, and `DescendantsResult` methods.

Execution order:

1. Guard suspension capability before I/O, including dry-run and empty library input on unsupported versions.
2. Normalize all seed keys; malformed or empty input fails before discovery. Check caller cancellation.
3. Copy options and append `WithDryRun` last for one local `SuspendProcessInstance` preview of the first normalized seed. This validates the existing backoff/timing contract without I/O, including plans that later have no active targets. Preserve NoWait's existing exemption from confirmation timing. Never use this preview as a state observation.
4. Build discovery options separately, ensuring `WithIgnoreTenant` for explicit-key authorization and adding existing workflow/process-instance detail suppression. Do not mutate caller option slices or forward the forced preflight dry-run into actual execution. Preserve caller context, remaining options, and DEBUG transport diagnostics.
5. Visit seeds in input order using `DescendantsResult`, skipping a seed already covered by an earlier complete traversal. Never use ancestry, FamilyResult, root cancellation planning, or the walk CLI.
6. Reject any error, non-complete outcome, wrong descendant scope/seed identity, absent seed, malformed returned key, missing observation, or mismatched observation identity. Return a zero plan and a domain-classified error, not an executable partial plan.
7. Merge traversal keys in first-seen order and classify using the observed state. Discovery must not filter/prune on ACTIVE or SUSPENDED because active descendants can lie below ineligible parents.
8. Check context again and publish the complete plan with copied/non-nil arrays. A successful empty TargetKeys collection is a valid no-op; invalid or failed planning is not.

Use existing validation and unsupported error classes. Report malformed discovery through `domain.ErrMalformedResponse`; explicitly partial/unresolved discovery is an unsatisfied planning precondition using the established domain precondition class. Preserve the original discovery cause when wrapping errors. Do not log raw errors or return success with partial executable targets.

## Existing collection execution

After approval, call existing `SuspendProcessInstances` exactly once with copied TargetKeys, chosen workers, and ordinary explicit-key options with dry-run disabled. A read-only local preview of these keys is permitted for existing local validation, but it must not trigger descendant discovery; avoid redundant preview loops when the planner already performed the required validation.

Collection execution retains worker limits, no-worker-limit, fail-fast, standard submission retries, result order, skipped slots, no-wait, bounded confirmation, and accepted-but-unconfirmed evidence. One logical target does not prohibit existing safe submission retries. No root collapse or ordering guarantee between parent and child mutations is added.

No call is required for an actual no-op. Do not invent a per-member execution record for an already-suspended or excluded instance. The command renders its frozen plan and an empty existing mutation array.

## Scope, permissions, and compatibility

- Explicit seed keys use backend authorization without configured discovery tenant filtering. Tree mode adds read/search permission requirements even for dry-run and no-wait.
- Capability checks reject 8.8/8.9 before discovery. Actual endpoint differences remain in existing adapters.
- Preserve standalone walk, cancellation, deletion, exact-key suspension, and shared pool behavior.
- Retain the existing single-page direct-child lookup bound of 1,000; do not add pagination, infer a new global tree cap, or add a live large-tree validation gate.
- A completed plan is only the available discovered scope within that retained limit, not an atomic snapshot or a guarantee about future descendants.

## Required isolated coverage

Planner tests cover guards before I/O, all-input validation, first-seed local preflight, invalid timing with no eligible targets, no-wait timing behavior, nested/mixed-state subtrees, overlap order, failure in a later seed, partial/unresolved/malformed results, context cancellation, and zero mutations.

Facade tests cover copied ownership, mechanical state/classification conversion, option propagation, error conversion, real configured-version guards composed with test-owned HTTP fakes, and execution using only copied frozen targets. Existing suspension collection/factory/waiter tests remain authoritative for execution semantics; extend relevant fixtures rather than duplicating a mutation engine in tests.
