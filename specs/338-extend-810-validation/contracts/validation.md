# Contract: Integration Release Validation

This is a proposed tooling interface for implementation, not an existing production CLI API.

## Operator controls

- Preserve `make integration-test-all` and all existing family targets.
- `C8VOLT_IT_RELEASE_VERSION=8.10` enables strict release mode for that minor. Initially accept 8.9 and 8.10 for the full real-state release matrix; 8.8 remains supported through existing baseline/volume selection. Empty means exploratory mode. Reject unknown values before mutation.
- Preserve `C8VOLT_IT_PROFILES`, `C8VOLT_IT_WORKDIR`, existing Go test verbosity, timeouts, and confirmation/automation controls. A configured work directory is a parent; always create a unique invocation child.
- Add `C8VOLT_IT_TENANT_ROLES` as a path to credential-free JSON role metadata naming existing profiles. It is not a c8volt config replacement.
- Add `C8VOLT_IT_REQUIRE_TENANTS=none|logical|physical`, default `none`. Physical includes logical proof. Missing optional infrastructure is listed as a coverage gap; explicitly required tenant coverage fails when unavailable.
- Preserve `C8VOLT_IT_BIN` with `C8VOLT_IT_BUILD=0` for an existing binary. Otherwise build once per aggregate invocation.
- Strict execution must reject test-selection flags that can remove required tests (such as arbitrary `-run`/`-skip` overrides). Preserve supported verbosity and existing timeout controls; validate overrides before any mutation.
- Add focused `integration-cli-common`, `integration-cli-batch`, and `integration-cli-tenants` targets. Common execution has a nonmutating preflight phase and a separately gated examples phase because examples may mutate.

## Tenant role metadata

Required logical group fields: `clusterId`, `physicalTenantId`, logical IDs A/B, profile names `tenantA`, `tenantB`, `bothTenants`, and `restrictedA`. Each profile already exists in local c8volt configuration. Test metadata must not contain credentials or auth overrides.

Optional physical group fields: declared `clusterId`, physical IDs A/B, corresponding profile names, common logical tenant ID, and default route/control profiles where testing default routing. Physical tenant provisioning and principal permissions are operator prerequisites.

Profile validation rejects duplicate logical IDs, same physical ID used as both physical controls, missing profiles, conflicting declarations, or version mismatch. Readiness uses topology. Explicit-key authorization assertions use the restricted principal; configured logical filters are not treated as authorization.

## Execution and output

1. Confirmation remains at the existing Make entry point; automation retains its existing explicit opt-out behavior.
2. Persist the planned manifest before subprocess execution. Preflight all required targets before any mutation. Recheck each destructive target before its slice.
3. Run slices sequentially. Child evidence path is unique by invocation, slice, and profile/group. Capture stdout/stderr separately and preserve original child status.
4. Write `release.json` and `summary.md` in the invocation directory, alongside manifest and child evidence. Existing family reports remain in their child directories. Version, scenario identities, proof levels, and gap rows follow [data-model.md](../data-model.md).
5. Ordinary failures do not suppress independent evidence collection. Readiness failures stop further mutations to affected targets. Interruptions stop new work and finalize available evidence; hard-killed unfinished manifests cannot later be interpreted as passed.
6. No credentials, sensitive headers, or raw configuration are copied into summaries. Existing log redaction requirements continue to apply.

## Result policy

| Condition | Verdict | Runner exit |
| --- | --- | --- |
| Every required check passes and all applicable proof is present | passed | 0 |
| Required checks pass; named nonrequired gaps remain | passed_with_gaps, coverageComplete=false | 0 |
| Required profile/family/scenario absent, skipped, blocked, or failed | failed | 1 |
| Child exits nonzero, evidence malformed/duplicate/stale, or required common check fails | failed | 1 |
| Invalid requested version or role configuration | failed | 2 |
| Interrupted execution | incomplete | 130 for interrupt; nonzero otherwise |

Make may normalize nonzero statuses; scripts must rely on success versus failure and use `release.json` for the detailed verdict. Exploratory runs retain permissive selection but must report gaps honestly. A missing physical environment never yields physical certification. A known production failure is not an accepted prerequisite gap.

## Minimum coverage and compatibility

- Baseline and volume apply to 8.8, 8.9, and 8.10. All seven real-state families apply to 8.9 and 8.10.
- Existing BPMN-error preview and real-job unchanged-state checks are required execution; catchable error mutation remains a separately named gap until setup exists. Report those proof levels accurately.
- Batch operations use their actual state enum; DRAINING tests apply to process-definition deletion. Normal wait asserts independent completion; no-wait asserts acceptance and records later observation separately.
- Logical scoped mutation assertions use discovery-based scope. Explicit-key operations honor backend permissions. All-tenants operates inside the selected physical endpoint.
- Inventory, manifest, profile, smoke, and example categories are explicitly selected and accounted for. Regex selection must include suffixed family tests such as user-task-variable scenarios without unintentionally selecting unrelated families.
- New evidence fields are additive to existing reports. No production CLI output schema changes are part of this feature.
