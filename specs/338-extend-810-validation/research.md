# Research: Camunda 8.10 Release Validation

Date: 2026-10-14. Scope: issue #338; read-only source inspection and official documentation. No live tests were run.

## Execution decision (supersedes earlier runner research)

On 2026-10-15 the operator explicitly requested removal of the Go runner and
restoration of direct Make execution. Keep the established per-group harness,
serial targets, version checks, useful tests and fixture fixes. Do not retain
aggregate policy/catalog/reducer infrastructure. Historical run evidence remains
in validation.md; old release verdicts describe only those historical runs.

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
