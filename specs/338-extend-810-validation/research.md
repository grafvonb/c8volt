# Research: Camunda 8.10 Release Validation

Date: 2026-10-14. Scope: issue #338; read-only source inspection and official documentation. No live tests were run.

## 1. Aggregate execution and evidence

**Decision:** Add a small Go integration runner and a shared, standard-library-only validation package under `integration/`. The aggregate Make target delegates to the runner after its existing confirmation gate. Keep individual Make slices available. The runner executes existing tagged tests sequentially, records each subprocess result, and finalizes a version-aware manifest even after failure.

**Rationale:** The existing Make target launches separate test processes, each with its own marker and work directory. Reusing a work directory directly overwrites reports and logs. Make prerequisite success and Go skip success cannot establish required coverage. A shared reducer needs deterministic unit tests without invoking the destructive tagged TestMain.

**Alternatives considered:** A single broad test regex is smaller but does not solve scenario-level completeness or isolated reports. Shell JSON parsing adds dependencies and duplicated policy. A new production command would expose internal release tooling unnecessarily.

Use one invocation directory, unique slice/profile child directories, and a parent manifest. Build the CLI once and pass its absolute path with the existing `C8VOLT_IT_BIN` / `C8VOLT_IT_BUILD=0` interface. Compile caching remains available. Separate child invocations per profile avoid existing scenario log-name collisions. No parallel destructive families on the same cluster.

## 2. Required coverage and accepted gaps

**Decision:** Versioned required-scenario catalogs and accepted-gap records are checked in before execution. Strict release intent is enabled by an explicit requested minor. Default exploratory behavior remains available. The reducer requires successful subprocess termination plus matching scenario evidence; process exit alone is insufficient.

**Rationale:** Real-state BPMN error coverage currently proves a real job, a preview, and unchanged-job follow-up, not catchable error completion. A blanket requirement that every family demonstrate every mutation would silently broaden #338 into closing all older fixture gaps. Conversely, a blanket waiver would preserve false-green results.

**Resolution:** Every required family must execute and satisfy its declared minimum evidence. Known additional blocked proofs may remain as named nonrequired gaps, producing `passed_with_gaps`, never `complete`. Required post-state assertions cannot be downgraded or waived at runtime. Unknown missing evidence fails. Inventory the actual scenario IDs and minimums during implementation and test catalog/evidence parity. A required family wholly skipped always fails.

**Alternatives considered:** Treat all skips as successes (unsafe), require closure of all historical gaps (outside the issue), or offer runtime blanket waiver flags (undermines the gate).

## 3. Version gate and fixtures

**Decision:** Reuse local profile resolution and topology-based readiness. Compare parsed major/minor values, replacing substring matching for the strict gate. Verify all selected targets in a nonmutating preflight and recheck readiness at each destructive slice. Support 8.9/8.10 real-state selectors; keep 8.8 applicability explicit. Resolve cascade IDs through existing fixture prefix and BPMN process-ID discovery.

**Rationale:** `real_state_harness_test.go` excludes every 8.10 profile, while `deploy_embed_run_test.go` and existing `C810_*` models already support version-aware discovery. `config test-connection` uses topology, suitable for physical tenant routes. Configuration remains the operator's default local configuration; no generated credentials or alternate config files.

**Alternatives considered:** Copy all C89 tests into a C810 suite (drift), infer version from fixture names (not a remote safety check), or certify a patch solely from the configured minor (inaccurate).

## 4. Batch operations and process-definition deletion

**Decision:** Treat `DRAINING` as process-definition lifecycle state, not a batch state. Cover actual batch terminal accounting and optional operation type separately. Corrected spec Story 3 and FR-012 accordingly.

**Evidence:** Pinned generated client and provenance identify Camunda 8.10.3, commit `3961b79c3fdaf43c48691ed77672d6db42d0abcc`. Its batch enum excludes DRAINING. `DeleteResourceRequest` / response comments explain that runtime-present definitions use draining and return no batch; decision resources or definitions already removed from runtime can return a history batch. See `internal/clients/camunda/v810/camunda/client.gen.go` and `provenance.json`. The [official migration guide source](https://github.com/camunda/camunda-docs/blob/main/docs/apis-tools/migration-manuals/migrate-to-810.md) supplies supplementary context; pinned generated contracts take precedence over moving documentation.

**Known product dependency:** `internal/services/resource/v810/service.go` requests history deletion and rejects a null batch before its no-wait branch. A legitimate 8.10 deletion can therefore fail. This is an inspected contract mismatch, not a live-reproduced result. Keep the new scenario failing with `product` classification until a separately accepted production fix lands. Do not mark it accepted, change production behavior here, or invent a batch DRAINING transition. Planning can complete; successful release certification depends on resolving this defect.

**Coverage:** Reuse existing omitted-type unit coverage; add missing controlled cases only. Check lifecycle response decoding, temporary invisibility, terminal failure counts, cancellation/timeout, and null-batch resource responses. Live cancellation/deletion must verify post-state independently. A held owned process instance can stabilize draining; release/cancel it and observe eventual cleanup with bounded polling. No live test depends on capturing a fleeting transition.

**Alternatives considered:** Assume all resource deletion is batch-backed (contradicts pinned contract), test batch DRAINING (nonexistent enum), or expand #338 into a production migration (outside scope).

## 5. Logical and physical tenants

**Decision:** Supply test-role metadata naming existing profiles, logical IDs, and a provisioned cluster identity. Logical tenant filters operate inside one physical endpoint. Physical endpoints are selected by profile base URLs ending in `/physical-tenants/<id>/v2`; the default `/v2` route remains a compatibility control. Verify prefix preservation deterministically before live tests.

**Primary sources:** [API routing](https://docs.camunda.io/docs/self-managed/concepts/physical-tenants/api-routing/), [isolation model](https://docs.camunda.io/docs/self-managed/concepts/physical-tenants/), [configuration reference](https://docs.camunda.io/docs/self-managed/concepts/physical-tenants/configuration-reference/), [authentication and authorization](https://docs.camunda.io/docs/self-managed/concepts/physical-tenants/authentication-authorization/). Research identified these pages as 8.10 documentation. Physical tenants require provisioned, distinct storage configuration and tenant-specific security; the harness does not provision them. Nondefault tenant status endpoints can return 404 by design; use topology readiness.

**Existing CLI contract:** Configured logical tenant does not constrain explicit or stdin keys; backend authorization does. Test search-derived scoped mutations for filter isolation. Separately test an A-only principal against a proven B resource, preserving B state. A dual-authorized explicit-key request for B with configured tenant A is valid and must not be asserted to fail. `--all-tenants` cannot cross physical endpoints and remains invalid for operations needing a single creation destination.

**Evidence limits:** Require positive visibility before absence assertions, distinct verified logical IDs, and the same logical ID in two known physical tenants for physical separation controls. CLI evidence proves observable routing, authorization, and resource separation, not disk isolation, backup behavior, provisioning, or compute isolation.

**Alternatives considered:** Treat --tenant as physical routing or an explicit-key security boundary (both wrong); infer cluster relationship solely from URL similarity (unreliable).

## 6. Supporting guards and validation boundaries

**Decision:** Update alpha4 expected tag/commit to the pinned release while retaining independent identity, hash, transformation, and target-isolation assertions. Run deterministic regeneration only in an isolated disposable checkout using the pinned tag; never refresh the developer checkout as a test side effect. Remove obsolete bans on integration edits and v810 references, retaining v88/v89 generation protection through before/after comparisons in isolated generator tests.

**Rationale:** Current provenance test invokes alpha4 refresh twice in-place; the repository boundary script prohibits the feature itself. Relaxing these without replacement integrity assertions would hide regressions.

**Alternatives considered:** Delete the guards entirely (loses integrity checks), or derive every expectation from the same provenance object (tautological).

All design unknowns are resolved. Environment availability and the resource deletion product dependency are explicit execution prerequisites, not assumed successful validation.
