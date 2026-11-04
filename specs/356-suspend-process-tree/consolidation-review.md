# Task Consolidation Review: Process-Instance Tree Suspension

**Review date**: 2026-10-26
**Feature / branch**: `356-suspend-process-tree` · issue #356
**Task list**: [tasks.md](tasks.md)
**Design context**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [CLI contract](contracts/cli.md), [facade/service contract](contracts/facade-service.md), [quickstart.md](quickstart.md)

## Current assessment

**Before: 2 tasks. After: 2 tasks. Outcome: no task changes.**

Both fresh-draft tasks remain unchecked and sequential: T001 [US1] provides the full frozen planning contract, and T002 [US2] integrates the command and all shared US3/US4 acceptance. No IDs were merged, removed, renumbered, or reassigned. The existing task file was not rewritten during this review.

Tasks SHA-256 before/after review: `378b22d25cd2b66b2e5ce8cdc84e93aea62ae16396778f64e22f56b3933bd7b8`.

## Code-grounded boundary review

### Strongest plausible merge: T001 + T002

**Candidate outcome**: Deliver the entire read-only tree planner, public interface/model/copy semantics, CLI approval/mutation lifecycle, every output mode, real-terminal behavior, docs, and their validation as one checkbox.

**Rejected for concrete implementation and validation complexity**:

- T001 creates a new fail-closed planning protocol: capability and all-input validation before discovery; existing local preview timing validation even for eventually empty target sets; complete descendant identity checks; deterministic mixed-state/overlap classification; zero executable plan on any late failure; copied public ownership. `internal/services/processinstance/v810/suspend.go` shows that local validation is already separate from mutation, while `internal/services/processinstance/traversal/result.go` exposes complete/partial/unresolved results rather than an automatically safe mutation plan. These requirements need direct planner/facade assertions before commands rely on that plan.
- T002 integrates a different execution/evidence protocol: completed empty plan versus dry-run, frozen approval and no rediscovery, accepted versus confirmed partial results, failure before no-wait envelope selection, logger routing, PTY control text, and activity cleanup. `cmd/suspend_processinstance_batch.go` currently conflates empty preview with dry-run, so a new tree path must deliberately handle actual no-op success. `cmd/cmd_views_processinstance_suspend.go` and the existing logging/terminal/activity tests establish the result and lifecycle boundaries that cannot be replaced by service-only tests.
- Combining creation and validation of both protocols, plus real-terminal cases, would make one iteration carry substantially different failure/state models and more difficult regression diagnosis. The retained boundary hands off only a typed frozen plan to existing execution; it does not require temporary implementation, repeated traversal/mutation logic, or a second schema rewrite. The new planner is independently usable with the already-existing executor after T001.
- No task boundary was retained merely because files, packages, layers, or story labels differ. A service/facade split was explicitly rejected below. The substantive split is between establishing the new complete-scope/copy contract and integrating it into existing mutation/output/interactive lifecycles, each with the validation needed to make its outcome reviewable.

T002 remains a focused command outcome because it reuses `SuspendProcessInstances`, `formatProcessInstanceSuspensionSummary`, `formatProcessInstanceSuspensionOutcome`, `formatProcessInstanceSuspensionFailure`, and the simple existing indicator wrapper. It does not implement workers, retries, waiting, transport diagnostics, or a new progress engine. Reuse targeted existing fixture families; an unnecessary cross-product of every tree scenario and logger/output configuration is not required.

### Other boundaries and shared behavior evaluated

| Candidate decomposition or merge | Decision and inspected evidence |
| --- | --- |
| Planner service versus thin facade/conversion | Already together in T001. Existing `c8volt/process/suspend.go` is mechanical delegation; `fromDomainSuspensionResult` / `fromDomainSuspensionResults` in convert.go show the mapping pattern. A separate facade task would add a thin handoff without an implementation outcome. |
| Public API versus required concrete test doubles | Already together in T001. `cmd/process_api_stub_test.go` and `c8volt/resource/client_test.go` concretely implement the process facade surface; widening it must keep these callers compiling in the same unit. Internal API already has the guard/preview/traversal methods and needs no widening. |
| Input/option validation versus frozen plan/classification | Already together in T001. `NormalizeProcessInstanceSuspensionKeys`, v810 local dry-run timing checks, and `DescendantsResult` feed one all-or-error planning outcome; splitting them could expose a superficially usable but unsafe plan. |
| Single instance versus collections/overlapping subtrees | One planner and existing collection executor throughout both units. `internal/services/processinstance/suspend.go` owns worker policy, fail-fast and ordered partial results through `pool.ExecuteSlice`; no new executor task is justified. |
| Command lifecycle versus tree views/error handling | Already together in T002. No-op/preview/accepted/failure envelope selection depends on the same frozen plan and call result; an intermediate runner without the correct view could claim acceptance or print duplicate envelopes. |
| Runtime behavior versus tests/fixtures/formatting | Kept inside each owner. Existing service stubs and command HTTP/subprocess/PTY helpers can verify the proper boundaries; unit/renderer assertions cannot replace the required execution, logger and terminal checks. |
| US3 outcome protection versus US2 command integration | Already covered by T001/T002; no extra checkbox. The same existing executor and tree renderer own partial acceptance/confirmation/unattempted evidence in all modes. A later story-specific implementation would repeat renderer and fixture edits. |
| US4 docs/permissions versus CLI integration | Already in T002, with guard facts from T001. `TestAllCommandHelpFocusesOnActions` rejects stream/logger implementation vocabulary and `TestSuspendProcessInstanceMetadata` checks the public grammar. README/source help/generated docs must match the implemented mode immediately. |
| Spinner/global progress changes | No additional task. The existing local `indicatorEnabled` / `startCommandActivity` wrapper already covers collection execution and waiting. `TestSuspendProcessInstanceActivity` owns scope and suppression assertions. |
| Traversal/pagination redesign | No task. `walker.Descendants` already includes the selected key and traverses all states; v810 GetDirectChildrenOfProcessInstance requests a single page of up to 1,000. The feature explicitly retains this documented constraint. |

There are no nonadjacent task pairs in a two-task list. Shared behavior across the four story phases was reviewed through the existing executor, output models, and fixture families; their acceptance obligations stay with T001/T002 rather than generating duplicate tasks.

## Coverage, dependency, and progress checks

Performed requirements-to-owner review and structural document checks:

- Every FR-001–FR-023 and SC-001–SC-008 has an owner in tasks.md. All four story phases retain independent acceptance criteria, including the operator feature needing both units and the usable library increment after T001.
- T001 includes models, public interface, service, thin facade, copies/conversion, affected concrete stubs, local-validation/discovery tests and relevant compile coverage. T002 includes CLI mode, tree-only view, all-ineligible no-op, error/exit/machine contracts, configured INFO/ERROR/DEBUG logger checks, real PTY interaction, simple activity, docs and targeted validation.
- Model field names, JSON names, classification values, ordered projection, non-nil arrays, no-executable-plan-on-error, and slice ownership constraints are explicit. Plan facts never become fake mutation outcomes.
- The only dependency is T001 -> T002. There are no [P] markers or false task-level parallel claims. Per-story examples are read-only review opportunities and preserve serial shared-Cobra execution unless subprocess-isolated.
- Exactly two strict checklist lines have sequential IDs, one primary story label each, and explicit file paths. No completed/partial task evidence exists; no progress or Ralph memory artifact was present to preserve. No ID changes were made and no external task-ID references require migration.
- Local artifact links, task/story references, requirement/success IDs, Markdown fences, and whitespace were checked. The no-change hash was verified after saving this separate review.
- No consolidation-review section existed in tasks.md. **Review relocation: none.** No standalone review text was added there.

## Limitations and checks not performed

This is a planning review, not implementation validation. Application tests, runtime benchmarks, live requests, and cluster checks were not run. Listed implementation checks are future work-unit obligations, not passing evidence. Source was available for the inspected ownership, helper, interface, caller, and fixture paths; no unresolved consolidation uncertainty was identified.

The retained direct-child lookup limit and non-atomic frozen scope remain explicit product constraints, not newly introduced blockers or promises of complete observation beyond that bound. This review modified only consolidation-review.md; tasks.md and all other artifacts were preserved.
