---
description: "Dependency-ordered implementation outcomes for explicit process-instance tree suspension"
---

# Tasks: Process-Instance Tree Suspension

**Input**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [CLI contract](contracts/cli.md), [facade/service contract](contracts/facade-service.md), [validation guide](quickstart.md)

**Feature / branch**: `356-suspend-process-tree` · GitHub issue #356

**Tests**: Required by the feature acceptance criteria. Necessary fixtures, behavioral tests, formatting, and targeted checks belong to each implementation outcome, not separate checkbox tasks. Use test-owned fakes only.

**Organization**: Two cohesive work units, primarily owned by US1 and US2. US3 and US4 retain their own acceptance phases and are delivered by those same units. This follows the repository rule to use stories for coverage without repeatedly editing the same behavior. No [P] tasks are declared.

## Execution rules

- Read the current artifacts, `AGENTS.md`, the constitution, and `specs/ralph-implementation-rules.md` before implementing. Routine context, branch checks, progress updates, and review are responsibilities of each work unit, not standalone tasks.
- Keep all backend traversal, classification, workers, retries, and waits in internal services. Facades map/delegate/copy/convert errors. Commands own validation, approval, views, and execution activity. Reuse existing mechanisms before adding helpers.
- Preserve the existing 1,000-direct-children-per-parent lookup constraint. Do not change traversal pagination, standalone walk/cancel/delete, generated clients, shared pools, global options, logging infrastructure, or activity frameworks.
- Do not run live integration/smoke scripts, real Camunda requests, cluster discovery/provisioning, deployments, or live cleanup. Inspect test targets and sanitize inherited flags/config/profiles/credentials as specified in [quickstart.md](quickstart.md). Report actual skips or unavailable coverage.
- Write focused failing regression cases within their owning work unit, then implement the missing behavior and pass relevant checks. Keep interface/model/stub changes compiling together before completion; do not commit a broken intermediate contract.
- Every new or materially changed function/type/test needs the repository-required explanatory comment. Run targeted gofmt on touched Go files.
- Validate proportionately under Constitution III. Start with the commands below; broaden only for actual uncovered impact or failures. Do not rerun suites merely to commit, change prose, or mark a task complete. Planning-only changes require document checks, not application tests.
- Preserve all task IDs and completed evidence once work begins. Record actual implementation checks in feature progress/memory as required by the selected implementation workflow. Conventional commit subjects for this feature must end in ` #356`.

## Phase 1: Setup (Shared Infrastructure)

**Goal**: Reuse the initialized repository and existing test/runtime tooling.

No setup checkbox is required. No new dependency, environment, generated client, configuration framework, or repository structure is needed.

## Phase 2: Foundational (Blocking Prerequisites)

**Goal**: Preserve a compiling public interface and one collection execution path.

The necessary domain/public models, planner surface, and concrete test-double updates are part of T001. Do not split them into temporary scaffolding tasks. T001 blocks CLI integration.

## Phase 3: User Story 1 — Suspend a selected subtree (Priority: P1)

**Goal**: Deliver a usable public planning capability that resolves each selected subtree, freezes active targets, and can be combined with the existing collection executor without ancestor escalation.

**Independent test**: With isolated nested/mixed-state/overlapping relationships, call the new planner and the existing collection executor. Assert the stable unique active target set, zero mutations during planning, no family/root lookup, no ineligible mutations, and unchanged exact-key behavior.

- [x] T001 [US1] Deliver the complete tree suspension planning contract in `internal/domain/processinstance_suspension_plan.go`, `internal/services/processinstance/suspend_plan.go`, `c8volt/process/api.go`, `c8volt/process/model.go`, `c8volt/process/suspend_plan.go`, and `c8volt/process/convert.go`, including concrete stub alignment and isolated service/facade tests.

  **Implementation and invariants**:

  - Add domain/public `ProcessInstanceSuspensionPlan`, `ProcessInstanceSuspensionPlanItem`, and the classification type exactly as [data-model.md](data-model.md) defines. Keep these field constraints verbatim: RequestedKeys — “All normalized unique explicit seeds, in input order, including seeds that overlap earlier subtrees.” Instances — “Unique discovered members in first-seen traversal order.” TargetKeys — “Frozen eligible mutation keys in the same order as their entries in Instances.” Key — “Discovered instance identity.” ObservedState — “Planning observation; never post-submission confirmation evidence.” Classification values are `eligible`, `already_suspended`, and `excluded`; JSON names are `requestedKeys`, `instances`, `targetKeys`, `key`, `observedState`, and `classification`.
  - Preserve the model constraints: “TargetKeys is exactly the ordered projection of Instances classified eligible. Each target appears once; parent/child identities remain distinct.” “Successful planning exposes non-nil arrays. Any discovery, malformed-response, explicit partial/unresolved-result, or context failure returns a zero plan plus error. The caller must check error before considering a no-op.” Copy input and all returned slices; do not expose traversal maps/edges, redundant counters, generated/internal types, or rendered messages through the public plan.
  - Add `PlanProcessInstanceSuspension(ctx, keys, opts...) (ProcessInstanceSuspensionPlan, error)` to public `process.API`. Its facade must only copy/map inputs/options, delegate to a version-neutral helper, copy/map results, and call `ferrors.FromDomain`. Do not add an internal/versioned interface method or a new execution method. Keep existing mutation result models and signatures unchanged.
  - In the helper, guard capability first; normalize every seed using the existing suspension validator, reject empty input, and check context before I/O. Copy options and append WithDryRun last for one local first-seed `SuspendProcessInstance` preflight before discovery. Validate waiting configuration even when the eventual eligible set is empty, retain existing NoWait timing behavior, and never treat this local preview as observed eligibility.
  - Build separate discovery options with explicit-key WithIgnoreTenant semantics and existing workflow/process-instance detail suppression. Preserve caller option/context ownership and independent DEBUG transport behavior. Do not leak forced preflight dry-run options into actual execution.
  - Use existing `DescendantsResult` per seed in normalized input order, not FamilyResult/ancestry/root cancellation planning. Traverse through ineligible parents; require complete descendant scope for the requested seed, inclusion of the seed, valid returned keys, and matching Chain identities. Missing observations/identities are malformed discovery; explicit partial/unresolved outcomes are planning-precondition failures. Any later-seed or context failure returns no executable plan.
  - Merge traversal-key order with first-observation classification; skip a seed already covered by an earlier complete traversal. ACTIVE is eligible, SUSPENDED is separately already_suspended, other observed states are excluded with unknown/unrecognized values normalized to UNKNOWN. Recheck context before publishing the complete plan. Do not perform race reconciliation, additional descendant paging, or backend mutation.
  - Keep public interface additions and every affected concrete implementation/test double together. At minimum update `cmd/process_api_stub_test.go` and `c8volt/resource/client_test.go`; give tree-specific interface-embedding fakes explicit planner callbacks. No unrelated production behavior changes are required.

  **Tests and completion evidence**:

  - Add/extend focused tests in `internal/services/processinstance/suspend_plan_test.go` and `c8volt/process/suspend_plan_test.go` using current suspension/traversal stubs. Cover guard-before-I/O on 8.8/8.9, all-input validation, first-seed local preflight, invalid timing before an empty eligible plan, NoWait timing, nested subtrees, mixed-state ancestors, child/ancestor overlap in both orders, deterministic ordering, and all-ineligible success.
  - Cover later-seed failure, forbidden/missing discovery, partial/unresolved traversal, missing seed/Chain record, mismatched identity, malformed descendant keys, planning cancellation, zero mutation, copy/alias ownership, option propagation and converted public errors. Exercise a successful plan followed by existing collection execution with only copied TargetKeys; retain existing collection retry/wait/evidence semantics rather than implementing another engine.
  - Use test-owned HTTP factory composition for public version guards and explicit tenant/discovery behavior so stubs cannot hide wrong adapter wiring. Keep existing `TestProcessInstanceSuspensionCollectionInput`, `CollectionFailures`, `Workers`, `FacadeMapping`, `FactoryGuards`, and `FactoryTargets` as relevant controls.
  - Run gofmt, then `env GOENV=off GOFLAGS= go test ./internal/services/processinstance/... -run 'ProcessInstanceSuspension|Descendants' -count=1` and `env GOENV=off GOFLAGS= go test ./c8volt/process -run 'ProcessInstanceSuspension' -count=1` after verifying matching tests. Compile all affected public-interface consumers together with `env GOENV=off GOFLAGS= go test ./... -run '^$'`; this checks interface impact and does not replace behavioral assertions. Follow the environment isolation rules before any command.

**Checkpoint**: Public planning and existing collection execution compose into an independently usable library workflow. The CLI --tree switch is not exposed until T002 completes its operator contract. US2 planning safety and US3 classification invariants already have service-level coverage here.

## Phase 4: User Story 2 — Review and approve the expanded scope (Priority: P1)

**Goal**: Deliver the opt-in CLI workflow from input through frozen plan, preview/no-op/approval, existing execution, and truthful output.

**Independent test**: Use the real command path against test-owned tree responses. Change the hierarchy/state before answering a real PTY approval prompt and assert that execution targets only the frozen active keys; failed planning, dry-run, abort, and successful no-active planning submit nothing.

- [ ] T002 [US2] Integrate and document the complete --tree command workflow in `cmd/suspend_processinstance.go`, `cmd/suspend_processinstance_tree.go`, and `cmd/cmd_views_processinstance_suspend_tree.go`, with output/activity/terminal regressions in `cmd/suspend_processinstance_*test.go`, metadata/help checks, `README.md`, and regenerated `docs/cli/` and `docs/index.md`.

  **Implementation and invariants**:

  - Register --tree as a default-false boolean with no shorthand and dispatch to a focused tree runner. Preserve exact-key dispatch, input validation, zero-discovery preview, mode validation, capability metadata, and existing flags. Put the distinct tree lifecycle in its mode file and final formatting in its view file; do not refactor global helpers.
  - Call the T001 planner with existing explicit-input options; handle any error before preview/no-op/approval. Preserve the returned plan, copy TargetKeys before approval, and derive all displayed scope counts only from that frozen plan. No command/facade traversal or classification loops and no post-approval rediscovery are allowed; deterministic view projection/counting of already-classified plan facts is permitted.
  - Follow the exact flow and message bodies in [contracts/cli.md](contracts/cli.md): tree dry-run renders succeeded preview, even with no-wait; actual zero-target execution renders succeeded no-op, even with no-wait. Both retain plan classifications with no mutation call or execution activity; dry-run/no-op never prompts. Do not reuse the ordinary batch runner's empty-preview shortcut for an actual tree no-op.
  - Show the compact human plan line where INFO output is enabled, then reuse the plain configured-stderr confirmation helper once with the unique eligible target count, preserving default-no/EOF/abort/automation/auto-confirm behavior. Decline performs no mutation. After approval, call existing `SuspendProcessInstances` once with copied frozen keys, existing worker controls, and ordinary execution options, not forced planner dry-run options.
  - Reuse the simple local indicatorEnabled/startCommandActivity wrapper with `suspending N process instance(s)`. Start only after approval, cover the full collection call including confirmation/backoff, and defer cleanup before final logging/rendering or unwinding. Preserve existing terminal, no-indicator, quiet, JSON, keys, automation, and structured-logging restrictions. Do not introduce a planning spinner, counters, completion events, or pool changes.
  - Add a typed tree-only payload with existing `items` plus `plan`; leave shared envelopes, existing SuspensionResults, and default exact-key schemas unchanged. Preserve the constraint “The extra plan member appears only for explicit tree mode.” Preview items represent eligible keys only with `status=preview`, `submission=not_submitted`, `confirmation=not_requested`, and no post-submission observation. Arrays are non-null; actual no-op has `items: []`. Never invent mutation outcomes for already-suspended or excluded members.
  - Retain per-target partial evidence and original exit classes. Select failure before no-wait acceptance; accepted envelope requires actual accepted submissions with no error. Render exactly one JSON envelope and do not invoke a second error renderer. Keys preview emits only eligible keys; actual execution emits only accepted keys, including accepted-but-unconfirmed failures with failure exit retained. Empty keys output is exactly zero bytes.
  - Human plan/scope/preview/outcome/verbose messages use configured INFO helpers, honoring logger level/format/destination/activity. Keep per-key facts verbose-only and use existing safe suspension outcome/count/failure formatters. Non-JSON partial failure emits one compact grouped configured ERROR before exitAfterRenderedResult; JSON gets no duplicate human failure. Machine-writer failures preserve any operation error's exit without another renderer. Audit every changed direct print and raw error formatting boundary; normal/verbose output must not expose transport chains, URLs/methods/bodies/cursors.

  **Tests, documentation, and completion evidence**:

  - Add a tree-specific extension of existing command fake-server/subprocess fixtures in `cmd/suspend_processinstance_tree_test.go` or nearby cohesive test files; retain the original exact-key fixture's rejection of discovery. Cover canonical/alias names, flags/stdin merges, nested/overlapping/mixed scopes, all-ineligible scopes, malformed/empty input, unsupported versions before discovery, failed later-seed/partial/unresolved planning, approval/abort, and state/hierarchy changes after planning.
  - Reuse/extend `cmd/suspend_processinstance_output_test.go` and `cmd/suspend_processinstance_logging_test.go` for human/verbose/quiet/JSON/keys and mode combinations, dry-run and actual no-op with/without no-wait, worker/fail-fast/interruption/partial confirmation outcomes, one envelope decoded with EOF, non-null arrays, original failure exits, accepted keys, and exact zero-byte empty keys. Assert no mutation facade call on preview/no-op, no discovery from rendering, and no newly accepted/confirmed claims for preexisting SUSPENDED members.
  - Through actual command execution, assert exact INFO/ERROR record framing/severity, configured/inherited destinations, plain-time/structured formats, INFO suppression at WARN/quiet, and compact repeated failures carrying distinctive fake HTTP markers. Check both streams for forbidden raw details, independent DEBUG via flag/configuration, quiet precedence, redaction, and no duplicates. Assertions must fail for a direct-print logger bypass or raw aggregate disclosure.
  - Extend `cmd/suspend_processinstance_terminal_test.go` using real PTY stdin and separate stdout/stderr capture for approval/decline/default/EOF, redirected output, configured/inherited destinations, supported human/keys/quiet/auto-confirm/automation/no-wait combinations, and prompt-free dry-run/failed planning/all-ineligible execution. Exercise actual empty-scope completion when interactive approval would otherwise be eligible; pipe-only or renderer-only checks are insufficient.
  - Extend `cmd/suspend_processinstance_activity_test.go` or a focused tree activity test for no execution scope during planning/approval/preview/no-op, one scope throughout approved work/waits, cleanup before output and unwinding, and protected-mode suppression. Reuse existing worker barriers and cancellation/timeout regressions rather than adding an execution engine to fixtures. Keep Cobra/global-flag tests nonparallel unless subprocess-isolated.
  - Update `TestSuspendProcessInstanceMetadata` in `cmd/command_contract_test.go` and run the existing `TestAllCommandHelpFocusesOnActions` gate in `cmd/command_help_focus_test.go`. Help must explain operator actions, scope, discovery permissions, unchanged default behavior, the retained limit, state races and non-atomic execution; do not insert stdout/stderr/logger implementation vocabulary into Short/Long help. Put routing detail in README/contracts.
  - Update README and source examples for --tree, --tree --dry-run, mixed-state exclusions, frozen counts, permissions, up to 1,000 direct children per parent lookup, and possible omission of future descendants. Regenerate via `env GOENV=off GOFLAGS= make docs-content`; never hand-edit generated CLI files. Add user-facing public planner guidance/compatibility note to README alongside the feature.
  - Run gofmt, targeted `env GOENV=off GOFLAGS= go test ./cmd -run '^(TestSuspendProcessInstance|TestAllCommandHelpFocusesOnActions)' -count=1`, then the targeted race variant with `GORACE='atexit_sleep_ms=0'` for concurrent fixtures. Run `env GOENV=off GOFLAGS= go test ./docsgen -count=1` after generation and `git diff --check`. Use [quickstart.md](quickstart.md) for fixture isolation and evidence; broaden only for actual uncovered impact/failures, not merely task/commit completion. Report any unperformed live behavior as unverified.

**Checkpoint**: The complete operator feature is usable with frozen approval scope, existing mutation execution, accepted-versus-confirmed evidence, script-safe output, appropriate activity, and matching documentation. US3/US4 are included in this result, not deferred quality work.

## Phase 5: User Story 3 — Retain truthful collection outcomes (Priority: P2)

**Goal**: Preserve acceptance, confirmation, failure, interruption, and unattempted evidence across the tree collection.

**Owners**: T001 owns mixed-state eligibility and frozen-target planning; T002 owns command integration and observable results. No additional checkbox is needed because the existing executor is reused and the same tree view/fixtures cover all collection outcomes.

**Independent test**: Run a mixed accepted/rejected/accepted-but-unconfirmed/fail-fast collection using isolated HTTP responses. Assert only frozen active targets mutate, every eligible target retains its disposition, keys indicate acceptance, one envelope contains partial evidence, required error/exit survives, and human diagnostic/activity routing remains correct. The no-wait case still performs discovery but no confirmation reads.

## Phase 6: User Story 4 — Understand permissions and practical limits (Priority: P2)

**Goal**: Make scope, permission, retained traversal-limit, and non-atomic behavior clear before use.

**Owners**: T001 provides guard-before-I/O and explicit-key discovery behavior; T002 owns all user-facing help/README/generated docs and their checks. No separate documentation polish task is required.

**Independent test**: Review generated command help/docs against the scenarios, verify metadata and action-focused help checks, and exercise 8.8/8.9 factory guards without discovery. Documentation must distinguish discovery reads from no-wait confirmation reads, selected child scope from root escalation, preexisting states from new mutation success, and 1,000 direct children per lookup from a whole-tree cap.

## Phase 7: Polish & Cross-Cutting Concerns

No standalone polish task. Each owner must leave its outcome compiling, formatted, documented where user-facing, and validated with the smallest relevant checks. No incomplete test-double repair, error-output correction, documentation regeneration, or progress housekeeping is deferred to this phase.

## Dependencies & Execution Order

```text
Existing repository/fixtures
        |
T001 [US1]: complete frozen planner + public contract + stubs/tests
        |
T002 [US2]: CLI approval/execution/views + shared US3/US4 acceptance + docs/tests
```

- Setup/foundation introduce no independent work. T001 must complete before T002 uses its public model/method.
- US1 has a usable library flow after T001 because the existing collection executor already accepts its frozen targets. CLI acceptance for US1 is exercised with T002.
- US2 depends on US1's complete plan; US3 shares T001/T002 and US4 shares their guards plus T002 documentation. Story labels identify primary ownership, not repeated implementation boundaries.
- Task count: 2. Primary ownership: US1 = 1, US2 = 1; US3 = 0 additional (T001/T002 coverage), US4 = 0 additional (T001/T002 coverage). No task is started or complete in this draft.

## Parallel Opportunities and Examples

No task-level parallel opportunities: T002 consumes T001's new public contract, and each unit includes its implementation, necessary stubs/fixtures, tests, and formatting. No [P] marker is justified.

- **US1**: A read-only review of planner overlap/classification cases can occur independently from a review of copy/error conversion cases after the contract is agreed. These remain T001 responsibilities, not separate tasks or an instruction to launch agents.
- **US2**: After T001 is complete, preview/no-op/envelope assertions and PTY approval assertions can be reviewed independently against the same contract. Execute shared Cobra/global-state tests serially unless subprocess-isolated.
- **US3**: Service worker/waiter behavior and command evidence can be reviewed separately using existing fixtures; do not parallel-edit the same tree renderer or rebuild the executor.
- **US4**: README/help prose and generated-doc consistency can be reviewed independently once behavior is fixed. Regeneration follows source edits and remains part of T002.

## Implementation Strategy

1. Complete T001 as a compiling service/facade planning increment. Its frozen keys compose with the existing executor; no provisional CLI flag is exposed.
2. Complete T002 once for all operator modes. Validate each story's acceptance through its owning fixture layer; do not add a single-target tree path first and rebuild it for collections later.
3. The smallest safe user-facing MVP is T001 + T002: explicit tree selection requires truthful planning, preview/approval/no-op handling, output, and documentation together. The T001 library capability is a useful intermediate deliverable.
4. Preserve original exact-key tests and untouched walk/cancel/delete behavior. Do not add live checks, global frameworks, pagination changes, or an automatic repeated-discovery loop.

## Requirement Ownership

| Requirement | Owning task(s) and evidence |
| --- | --- |
| FR-001 | T002 flag, alias, explicit input and metadata/help. |
| FR-002 | T001 descendant-only scope; T002 child-versus-root execution evidence. |
| FR-003 | T001 stable overlap deduplication; T002 one mutation per frozen eligible key. |
| FR-004 | T001 complete-plan/error boundary; T002 fail-before-approval/mutation. |
| FR-005 | T001 mixed-state traversal/classification; T002 classified reporting and zero ineligible mutation. |
| FR-006 | T001 advisory plan; T002 state-race/acceptance evidence through existing executor. |
| FR-007 | T002 tree dry-run reads, preview payload, zero writes/prompts. |
| FR-008 | T001 frozen targets; T002 one expanded-count approval and abort behavior. |
| FR-009 | T002 copies approved keys/counts and never rediscovers. |
| FR-010 | T002 actual no-op across modes, items array empty, no prompt/activity/call. |
| FR-011 | T001 public input contract; T002 command invalid/empty input before discovery. |
| FR-012 | T002 existing collection worker/fail-fast/interruption controls and evidence. |
| FR-013 | T002 existing bounded confirmation/no-wait semantics and read counts. |
| FR-014 | T002 partial evidence, original failure exits and no false tree success. |
| FR-015 | T001 planner-local suppression; T002 configured INFO/ERROR/DEBUG routing matrix. |
| FR-016 | T002 plain prompts, precedence, one envelope and preserved machine evidence. |
| FR-017 | T002 eligible-preview/accepted-execution keys and zero-byte no-op. |
| FR-018 | T002 simple approved execution activity, cleanup and mode suppression. |
| FR-019 | T001 capability/factory guards; T002 unsupported tree/dry-run before discovery. |
| FR-020 | T001 unchanged executor contract; T002 exact-key request/schema controls. |
| FR-021 | T002 source help, README, metadata and generated docs. |
| FR-022 | T001 retained traversal limit; T002 documented permissions/limits/non-atomic scope. |
| FR-023 | T001/T002 minimal extension and relevant unchanged-neighbor controls. |

| Success criterion | Evidence owner |
| --- | --- |
| SC-001 | T001 scope projection and T002 exact mutation keys. |
| SC-002 | T001 failed-planning guards and T002 preview/abort/invalid/no-op request counts. |
| SC-003 | T002 real approval race with frozen keys/counts. |
| SC-004 | T001 classified scope and T002 final per-target evidence. |
| SC-005 | T002 existing waiter/no-wait integration controls. |
| SC-006 | T002 logger/PTY/envelope/keys/activity matrix. |
| SC-007 | T001 unchanged adapters/traversal and T002 exact-key/neighbor regression selection. |
| SC-008 | T002 operator-facing examples, permission and scope/limit documentation. |
