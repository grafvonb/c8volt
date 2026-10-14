# Tasks: Extend Release Validation to Camunda 8.10

**Input**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [validation contract](contracts/validation.md), and [quickstart.md](quickstart.md).

**Issue / branch**: #338 / `338-extend-810-validation`.

**Organization**: Nine implementation units and one live acceptance unit. Each implementation unit includes its tests, formatting, and affected documentation. Existing helpers take precedence over proposed new files; no standalone setup or housekeeping tasks are needed.

**Execution boundary**: This task list does not authorize a live run now. Implement and validate offline first. Live acceptance requires an explicitly selected disposable environment and execution authorization. A standard release run needs only one profile and one version environment:

```sh
C8VOLT_IT_PROFILES=dev810 C8VOLT_IT_RELEASE_VERSION=8.10 make integration-test-all
```

No simultaneous 8.9/8.10 deployments are required. Older-version runs may use a later redeployment or another machine. Optional tenant coverage must remain a named gap when unavailable; explicitly required tenant coverage must fail when unavailable.

## Phase 1: Foundational — shared coverage policy

**Goal**: A deterministic coverage contract reusable by runner and tagged harness without invoking live tests.

- [x] T001 Implement request, target, requirement, gap, slice, evidence, and summary records plus validation/reduction in `integration/internal/validation/model.go`, `request.go`, `catalog.go`, and `reduce.go`, with adjacent tests and checked-in `integration/assets/release-coverage.json` and `release-gaps.json`. Enumerate real existing scenario IDs, applicability, family minima, and accepted additional gaps; use all normative constraints quoted below. Map actual baseline/volume families and seven real-state families, including suffixed tests, without inventing successful evidence. Cover canonical minor parsing, absent/mismatched targets, whole-family skips, required proof mismatches, unknown/duplicate/stale evidence, interruption, child failure, gap policy, secret redaction, and safe path validation. Keep catalog data accessible consistently from arbitrary working directories. Validate with `go test ./integration/internal/validation -race -count=1`; document strict/exploratory verdicts in `integration/README.md`.

**Checkpoint**: Pure policy tests pass without building or executing the product CLI or contacting a cluster.

## Phase 2: US1 — Trust the requested release validation result (P1)

**Goal**: A single selected environment has complete, isolated, auditable release evidence.

**Independent test**: Fake subprocesses demonstrate preflight-before-mutation, missing evidence rejection, preserved failures, and interrupt finalization. A requested 8.10 release with only 8.9 evidence cannot pass.

- [x] T002 [US1] Implement the integration-only sequential runner in `integration/runner/main.go`, `run.go`, and `run_test.go` using T001's policy. Parse contracted controls; reject unsafe overrides and invalid requests before mutation; freeze the manifest, build once or honor an existing binary, isolate child directories by invocation/slice/profile, capture separate streams and test events, and persist atomic `release.json` / `summary.md`. Preflight all required targets before destructive work, recheck each slice, retain ordinary failures while collecting safe independent evidence, stop mutations after readiness loss, and terminate children/finalize on interruption. Test injected subprocesses for zero mutation on gate failures, 0/1/2/130 runner exits, unique directories, malformed output, hard-kill incomplete manifests, and inability of later successes to overwrite failure. Run `go test ./integration/internal/validation ./integration/runner -race -count=1`; retain a testable runner entry point without wiring Make until T003.

- [x] T003 [US1] Wire the runner into `Makefile` while preserving existing slices and confirmation controls; integrate additive normalized evidence in `integration/cli/harness_test.go`, `config_test.go`, and new `release_evidence_test.go` / `common_test.go`. Adapt existing family writers to stable scenario identities and proof levels without rewriting legacy report semantics. Add `integration-cli-common`, account for inventory/manifest/profile/smoke/examples, and split nonmutating preflight from potentially mutating examples. Preserve test suffix selection, per-profile records, verbosity and timeouts; ensure ordinary no-tag tests cannot execute the harness. Add runner/catalog parity and fake-result tests, format touched Go, run targeted pure tests and compile-only tagged validation (`go test -c -tags=integration -o /tmp/c8volt-338-integration.test ./integration/cli`). Update `integration/README.md` with the single-profile command and explicit `passed_with_gaps` limitation. Do not require a second server version for preflight or execution.

**Checkpoint**: The operator interface exists and cannot certify omitted required work. Until US2 is implemented, an 8.10 run must honestly fail for missing real-state coverage rather than weaken its catalog.

## Phase 3: US2 — Exercise real operational state on 8.10 (P1)

**Goal**: Enable the existing real-state workflows and correct version fixtures without duplicating the C89 suite.

**Independent test**: Deterministic version/fixture tests prove selection for 8.9 and 8.10; a later single-version live run records all seven families with their real proof levels.

- [x] T004 [US2] Generalize `integration/cli/real_state_harness_test.go` and all seven `real_state_*_test.go` family consumers together; replace cascade C89 constants in `real_state_cascade_test.go` with existing version-aware fixture resolution from `deploy_embed_run_test.go`. Preserve 8.8/8.9 baseline/volume behavior and 8.9 real-state support, actual full server versions, topology readiness, and dirty-cluster assertions. Align `real_state_gap_validation_test.go`, proposal versions in `harness_test.go`, `integration/assets/release-gaps.json`, and affected `specs/257-c89-real-state-integration/` gap/matrix references without deleting historical context. Cover mismatched/unparseable versions, unsupported 8.8 real-state applicability, missing fixtures, and C810 parent/mid/leaf identities. Keep BPMN-error real-job preview and unchanged-state proof distinct from its missing catchable-error mutation prerequisite. Add focused pure selection/fixture tests in `integration/internal/validation/fixtures_test.go` or the existing appropriate test owner, perform tagged compile validation and targeted non-live checks, and update `integration/README.md`. Do not attempt to close unrelated historical product gaps.

## Phase 4: US3 — Verify batch mutation outcomes (P2)

**Goal**: Validate truthful acceptance, completion accounting, and independent resource post-state, with draining modeled correctly.

**Independent test**: Controlled batch lifecycle cases are repeatable; later live evidence distinguishes submission from completion and reports the known deletion defect as a product failure.

- [x] T005 [US3] Add focused tagged scenarios in `integration/cli/batch_test.go` and process-definition draining coverage in `integration/cli/real_state_destructive_test.go`, with `integration-cli-batch` wiring and catalog entries. Cover successful batch-backed cancellation/deletion, failed-operation accounting, no-wait acceptance followed by separate observation, and held-instance definition draining with bounded cleanup checks. Reuse existing omitted-type tests in `internal/services/batchoperation/v810/service_test.go`; add only missing controlled terminal failure, temporary invisibility, timeout/cancellation, and lifecycle-decoding cases. Never introduce a batch DRAINING state. Treat a null deletion batch as valid upstream contract evidence, and preserve the current adapter failure as an unwaived product dependency in `specs/338-extend-810-validation/progress.md`. Do not add an unconditionally failing ordinary unit test or change production code to complete this task: keep the exposing adapter regression in explicitly invoked compatibility/integration coverage until the separate fix exists, then promote it to ordinary regression coverage with that fix. Validate passing focused batch tests, tagged compilation, and fake evidence failure classification; document the pending live acceptance in `integration/README.md` and do not mark product compatibility as passing.

## Phase 5: US4 — Verify tenant boundaries (P2)

**Goal**: Prove logical filtering and authorization separately, then physical routing where provisioned.

**Independent test**: Role configuration tests require no live tenant. Live logical checks use two tenants within one environment; physical checks additionally require two provisioned physical endpoints in that same declared cluster.

- [x] T006 [US4] Implement credential-free role loading/validation in `integration/internal/validation/tenant_roles.go` and adjacent tests, add `integration/assets/tenant-roles.example.json`, and implement logical scenarios in `integration/cli/tenant_test.go`. Wire `integration-cli-tenants`, role metadata, and required-tenant controls into `Makefile`, runner, and catalog in the same unit. Validate A/B, dual-authorized, and restricted-A profiles from default local config; positive owned visibility must precede negative assertions. Verify scoped search/mutation leaves B unchanged, restricted credentials cannot mutate proven foreign resources, and dual-authorized explicit/stdin keys are not falsely denied by configured tenant filters. Cover supported all-tenant discovery/mutation and existing single-destination/conflicting-flag restrictions. Reject duplicate tenant IDs, missing profiles, conflicting roles, and required missing infrastructure; optional absence remains a named gap. Apply target constraints quoted below, run pure role/reducer/runner tests and tagged compilation, and document required permissions in `integration/README.md`.

- [ ] T007 [US4] Extend `integration/cli/tenant_test.go` and `integration/internal/validation/tenant_roles.go` for explicit physical profile groups, with deterministic path-preservation tests in `config/api_test.go` and focused v810 client request coverage in `integration/internal/validation/physical_route_test.go`. Verify topology readiness, declared shared-cluster identity, distinct physical IDs, same logical tenant ID across endpoints, positive local visibility, absent foreign fixtures, unchanged control resources, and all-tenants confinement to the selected endpoint. Include explicitly configured default-route equivalence controls. Require physical mode to include logical proof; classify absent infrastructure as blocked when required and a gap otherwise. Do not use nondefault status-endpoint 404 as failure, infer topology from URL similarity alone, provision infrastructure, or claim storage/compute isolation. Run targeted URL/role/reducer tests and tagged compile checks; update `integration/assets/tenant-roles.example.json` and `integration/README.md` with the evidence limits.

## Phase 6: US5 — Use a coherent aggregate validation workflow (P2)

**Goal**: Supporting guards and alternative entry points agree with the actual release baseline and evidence scope.

**Independent test**: Static guard fixtures detect bad identity/hash/older-client changes; fake shell CLI tests prove wrong-version refusal. Aggregate category selection is already tested by T003 and must remain intact.

- [ ] T008 [US5] Align `api/tests/v810_provenance_test.py`, `v810_repository_boundary_test.sh`, and related cases in `v810_generation_test.sh` with pinned 8.10.3 and commit `3961b79c3fdaf43c48691ed77672d6db42d0abcc`. Keep independent expected identity, canonical command, hashes, transformations, package identity, and unaffected v88/v89 protection. Replace in-place alpha4 regeneration with disposable-checkout reproducibility against the pinned tag; eliminate obsolete bans on v810 integration content and any blanket cleanliness requirement that prohibits intentional harness edits. Add positive/negative guard fixtures and prove older-client changes are detected by before/after fingerprints, not merely an already-committed clean tree. Run static Python cases and shell fixture checks first; networked reproducibility remains a separately reported check in the isolated checkout. Update `api/README.md`; never refresh the working checkout as a validation side effect.

- [ ] T009 [US5] Add `integration/scripts/run-c810-suite.sh` and align `integration/scripts/run-suite.sh`, `integration/scripts/lib/suite-lib.sh`, `integration/README.md`, and relevant integration prompts/matrix descriptions with C810 fixtures and the version gate. Keep the shell runner's distinct scope explicit; its success is not aggregate certification. Add offline shell-runner tests with a fake CLI for accepted/rejected versions and no mutation on mismatch, and use `bash -n` on changed scripts. Reconcile `specs/338-extend-810-validation/quickstart.md` with implemented controls, five common categories, required-tenant semantics, proof levels, and current product dependency. Retain single-environment examples first and older-version commands as separate runs; no product CLI documentation generation is needed for tooling-only changes.

## Phase 7: Cross-cutting acceptance evidence

This is release proof, not a routine formatting or bookkeeping task. Do not start it without the required environment selection/authorization; do not fabricate evidence to finish the checklist.

- [ ] T010 Validate the implemented release workflow against [quickstart.md](quickstart.md) and record an evidence index and exact results in `specs/338-extend-810-validation/validation.md`, referencing retained external run directories rather than copying credentials or large logs into the repository. Use deterministic fault cases for missing/duplicate/stale evidence, wrong-version refusal, interruption, and catalog completeness; then perform the separately authorized single-profile 8.10.3 live run. Run older-version regression separately when available and required logical/physical scenarios only with explicit role prerequisites. Account for every FR/SC and every gap, actual server version, all seven real-state families, five common categories, batch outcomes, and tenant controls. Reuse prior valid checks; do not broaden testing solely to commit. Leave acceptance incomplete for unavailable required environments or the unresolved null-batch product defect, stating the exact blocker and follow-up; neither skips nor known product failures satisfy this task.

## Normative Data Constraints for T001–T003 and T006–T007

These quotations are acceptance requirements, not suggestions. T001 implements common validation; T002–T003 enforce lifecycle and evidence constraints; T006–T007 complete tenant-specific validation.

> Rules: release mode requires a supported canonical minor; profiles refer only to default local configuration. No credentials, tokens, or full config dumps are serialized. Required coverage is frozen before execution. One requested minor per strict invocation; mixed-profile exploratory runs retain per-version results. Foreign-minor profiles cannot satisfy a release request.

> Rules: version is parsed and verified before mutation; unknown/mismatched versions cannot be ready. Logical pairs must be distinct tenants within a declared physical endpoint. Physical pairs must identify distinct physical tenants within a declared cluster. URLs alone do not establish cluster identity. Positively observed fixtures substantiate each role before negative assertions.

> Rules: scenario identities are unique within the catalog; every applicable family has explicit minimum scenarios. No runtime wildcard waivers. Unsupported 8.8 real-state scenarios are not required. Controlled proof never satisfies a live post-state requirement.

> Rules: a gap may describe nonrequired extended proof but cannot remove an entire required family or convert product failure into success. Unrecognized gaps are not accepted automatically. Historical C89 gaps are reassessed for each minor rather than copied as validation evidence.

> States: `planned -> running -> finished | failed | interrupted`. Planned or running slices at evaluation time are incomplete. Separate child directories prevent overwrites; duplicate completion records are invalid. An exit code of zero does not establish scenario success without matching evidence.

> Outcomes: `passed`, `skipped`, `blocked`, `failed`, `unsupported`.

> Evidence levels: `live-post-state`, `submission-only`, `dry-run`, `controlled`, `read-only`.

> Failure classes: `product`, `harness`, `fixture`, `environment`, `authorization`, `version`, `incomplete-evidence`.

> Rules: passed evidence must meet the catalog minimum. Submission success must not imply completion. Read-only positive/negative tenant assertions include confirmed resource ownership; absence alone is insufficient. Cleanup is separate and cannot erase a scenario failure. Paths remain beneath the invocation root; secrets are redacted.

> Verdicts: `running`, `passed`, `passed_with_gaps`, `failed`, `incomplete`.

> Transitions: initial `running`; any unmet required check or failed child produces `failed`; interruption or unfinished mandatory child produces `incomplete`; all requirements met with nonrequired accepted gaps produces `passed_with_gaps`; all applicable proof complete produces `passed`.

> `coverageComplete` is true only for `passed`. `passed_with_gaps` means declared required checks passed, not certification of the blocked behaviors. Counters and lists derive from expected requirements joined to current-run evidence, never only from files that happen to exist. Stale or foreign-run evidence is rejected.

## Dependencies and Execution Order

```text
T001 -> T002 -> T003 -> T004 -> T005
                           -> T006 -> T007
T001 -> T008
T003 + T004 -> T009
T004 + T005 + T007 + T008 + T009 -> T010
```

- US1 requires the shared foundation; runner and harness wiring are sequential and share a contract.
- US2 follows US1 so version-specific execution immediately produces enforceable evidence.
- US3 follows US2; successful live deletion acceptance additionally depends on the separately accepted production fix. T005 can deliver truthful test coverage while that acceptance remains blocked.
- US4 follows US1; physical scenarios extend the logical roles and are sequential within US4.
- US5 reuses common category wiring from US1. T008 can be developed independently of live family changes after T001; T009 depends on version-aware fixture behavior.
- T010 requires completed harness work and separately authorized live prerequisites. No requirement to have 8.9 and 8.10 running simultaneously is introduced.

## Parallel Opportunities

Default implementation is sequential because runner, catalog, Make targets, and evidence writers overlap. No whole tasks are marked `[P]`. After T001, T008's isolated API guards can be developed independently of family scenarios if shared documentation edits are coordinated. Offline reducer, runner, fixture, and role checks may run independently; destructive families sharing a cluster must remain serial.

## Requirement Traceability

| Requirements | Tasks |
| --- | --- |
| FR-001–003: requested version, required coverage, pre-mutation gate | T001–T004 |
| FR-004–006: seven families, older support, version fixtures | T001, T004 |
| FR-007–009: records, proof levels, predeclared gaps | T001–T003 |
| FR-010: version-specific historical prerequisite gaps | T004 |
| FR-011–012: batch outcomes and draining compatibility | T005 |
| FR-013–014: logical and physical tenant proof | T006–T007 |
| FR-015: all five common validation categories | T003, T009 |
| FR-016: proposal versions, artifacts, shell runner | T004, T009 |
| FR-017: pinned provenance and boundary protection | T008 |
| FR-018–019: disposable targets, failure classes, interruption/cleanup | T001–T007 |
| FR-020: operator instructions | T003–T009 |
| SC-001–008: final acceptance evidence | T010, using the targeted checks in each unit |

## Consolidation Review

**Decision: retain 10 tasks — nine implementation units plus one external acceptance gate.** The existing units are already substantial; reducing the count would mostly hide distinct work inside larger checkboxes.

| Possible merge | Decision |
| --- | --- |
| T001 policy + T002 runner | Keep separate: pure coverage decisions and subprocess lifecycle have independent tests and different failure modes. |
| T002 runner + T003 harness wiring | Keep separate: orchestration is already substantial; wiring touches existing evidence producers and Make behavior. |
| T006 logical + T007 physical tenants | Keep separate: physical routing has additional provisioning, URL, and isolation controls; logical coverage is independently useful. |
| T008 API guards + T009 shell runner | Keep separate: independent generation integrity and release-entry behavior; merging would only reduce checkbox count. |
| T010 acceptance into implementation | Keep separate: live proof depends on selected infrastructure and the external deletion fix; offline completion must not imply certification. |

T004 already keeps version selection, fixtures, and all affected consumers together. T005 already combines batch/draining scenarios with their controlled checks. Tests and necessary documentation remain inside every implementation unit. No requirements or coverage were dropped in this review.

## Implementation Strategy

**MVP**: T001–T003 deliver trustworthy single-environment release selection and missing-coverage detection. T004 is the next increment needed to execute 8.10 real-state families. An MVP may truthfully fail incomplete release coverage; it must never claim full 8.10 compatibility prematurely.

**Incremental delivery**: foundation and runner -> existing family version support -> batch and tenant proof -> supporting guards/entry points -> authorized live acceptance. Each unit leaves the project compiling with relevant offline tests passing. Required live checks may remain blocked by explicit prerequisites, but their pending status must not be hidden or marked passed.

**Validation discipline**: write focused regression assertions before changing the corresponding behavior within each unit, then finish the implementation and relevant checks together. Use pure packages and injected subprocesses to test control policy. Compile tagged packages without invoking TestMain when live execution is not intended. Do not add routine failing tests to the ordinary unit suite to track an out-of-scope product bug; keep the opt-in exposing coverage and its dependency explicit. Do not perform a client refresh in the working checkout.

**Commits if requested or required by an implementation workflow**: Conventional Commits with a clear scope and subject ending ` #338`. Do not automatically commit or start Ralph merely because this task list was generated.
