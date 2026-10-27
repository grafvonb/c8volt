# Tasks: Resume Process Instances

**Input**: Design artifacts in `specs/347-resume-process-instances/`.
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [CLI contract](contracts/cli.md), [service contract](contracts/service-api.md), [quickstart.md](quickstart.md).
**Branch / issue**: `347-resume-process-instances` / #347.
**Tests**: Required by the specification. Every checkbox includes implementation, necessary fixtures, tests, formatting, and targeted validation; no separate test-only completion phase.
**Organization**: Shared prerequisites precede story-specific delivery. US3 and US4 acceptance is implemented in the shared and command work units that own it; their phases map coverage instead of adding duplicate implementations or late hardening tasks. One checkbox is one compiling, verifiable Ralph work unit, not permission to cross into another story's tasks.

## Format and execution rules

- Checkbox lines use `TNNN`, optional `[P]`, and `[USn]` only for story-specific work. Paths are repository-relative; newly proposed files are intentional.
- Before each unit, read repository guidance and this feature's artifacts; use `specs/ralph-implementation-rules.md`. Context reading, branch checks, review, and progress recording are responsibilities, not tasks.
- Write/update the meaningful regression assertions with their owning behavior, preferably establish their failing behavior before implementing, and finish the unit only after its checks pass. Do not leave unimplemented interface methods or broken test doubles for a later unit.
- Follow the sanitized environment and fixture checks in `quickstart.md`. Remove inherited `C8VOLT_*`; use `GOFLAGS=''`, `GOENV=off`, and `GOWORK=off`. Verify selected tests exist and use only test-owned fakes before execution. No integration build tags, live profiles, clusters, deployment, smoke runs, or live cleanup. PTY skips are coverage gaps.
- Run `gofmt` on touched Go files; add required purpose/doc comments to new or materially changed declarations and tests. Update source help, README, and generated CLI docs with each command behavior change through `make docs-content`.
- Keep backend planning, workers, retries, and waiting in internal services. Keep facade conversion thin and command construction, batch/tree runners, and views in their named files. Preserve suspension/cancel/delete behavior. No generated-client edits, generic mutation framework, extra search mode, pagination expansion, rollback, or rediscovery.
- Preserve existing partial-result/error classes and exits. Human information/diagnostics use configured logging, selected machine results use stdout, and prompts use plain configured stderr. Trace service → facade → rendering → logging → exit for each new failure path.
- Record actual checks and limitations in `specs/347-resume-process-instances/progress.md` during implementation. If the execution workflow reaches a commit step, use Conventional Commits ending in ` #347`. This task-generation command does not launch or implement Ralph.

## Phase 1: Setup (Shared Infrastructure)

No setup checkbox is necessary: the repository, dependencies, generated resumption client, and test infrastructure already exist. Reuse them. No new project scaffolding, dependency installation, or live-target preparation is required.

## Phase 2: Foundational (Shared Resumption Behavior)

**Goal**: Deliver complete submission/confirmation and ordered collection APIs before exposing commands. This core supplies US1, US2, US3, and US4.
**Independent test**: Isolated reader/adapter tests prove state and protocol semantics; collection/facade tests prove ordering, controls, and evidence without CLI involvement.

- [ ] T001 Deliver the complete versioned resumption lifecycle, including resumption evidence and the bounded read-only confirmation primitive in `internal/domain/processinstance_resumption.go`, `internal/services/processinstance/waiter/resumption.go`, and `internal/services/processinstance/waiter/resumption_test.go`, including meaningful model serialization assertions in `internal/domain/processinstance_resumption_test.go`; enforce the quoted model constraints and observation matrix below, run the focused waiter/domain checks, and format touched files.

  **Model constraints (verbatim from data-model.md):** Key is “Normalized execution target”; Status is “Workflow disposition, not engine state”; Submission is “Cumulative submission evidence”; Confirmation is “Observation evidence independent of submission”; StatusCode is “Last valid submission HTTP status, never read status”; HasObservation “Distinguishes no read evidence from observed UNKNOWN”; LastObservedState is “Last valid matching observation; absent when none”; SuspendedDate is “Latest successful observation's supplied date; no synthesis”; Error is “Machine evidence; never rendered directly as human diagnostics”. Preserve `omitempty` on `statusCode`, `lastObservedState`, `suspendedDate`, and `error`, but not `hasObservation`. Enum values are exactly Status: `preview`, `accepted`, `confirmed`, `failed`, `timed_out`, `interrupted`, `not_attempted`; Submission: `not_submitted`, `accepted`, `rejected`, `unknown`; Confirmation: `not_requested`, `unconfirmed`, `confirmed`.

  **Acceptance:** The waiter exposes only reads; caller acceptance is a precondition, never inferred by the waiter. Matching ACTIVE or immediate COMPLETED succeeds; CANCELED/TERMINATED stops with conflict. SUSPENDED/ABSENT/UNKNOWN continue within timeout/attempt bounds. Verify authorization/precondition/read failures, cancellation, deadline/attempt limit, wrong identity, and reads returning after deadline. Not-found records ABSENT; other unsuccessful reads preserve prior observations. Every successful observation replaces date evidence, including omission clearing stale dates. No mutation calls or manufactured timestamps. Existing suspension waiter semantics remain unchanged.

  **Validation:** `go test ./internal/services/processinstance/waiter -run 'Resumption' -count=1`; `go test ./internal/domain -run 'Resumption' -count=1`. Create matching tests before relying on these patterns. This internal primitive does not change command documentation.

  **Adapter and interface integration in this work unit:** Implement the complete versioned resumption operation and guards in `internal/services/processinstance/v810/resume.go`, `internal/services/processinstance/v810/resumption_submission.go`, `internal/services/processinstance/v88/resume.go`, and `internal/services/processinstance/v89/resume.go`; extend `internal/services/processinstance/api.go` and all three `contract.go` files, narrowly reuse key normalization in `internal/services/common/processinstance_keys.go`, and include required adapter implementations/test-double updates plus `v810/resume_test.go`, `v810/resumption_submission_test.go`, `v88/resume_test.go`, `v89/resume_test.go`, and normalization regression tests in the same compiling unit.

  **Acceptance:** Use the existing generated `ResumeProcessInstanceWithResponse` and empty typed body at the single-instance resumption path. Guard 8.8/8.9 before unsupported I/O, including preview. Normalize trimmed exactly 16-digit decimal keys without hierarchy reads. Validate timing before requests, preserving no-wait exemptions. Exact dry-run is local; no-wait skips all confirmation reads. Valid empty 204 establishes acceptance; preserve explicit rejection for active/terminal/terminating/missing/unauthorized targets. Retry only via established mutation policy and request-rate attempt tracking. Ambiguous response/transport evidence persists across later rejection; local admission failure preserves prior rejected/unknown evidence and last valid status. Only valid acceptance may resolve ambiguity. Compose this unit's waiter after acceptance and preserve results alongside original errors; cancellation is failed/accepted/unconfirmed with cancellation state and conflict. Never use later state to repair rejected or ambiguous submission.

  **Compilation and fixtures:** Find every impacted implementation/stub with `rg`; include relevant stubs in `c8volt/process/*_test.go`, `c8volt/resource/client_test.go`, `internal/services/processdefinition/delete_test.go`, `internal/services/ops/orphan_purge_test.go`, process-instance dry-run and versioned tests. Do not defer these repairs. Preserve the current suspension normalizer's behavior and callers during the narrow extraction; do not generalize mutation execution.

  **Validation:** `go test ./internal/services/processinstance/... -run 'Resum|Suspend|Suspension' -count=1`; `go test ./internal/services/common -run 'NormalizeProcessInstance' -count=1`, using actual matching test names. Check affected stub consumer packages using their relevant isolated tests; an untagged `go test <affected-packages> -run '^$'` may check compilation but is not behavioral evidence. Protocol tests cover strict response validation, retry ambiguity, rate admission, original status, no unwanted reads, and waiter errors. Format all touched files; no public command behavior yet.

- [ ] T002 Deliver one ordered collection workflow and its thin public singular/collection APIs in `internal/services/processinstance/resume.go`, `internal/services/processinstance/resume_test.go`, `c8volt/process/resume.go`, `c8volt/process/resume_test.go`, `c8volt/process/api.go`, `c8volt/process/model.go`, and `c8volt/process/convert.go`, updating all affected public-interface test doubles together; depend on T001 and include conversion/worker/partial-result tests and formatting.

  **Acceptance:** Use existing worker policy and `pool.ExecuteSlice`; validate all normalized keys and local timing before any dispatch, initialize unattempted slots, and preserve first-input order regardless of completion order. Both public singular and multiple-key methods delegate to this collection; the singular facade only wraps/extracts and converts. Preserve duplicates only as unique identities, but keep explicitly selected parent and child independent. Propagate dry-run/no-wait/fail-fast/no-worker-limit options and wanted workers without facade backend mechanics. Known outcomes survive fail-fast/interruption, including in-flight work and unattempted slots. Return partial results and `ferrors.FromDomain(err)` together. No raw joined-error logging.

  **Model constraints:** Public fields and enums preserve T001's quoted meanings without internal/generated types; copy slices at conversion boundaries. “Exactly one slot per normalized execution target, in stable order, survives fail-fast or interruption.” “Counts overlap deliberately: accepted-but-unconfirmed failures count as both accepted and failed.” `items` is an initialized array in rendered results; internal empty collection may succeed empty, but command empty input and public singular empty key remain invalid. Test copy boundaries, enum/omission mapping, singleton delegation, factory version guards, all-key prevalidation, worker limits, no-worker-limit, fail-fast, mixed confirmed/rejected/unknown/unconfirmed outcomes, and no confirmation reads under no-wait.

  **Validation:** `go test ./internal/services/processinstance -run 'ResumeProcessInstances' -count=1`; `go test ./c8volt/process -run 'Resum' -count=1`; targeted suspension regressions for shared changes and affected public-stub compilation. Document exported library semantics in Go doc comments; CLI/README examples wait until command delivery.

**Checkpoint**: Library resumption supports complete collections with truthful bounded confirmation and all controls; no temporary single-only execution implementation is required.

## Phase 3: User Story 1 — Resume Explicitly Selected Instances (P1, MVP)

**Goal**: Expose the full exact-key command using the complete shared workflow.
**Independent test**: Fake-backed command execution targets exactly unique selected keys and produces correct preview, approval, accepted/confirmed/failure output, and exits in every supported mode.

- [ ] T003 [US1] Deliver the exact-key command, selected-format views, and complete command acceptance coverage in `cmd/resume.go`, `cmd/resume_processinstance.go`, `cmd/resume_processinstance_batch.go`, `cmd/cmd_views_processinstance_resume.go`, `cmd/resume_processinstance_test.go`, `cmd/resume_processinstance_output_test.go`, `cmd/resume_processinstance_logging_test.go`, `cmd/resume_processinstance_terminal_test.go`, `cmd/resume_processinstance_activity_test.go`, and `cmd/command_contract_test.go`; update `README.md`, command help/metadata, and regenerate `docs/cli/*`/`docs/index.md` with `make docs-content`; depend on T002.

  **Acceptance:** Wire process-instance/pi, repeated/comma-separated keys and stdin `-`, worker controls, fail-fast, preview, confirmation/automation, and bounded/no-wait options. Reject malformed/missing/empty stdin before work. Preserve JSON precedence, JSON approval prerequisite, JSON/verbose exclusion, and quiet machine results. Freeze preview keys, approve once with existing default/EOF/abort behavior, then call the collection without discovery. Start shared activity after approval and stop before rendering on every exit path. Display acceptance separately from ACTIVE/COMPLETED confirmation; cancellation is a distinct bounded failure. Do not expose `--tree` until T005 provides its complete lifecycle.

  **Required command tests, included here:** Capture stdout/stderr and configured/inherited logger destinations separately. Human mutation/preview/verbose output must be INFO records, absent at WARN/quiet, with empty human stdout, across plain and structured formats. Use repeated backend errors with distinctive URL/method/body/key/secret markers: one grouped bounded ERROR, no raw details on either human stream at normal/verbose/quiet, DEBUG flag and configured debug independently visible with redaction, and no duplicate diagnostics. JSON preserves raw machine evidence and partial results in exactly one envelope followed by EOF and original failure exit. Keys-only previews planned keys and executions accepted keys, including accepted-but-unconfirmed; zero qualifying keys emits zero bytes. Test result writer failure preserving operation-error precedence. Assertions must detect direct-print logger bypass and raw-aggregate leakage. Use real terminal stdin with redirected stdout and configured/inherited stderr to verify plain prompts, default/EOF/decline/auto-confirm behavior. Test all accepted/confirmed/unconfirmed/immediate-completed/canceled states, both no-wait and waiting, empty/invalid input, collection controls, and activity eligibility/cleanup. Reuse isolated fixtures; do not reproduce every matrix separately for each state.

  **Validation:** `go test ./cmd -run '^TestResumeProcessInstance' -count=1`, with meaningful matching test names and isolated fake configuration; targeted command-contract/help tests matching existing names. Include a fake-backed resume → fresh get/expect regression proving actual resumed state and omission of stale `suspendedDate` without a live engine. Run `gofmt`, `make docs-content`, and review generated output and changed direct prints/raw error formatting. This unit owns exact-key US3/US4 correctness; they are not deferred polish.

  **Integration source and guidance in this work unit:** Update intended CLI resumption coverage and its documentation in `integration/cli/get_processinstance_suspended_test.go`, `integration/cli/suspended_fixture_test.go`, and `integration/README.md`, replacing the scenario's direct resume transition with `resume pi` plus envelope/evidence assertions and correcting the obsolete “no public resume workflow” fallback record.

  **Acceptance:** Preserve owned-key checks, tracking before mutation, independent observation/cleanup evidence, and the existing post-resumption state/date-omission assertions. CLI coverage asserts accepted plus confirmed resumption before fresh reads; direct setup/cleanup calls may remain only where intentionally outside CLI coverage and documented as such. Keep existing integration build tags and do not alter unrelated suspension/cancel/delete scenarios or `integration/scripts/run-c810-suspended.sh`, which has no resumption scenario. All general operator guidance is already required in T003/T005: eligibility/version, permissions, independent relatives, buffer/failure meaning, no incident-free/completion/readiness promise, tree limit/frozen scope/no rollback, and acceptance versus observation.

  **Validation:** `gofmt` touched Go fixture source, `git diff --check`, inspect command arguments/result assertions and preserved ownership/cleanup paths, and verify changed documentation links. Do not execute, provision for, or require integration/smoke/live validation. Record real-engine behavior as unverified. This source-only change does not justify rerunning application tests or the full race suite.

**Checkpoint / MVP**: T001–T003 deliver a usable exact-key command with confirmation, collection controls, safe diagnostics, docs, and isolated evidence. Tree mode remains undisclosed until complete.

## Phase 4: User Story 2 — Preview and Resume a Selected Subtree (P1)

**Goal**: Resolve and approve a complete frozen eligible subtree scope, then reuse collection execution.
**Independent test**: Nested/overlapping mixed-state trees produce one ordered plan, no ancestor/sibling escalation, one approval, no rediscovery, and prompt-free empty-scope completion.

- [ ] T004 [US2] Implement complete tree planning with thin public mapping in `internal/domain/processinstance_resumption_plan.go`, `internal/services/processinstance/resume_plan.go`, `internal/services/processinstance/resume_plan_test.go`, `c8volt/process/resume_plan.go`, `c8volt/process/suspend_plan_test.go` where shared fixtures require updates, `c8volt/process/resume_plan_test.go`, and public `api.go`, `model.go`, `convert.go`; narrowly share existing structural traversal validation only if identical, include affected stubs and regression coverage, and depend on T002.

  **Model constraints (verbatim):** RequestedKeys: “Trimmed 16-digit unique seeds in first-input order; preserve overlapping explicit seeds”. Instances: “Unique discovered members in first-seen traversal order, including seeds”. TargetKeys: “Exactly the ordered projection of eligible `Instances`; frozen before approval”. Classification `eligible`: “Observed SUSPENDED during complete discovery”; `excluded`: “Every other state; preserve known state, normalize unrecognized values to UNKNOWN”. “Successful plans initialize all arrays, including empty target arrays. Failure returns zero plan plus error and is never a successful empty plan.” Preserve each item's `key`, `observedState`, and `classification`; copy arrays mechanically at the facade boundary.

  **Acceptance:** Capability and local input/timing guards precede discovery. Reuse `DescendantsResult`, explicit-key ignore-tenant/backend authorization, and suppression options; traverse every state before eligibility filtering. Include child subtrees without ancestors/siblings; merge overlapping seeds without losing explicit requested-key evidence. Independently suspended descendants are eligible. Validate complete outcome, mode, seed/root identity, missing ancestry, seed presence, discovered key syntax, and matching observations, including overlaps. Any failed/partial/unresolved/malformed discovery aborts the entire plan without mutation. Preserve the 1,000-direct-children lookup bound, no pagination guarantee, no pruning on empty filtered intermediate results, and no resumption call other than a strictly local validation preview if using the existing pattern. Empty target arrays are valid only after authoritative complete planning.

  **Validation:** `go test ./internal/services/processinstance -run 'PlanProcessInstanceResumption|PlanProcessInstanceSuspension' -count=1`; `go test ./c8volt/process -run 'ResumptionPlan|PlanProcessInstanceResumption' -count=1`, choosing actual test names. Include nested/mixed/overlap/child-only/unknown/missing/denied/interrupted/partial/malformed scopes, no-I/O version guards, all-seed failure, ordering, copied slices, and unchanged suspension planning. Format touched files. Command docs remain with T005 because this unit adds library planning only.

- [ ] T005 [US2] Expose the complete tree lifecycle and views in `cmd/resume_processinstance.go`, `cmd/resume_processinstance_tree.go`, `cmd/cmd_views_processinstance_resume_tree.go`, `cmd/resume_processinstance_tree_test.go`, `cmd/resume_processinstance_tree_failure_test.go`, `cmd/resume_processinstance_tree_output_test.go`, `cmd/resume_processinstance_tree_terminal_test.go`, and shared resumption logging/activity tests; update command metadata/help, `README.md`, and regenerate `docs/cli/*`/`docs/index.md`; depend on T003 and T004.

  **Acceptance:** Plan through the facade, clone eligible keys before approval, ask once for the expanded count, execute exactly those keys through T002, and never rediscover. Dry-run displays selected/discovered/eligible/excluded scope without mutation or prompt. Successful zero eligible scope returns before approval/activity/collection submission, even with no-wait; do not classify incomplete planning, missing reports, or aborts as no-ops. Advisory state races preserve authoritative rejection. Normal execution retains planned classifications and partial collection evidence; excluded members receive no mutations. Render tree payload with initialized `items`, `plan.requestedKeys`, `plan.instances`, `plan.targetKeys`; never claim an already-resumed success. No-wait envelope is accepted only when a mutation was actually accepted and no operation error exists.

  **Required command tests, included here:** Extend T003's diagnostic/INFO suppression/DEBUG/one-envelope/original-exit matrix to tree planning and execution failures using shared fixtures, retaining raw machine evidence. Exhaustively cover resolved-empty scope across normal/dry-run, no-wait, human/JSON/keys-only/quiet, quiet-plus-machine, and supported auto-confirm/automation combinations; require exact human wording, EOF after one successful envelope, zero-byte keys, fixed discovery counts, no confirmation or mutation, and no extra requests from rendering. Use real terminal stdin for empty-scope prompt-free proof and nonempty frozen-scope approval, with redirected stdout and configured/inherited stderr. Retain nonempty mixed-state/overlap/child-only, failed discovery, declined approval, post-planning state changes, partial failure, and exact-key regressions. Ensure all-ineligible intermediate members do not hide eligible descendants. Activity still starts after approval and stops before final output.

  **Validation:** `go test ./cmd -run '^TestResumeProcessInstance' -count=1`; targeted suspension-tree/confirmation/output regressions for reused helpers. Format and regenerate docs; inspect help and generated examples for exact/tree scope, read/search permissions, independently suspended descendants, frozen/non-atomic scope, lookup bound, no rollback, and wait/no-wait. After relevant slices pass, run sanitized `make test` once for the completed shared interface/concurrency change if its actual blast radius warrants it; verify no live tags/helpers, record actual results, and do not rerun solely for later documentation/source-only fixture edits.

**Checkpoint**: Exact-key and tree workflows are both complete with the same executor and confirmation semantics.

## Phase 5: User Story 3 — Distinguish Acceptance from Observed Resumption (P1)

**Goal**: Prove accepted submission plus ACTIVE/immediate COMPLETED, distinct cancellation, bounded uncertainty, and fresh date mapping.
**Independent test**: Run the isolated resumption waiter/adapter/facade suites and fake-backed command resume/get/expect scenario; inspect all state/evidence outcomes independently of hierarchy selection.

No additional checkbox: T001 owns the observation matrix/date replacement plus submission/retry evidence and wait/no-wait composition, T002 owns evidence-preserving conversion, T003 owns exact-key end-to-end output/exit/fresh-read coverage, and T005 proves the same semantics under tree scope. US3 must pass in those owning units; this phase is an acceptance index, not a delayed implementation or test gate. The historical integration source adjustment is included in T003 and cannot substitute for isolated coverage.

## Phase 6: User Story 4 — Reliable Collection Controls and Output (P2)

**Goal**: Preserve ordered partial results, controls, configured logging, uncontaminated machine output, prompts, and activity.
**Independent test**: Run isolated collection worker/fail-fast cases and actual command/PTY output matrices, including repeated errors and partial-result exits.

No additional checkbox: T002 implements collections and controls, T003 implements full exact-key output/diagnostics/PTY/activity, and T005 covers tree/empty-scope variants. These are mandatory acceptance obligations of each delivered mode. Do not add a later logger-routing repair task or move command/terminal assertions into renderer-only tests.

## Phase 7: User Story 5 — Understand Permissions and Operational Limits (P2)

**Goal**: Keep user documentation and intended CLI integration scenarios consistent with delivered resumption.
**Independent test**: Review generated help/README/contract examples and source fixture ownership/cleanup/evidence; do not execute live fixtures.

No additional checkbox: T003 includes intended CLI integration resumption source, fixture evidence, and integration guidance alongside exact-key command delivery; T003/T005 each include their command documentation and generated docs. US5 is complete only when both owning command units meet that documentation contract. This does not authorize running integration fixtures.

## Phase 8: Polish and Cross-Cutting Completion

No separate checkbox: formatting, comments, fixtures, documentation, output review, and targeted tests belong to T001–T005. Preserve already recorded successful checks; only rerun checks invalidated by relevant changes. All specification obligations and the actual validation limitations must be recorded before feature completion. Do not add live checks as a final gate.

## Dependencies and Execution Order

```text
T001 → T002 → T003 (exact-key MVP + fixture source / guidance)
           └→ T004 ─┐
              T003 ─┴→ T005 (tree command)
```

Sequential execution order is T001 through T005. T004's prerequisite is T002; T005 needs both T003/T004. US1 is independently usable after T003; US2 builds on that collection and command convention. US3 is satisfied as shared prerequisites and each command mode land; US4 is included in the collection/command completion gates. US5's general docs ship with each mode and fixture guidance ships in T003. No cross-story obligation is postponed to make an earlier checkbox appear complete.

### Parallel examples by story

| Story | Execution guidance |
| --- | --- |
| US1 | Keep T002→T003 serial: public API/fakes, command behavior, fixtures, and generated docs must agree. No parallel split between implementation and required tests. |
| US2 | Keep T004→T005 serial. Although T003 and T004 share a prerequisite, adding the public plan interface can require changes to command test doubles being written in T003; do not promise conflict-free parallel execution. |
| US3 | No standalone parallel unit: state-sequence fixtures stay with T001 and the command modes. |
| US4 | No standalone parallel unit: diagnostics and activity share the owning command fixtures and must be validated with that mode. |
| US5 | No standalone parallel unit: command docs and intended CLI fixture source are part of command delivery, not a later handoff. |

No task is marked `[P]`; no independently schedulable work unit is established without shared-contract or fixture overlap. The implementation may use existing safe isolated-test concurrency, but this artifact does not invent tasks or launch agents merely for parallelism.

## Requirement Ownership and Independent Acceptance

| Requirements | Owning tasks | Required acceptance |
| --- | --- | --- |
| FR-001–FR-004 | T001–T003 | Key forms/validation, guards including preview, exact-key scope/rejection, parent/child preserved |
| FR-005–FR-008 | T004–T005, T001 | Traverse before filter, overlap/identity/completeness, frozen eligible keys, state races |
| FR-009–FR-011 | T001–T005 | Preview/local guards, no-op vs invalid/error/abort, one prompt, automation |
| FR-012–FR-015 | T001–T003, T005 | Acceptance/ambiguity, ACTIVE or immediate COMPLETED, distinct cancellation, bounded waiting, no-wait/no resubmission |
| FR-016 | T002–T003, T005 | Order/workers/fail-fast/in-flight/unattempted/partial evidence and original exits |
| FR-017–FR-021 | T003, T005 | JSON once + EOF, keys, configured INFO/ERROR, compact diagnostics/redacted DEBUG, real PTY prompts/activity |
| FR-022 | T001, T003 | Replace fresh date evidence, fake-backed get/expect omission, corresponding fixture source |
| FR-023–FR-025 | T003, T005 | Help/metadata/README/generated docs, permissions/limits/outcome meaning, integration guidance |
| FR-026 | T003 | Intended CLI resumption source updated; no live run |
| FR-027 | All, especially T001/T004/T005 | No scope expansion or unrelated behavior change; closest suspension/cancel/delete/traversal regressions |

SC-001/SC-005 map to T001–T003/T005; SC-002–SC-004 to T001–T005; SC-006 to T001–T003/T005; SC-007 to T003/T005; SC-008 to T003/T005; SC-009 to T001/T004/T005/T003. The model/CLI/service contracts are mandatory acceptance detail for these owners, not optional background.

## Implementation Strategy

1. Complete shared confirmation, protocol, and collection behavior once (T001–T002). Keep new interfaces/implementations/stubs atomic and facades thin.
2. Deliver the documented exact-key MVP with full output/terminal/activity checks (T003). This includes the necessary US3/US4 semantics, not an acceptance-only temporary command.
3. Deliver complete tree planning then its CLI interaction (T004–T005), preserving the same executor and output rules.
4. Confirm source guidance delivered with T003/T005 stays aligned; do not run live workflows. All task checkboxes remain unchecked until actual implementation and their permitted validation are complete.
