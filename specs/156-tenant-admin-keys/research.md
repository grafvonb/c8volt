# Research: Tenant Scope For Discovery And Explicit Admin Keys

## Decision: Preserve tenant filtering for c8volt-produced discovery/search candidate sets

**Rationale**: The issue distinguishes candidate sets discovered by c8volt from explicit operator input. Search/list flows, search-derived cancellation, and search-derived deletion must keep applying the selected tenant where supported so the scope an operator previews or confirms does not broaden after c8volt has produced the target set.

**Alternatives considered**:
- Remove tenant filtering everywhere: rejected because discovery/search/create/deploy/run flows still need tenant scoping.
- Treat direct keys and search-derived candidates identically: rejected because the source of target selection has different operator intent and risk.

## Decision: Treat direct keys, stdin keys, IDs, and direct flag values as backend-authorized admin input

**Rationale**: c8volt is an admin tool. When an operator supplies an explicit key or ID and Camunda returns or accepts the target, c8volt should not add a stricter local tenant authorization rule based on selected `--tenant` metadata. Existing safety controls can still apply, but they must not convert returned tenant metadata into a local block for explicit input.

**Alternatives considered**:
- Enforce selected-tenant equality for every returned resource: rejected because it makes c8volt stricter than Camunda and conflicts with the issue's explicit admin-input contract.
- Add a new opt-out flag for cross-tenant explicit keys: rejected because the desired default for explicit admin input is backend authorization.

## Decision: Keep existing safety behavior for destructive direct-key commands

**Rationale**: The issue permits existing preflight, dry-run, confirmation, dependency expansion, force, wait, and verification behavior. These are mutation-safety concerns, not tenant authorization. The implementation should remove only tenant-mismatch blocking for explicit inputs while preserving scope visibility and operational proof.

**Alternatives considered**:
- Skip preflight and dependency expansion for direct keys: rejected because it would weaken destructive command safety.
- Apply search-derived tenant scope during direct-key dependency expansion without checking behavior: rejected because expansion must be audited to avoid both cross-tenant broadening from discovered sets and stricter-than-backend blocking for explicit inputs.

## Decision: Scope behavior changes to Camunda 8.8 and 8.9

**Rationale**: The issue explicitly excludes Camunda 8.7. Existing documentation already notes upstream limitations where tenant-safe direct keyed process-instance behavior is not available for 8.7. The implementation should keep v8.7 behavior intact and avoid expanding the compatibility matrix inside this issue.

**Alternatives considered**:
- Rewrite v8.7 tenant behavior for consistency: rejected as out of scope.
- Hide v8.7 differences behind command-layer special cases: rejected because version-specific behavior belongs below the command surface where existing factories and services own it.

## Decision: Update help/docs through command metadata and generated docs

**Rationale**: Operators need to understand the contract before invoking broad or destructive workflows. The repository constitution requires user-visible command changes to keep README and generated CLI docs aligned with executable help.

**Alternatives considered**:
- Update README only: rejected because generated command docs would drift from help.
- Add verbose runtime warnings for every explicit key: rejected as noisy; help/docs should carry the general contract, while returned tenant metadata can remain visible in normal output.

## Decision: Validate with one discovery-derived flow and one explicit direct-key flow first

**Rationale**: The issue's acceptance criteria require both sides of the contract. A targeted discovery/search-derived test proves c8volt-produced scope remains tenant-bounded, while a direct-key mismatch test proves backend-authorized admin input is not blocked by local tenant equality.

**Alternatives considered**:
- Only add command help tests: rejected because they do not prove behavior.
- Only add broad end-to-end tests: rejected because service/facade tests can isolate version-specific tenant option propagation more clearly.
