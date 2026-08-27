# Research: Camunda 8.10 Support

## Decision 1: One Ordinary V810 Identity

**Decision**: Add `toolx.V810` with canonical value `8.10` and aliases `8.10`, `810`, `v810`, and `v8.10`. The original delivery kept `CurrentCamundaVersion = V88`; issue #277 later supersedes that default with V89. Treat `8.10.0-alpha4` as source provenance, not a selectable identity.

**Rationale**: `toolx/version.go` already owns normalization, display, support sets, and the default. One identity lets later alpha, RC, and GA baselines replace the implementation without operator/package renames.

**Alternatives considered**: The prior `V810Alpha`/`v810alpha` proposal conflicts with issue #273. Generic semantic-version ordering was rejected because capabilities are not guaranteed to be monotonic.

## Decision 2: Isolated Target-Aware Generation

**Decision**: Add `bash api/refresh-clients.sh --target v810 --camunda-tag <tag>` as an isolated mode using temporary source/output directories and publishing only `internal/clients/camunda/v810/camunda`.

**Rationale**: The current refresh path fetches product plus legacy docs sources and regenerates a fixed client list. Feeding it an 8.10 source would overwrite protected v8.8/v8.9 unified clients and rotate the tracked `api/camunda` directory.

**Alternatives considered**: Extending the fixed all-client list was rejected because one fetched spec cannot represent multiple minor lines. Hand-generation was rejected as non-reproducible.

## Decision 3: Deterministic Machine-Readable Provenance

**Decision**: Store `provenance.json` beside the generated client. Record schema version, repository, tag, peeled commit, source path/hash, ordered transformation paths/hashes, exact Redocly/oapi-codegen versions, prepared-spec hash, generated-client hash, and canonical command. Exclude timestamps and temporary paths.

**Rationale**: Content hashes and tool versions expose drift while deterministic fields let identical generation produce no diff.

**Alternatives considered**: Commit messages and prose-only docs are not machine-verifiable. Tag-only provenance cannot prove source/tool/transformation identity.

## Decision 4: Reuse the Product-v2 Mutation Chain

**Decision**: Bundle `zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml`, then run the existing search-query, search-result, process-instance-filter, job-result-discriminator, and operation-ID mutations in their current order.

**Rationale**: This is the repository-native unified-client path and preserves fields required by current workflows. Tests must assert transformation effects because a script can exit successfully while no longer matching an upstream schema.

**Alternatives considered**: Unmodified generation breaks known shapes. Speculative 8.10-only mutations are deferred until a compile/behavior gap proves they are needed.

## Decision 5: Eleven Native V810 Adapter Families

**Decision**: Add native `v810` packages and explicit factory selection for batch operations, cluster, elements, incidents, jobs, process definitions, process instances, resources, tenants, user tasks, and variables. Follow v89 ownership and keep conversion local.

**Rationale**: These are the eleven version-aware services. `c8volt/client.go` constructs ten; process instances construct variables internally. Native adapters prove no older contract is silently reused.

**Alternatives considered**: Aliasing v89 services/types hides upstream differences and violates isolation. Moving conversion into facades violates layering.

## Decision 6: Unified Client Only

**Decision**: V810 adapters import only `internal/clients/camunda/v810/camunda` among generated clients. They do not construct Operate, Tasklist, or Administration SM clients. User tasks use the unified API and return explicit existing-domain outcomes where the baseline lacks an operation.

**Rationale**: The current v89 user-task package still has a Tasklist fallback; copying it would create a false compatibility claim for removed component contracts.

**Alternatives considered**: Legacy fallbacks and older component clients violate the specification and cannot prove 8.10 behavior.

## Decision 7: Named Capability Predicates

**Decision**: Add explicit predicates in `toolx`, initially full process-definition history deletion with set `{V89, V810}`. Use it in direct deletion and all-process-definitions purge.

**Rationale**: Both current checks test exact V89 while promising “8.9 or newer.” One named predicate avoids scattered boolean checks and non-monotonic version assumptions.

**Alternatives considered**: String/ordinal comparisons were rejected. Leaving exact checks would deny V810 incorrectly.

## Decision 8: Explicit Native Fixture Compatibility

**Decision**: Add a named production-fixture mapping from V810 to `C810_`. Keep report identity `8.10`; add no live 8.10 integration profile.

**Rationale**: Native C810 embedded/smoke resources make the 8.10 ownership visible while preserving the already-compatible workflow behavior. Naming/testing the mapping is more auditable than silently overloading `FilePrefix`.

**Alternatives considered**: Continuing to reuse C89 resources obscures 8.10 fixture ownership. Extending live integration selection remains out of scope for this support line.

## Decision 9: Release-Line Gateway Compatibility

**Decision**: Compare parsed major/minor lines. Configured `8.10` matches observed `8.10`, `8.10.x`, and `8.10.0-alpha4`. Different, empty, or unparseable lines are not matches and remain visible through configuration diagnostics.

**Rationale**: Existing diagnostics already parse major/minor. Exact equality rejects valid patches/prereleases; unknown versions cannot prove compatibility.

**Alternatives considered**: Exact equality is too strict. Treating unparseable versions as compatible is unsafe.

## Decision 10: Additive Version Metadata

**Decision**: Keep the human layout and shared JSON envelope. Extend the supported string with 8.10 and add baseline tag/status fields; human output adds one compact disclosure line.

**Rationale**: Existing automation consumes string fields. Additive fields disclose the source without renaming the configured version or forcing prose parsing.

**Alternatives considered**: Encoding alpha in the identity conflicts with issue #273. Replacing the payload with nested objects is an unnecessary contract break.

## Decision 11: Remove Generated-Enum Leakage

**Decision**: Replace the v89 generated enum imported by `internal/services/incidentfilter` with version-neutral canonical values.

**Rationale**: Command filter validation is version-neutral. Importing v810 would make stable behavior depend on prerelease artifacts; retaining v89 weakens strict source-boundary audits.

**Alternatives considered**: Allowing the leak obscures ownership. Per-command generated dependencies violate command layering.

## Decision 12: Local Proof, Stable Regression, No Integration Expansion

**Decision**: Validate with generation guards, factory/interface tests, adapter and command fake servers, source scans, docs tests, and `make test`. Require zero new diff under `integration/` and protected v8.7-v8.9 clients.

**Rationale**: The issue defers live 8.10 integration. Controlled local servers provide operational proof without expanding destructive real-cluster coverage.

**Alternatives considered**: A C810 integration profile is out of scope. Compile-only coverage cannot prove request, conversion, mutation, or confirmation behavior.
