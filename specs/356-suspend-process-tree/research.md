# Research: Process-Instance Tree Suspension

**Feature**: [spec.md](spec.md) | **Date**: 2026-10-26

All design questions are resolved. Research used the repository's existing traversal, suspension, mutation views, test fixtures, constitution, and operator UX rules. No runtime or live-environment validation was performed.

## 1. Plan a tree, execute an ordinary collection

**Decision**: Add `PlanProcessInstanceSuspension(ctx, keys, opts...)` to the public process facade, backed by a version-neutral helper in `internal/services/processinstance/suspend_plan.go`. It uses the existing internal `API` guard, local preview, and `DescendantsResult` methods. Execute the returned active keys through the existing `SuspendProcessInstances` method. Do not add an execution method, global `WithTree` option, or new versioned interface method.

**Rationale**: `c8volt/process/suspend.go` already delegates collection execution to the internal service. `internal/services/processinstance/suspend.go` owns normalization, worker policy, fail-fast, ordered skipped slots, and partial evidence. Reusing it preserves proven mutation and waiter semantics.

**Alternatives considered**: A command calling walk or looping through descendants would put backend orchestration in `cmd`. A facade doing the traversal would violate the thin facade boundary. Reworking `toolx/pool` or `ExecuteBulkOperation` would add unrelated behavior and duplicate summaries.

## 2. Descendants only, within the retained lookup limit

**Decision**: Call `DescendantsResult` for normalized seeds, never `FamilyResult` or ancestor/root expansion. Require a complete descendant result for the requested seed, including the seed and matching per-key observations. Retain the existing up-to-1,000-direct-children lookup behavior without changing paging.

**Rationale**: `walker.Descendants` includes the selected instance and visits children regardless of state. `FamilyResult` first walks ancestors and would expand scope incorrectly. The v810 direct-child lookup currently requests one search page of size 1,000; #356 explicitly retains this constraint.

**Alternatives considered**: Root cancellation planning uses different family and terminal semantics. A new traversal or pagination redesign exceeds the authorized scope. Do not claim completeness outside the documented lookup limit or add a beyond-limit implementation gate.

## 3. Fail before mutation and before misleading no-ops

**Decision**: Check capability before discovery, normalize every seed, check context, and perform one existing single-instance dry-run on the first normalized seed before traversal. Force dry-run only on that copied preflight option slice. Return no executable plan on any error, missing identity/observation, or explicitly partial/unresolved traversal; check context again before publishing success.

**Rationale**: The v810 single-instance preview validates configured confirmation timing without I/O when waiting applies. Calling it before discovery also catches invalid timing when the eventual active target set is empty. No-wait retains its existing omission of irrelevant waiting checks. A preview is local validation, not eligibility evidence.

**Alternatives considered**: Validating timing only by previewing the final target collection misses invalid waiting configuration when that collection is empty. Reimplementing backoff validation or adding a new timing API is unnecessary. Returning partial target keys with an error invites accidental partial mutation or false no-op handling.

## 4. Stable classification and frozen ownership

**Decision**: Merge seed results in normalized input order and traversal-key order. Keep the first observed occurrence of each key; skip a seed whose subtree was already completely visited. Classify ACTIVE as `eligible`, SUSPENDED as `already_suspended`, and other observed states as `excluded`. Unknown state is a real excluded observation; a missing observation record is malformed discovery. Copy all slices at facade boundaries and copy target keys before approval.

**Rationale**: The plan is an advisory snapshot, not a state reconciliation engine. Existing traversal returns ordered keys plus an identity-indexed observation map. First-seen observations keep deduplication deterministic across overlap and state races. A small plan needs only requested keys, classified members, and active target keys; it needs no ancestry graph, ownership token, or persistence.

**Alternatives considered**: Re-reading duplicate keys or rediscovering after confirmation could change approved scope. Filtering discovery to ACTIVE would hide active descendants below an ineligible parent. Collapsing selected children to roots would violate subtree boundaries.

## 5. Preserve explicit-key authorization and compact diagnostics

**Decision**: Tree discovery uses explicit-key authorization semantics with `WithIgnoreTenant`, including public planner calls, and copies caller options before adding planner-local workflow/detail suppression. Keep existing DEBUG transport diagnostics and redaction. Do not install a new logger, callback framework, or global suppression setting.

**Rationale**: The CLI already uses `collectExplicitPIAdminInputOptions`; descendants should not silently disappear because of a configured discovery tenant filter. Existing service options can suppress nested workflow/detail chatter while the command owns final scope reporting. Read/search permissions are additional requirements for tree mode, including dry-run.

**Alternatives considered**: Default tenant-scoped searches could produce an incomplete apparent subtree. Printing joined discovery or mutation errors would violate mandatory diagnostic rules. Broad changes to standalone walk or HTTP logging are unnecessary.

## 6. Keep plan facts separate from mutation evidence

**Decision**: Add new domain/public plan models, but leave existing `SuspensionResults` and its exact-key JSON shape unchanged. A tree-only command payload contains `items` (existing eligible preview or actual mutation results) and `plan` (frozen classifications). Already-suspended and excluded instances appear only in the plan. On actual no-op, `items` is `[]`, the envelope is succeeded even with no-wait, and plan facts remain available.

**Rationale**: Current suspension outcomes distinguish accepted requests from observations and unattempted work. Fabricating mutation results for an already-existing state would corrupt that evidence. An additive tree-only payload makes the opt-in scope inspectable without changing existing exact-key consumers or shared envelope schema.

**Alternatives considered**: Adding fake accepted/confirmed/skipped mutation items for ineligible members conflates discovery with execution. Emitting a separate plan JSON document before results breaks the one-envelope contract. Reusing the exact-key runner's empty-preview branch would falsely label an actual no-op as dry-run.

## 7. Keep tree orchestration and rendering focused

**Decision**: Register `--tree` and dispatch in `cmd/suspend_processinstance.go`; own tree plan/approval/execution orchestration in `cmd/suspend_processinstance_tree.go`. Put tree-only payload and final formatting in `cmd/cmd_views_processinstance_suspend_tree.go`, reusing current suspension count, outcome, and bounded-failure formatters. Human scope/outcome messages use `renderHumanLine`; prompts stay plain configured stderr; machine modes use stdout.

**Rationale**: A distinct tree lifecycle should not grow the existing ordinary batch runner. Existing helpers already implement JSON-before-keys precedence, quiet/level filtering, accepted-key output, safe aggregate errors, and exit-after-render behavior.

**Alternatives considered**: Global view/helper refactors and a generic mutation plan framework add complexity unrelated to #356. Explicit stream/logger implementation vocabulary in command Short/Long help fails `TestAllCommandHelpFocusesOnActions`; keep that detail in contracts/README and describe operator actions in help.

## 8. Retain the simple spinner and existing executor

**Decision**: Wrap only approved mutation execution in the existing `indicatorEnabled` / `startCommandActivity` pattern, with label `suspending N process instance(s)` and deferred cleanup before final rendering. Use no new completion events, counters, semantic reporter, or pool changes. Preview, failed planning, and empty active scope do not start execution activity.

**Rationale**: #346 already provides this local wrapper and a shared activity writer that owns terminal eligibility, animation timing, and priority over HTTP fallback activity. Collection execution includes waiting and backoff, so a single activity spans the entire operation.

**Alternatives considered**: A completion-event framework is unnecessary for a busy indicator and would widen implementation scope. Per-request activity alone disappears between confirmation reads.

## 9. Validate proportionately and preserve compiling work units

**Decision**: Use isolated service stubs, generated-client/test-owned HTTP fixtures, subprocess exits, and real PTY stdin. Extend existing suspension regression families and the global help/metadata gates. Compile-check all consumers after widening `process.API`; add required concrete stub methods in the same unit. Run targeted service/facade/command tests first, with race coverage for concurrent command fixtures. Broaden only if actual implementation blast radius or failures justify it.

**Rationale**: Constitution III rejects runtime tests for planning and blanket full-suite gates. The prior #346 regression showed that command-specific tests alone miss cross-command help policy. Existing fixture cancellation tests also show the value of deterministic synchronization rather than sleeps.

**Alternatives considered**: Real clusters, integration tags/scripts, environment designation, and live cleanup are prohibited implementation gates. Runtime test execution during this planning phase adds no useful evidence.
