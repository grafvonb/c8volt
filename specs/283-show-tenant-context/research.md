# Research: Effective Tenant Context Before Mutations

## Decision 1: Model tenant context once, map it through existing layers

**Decision**: Add a version-neutral `internal/domain.TenantContext` and warning enums, with a public mirror in `c8volt/tenant`. Use the same serialized field names in command results and operations reports.

**Rationale**: Process, resource, job, incident, and ops packages all need the same evidence. A tenant-owned public type avoids cycles and keeps facades mechanical, while domain types remain independent of CLI rendering.

**Alternatives considered**: Per-command strings cannot support stable structured/audit contracts. A cmd-only model loses traversal detail before it can be summarized. Reusing `config.App.ViewTenant()` is incorrect because empty tenant means unfiltered discovery but `ViewTenant()` renders `<default>`.

## Decision 2: Make operation semantics explicit

**Decision**: Represent four operational modes—configuration, discovery, creation, and explicit keys—and a separate filter state (`named`, `none`, `not_applied`, or `not_applicable`). Use raw `cfg.App.Tenant` for configuration/discovery, `cfg.App.TargetTenant()` for creation, and explicit-key mode whenever the command intentionally uses `IgnoreTenant`.

**Rationale**: One empty configuration value has two valid meanings: no discovery filter and default creation target. The explicit-key path is a third behavior governed by backend authorization. Encoding mode prevents a generic display helper from collapsing these meanings.

**Alternatives considered**: Inferring behavior from an empty string cannot distinguish discovery from creation. Treating explicit keys as a configured tenant search contradicts existing authorization behavior.

## Decision 3: Aggregate only data already present in resolved plans

**Decision**: Add a deterministic accumulator under `internal/services/common`. Services feed it unique target keys and tenant IDs from search pages, traversal chains, plan items, current jobs, frozen incidents, variables, deployments, and run results already obtained by the workflow. Known tenant IDs are deduplicated and sorted; each unique affected key without metadata contributes once to `unknownTargetCount`.

**Rationale**: Internal services already own paging, traversal, and frozen plans. Aggregating there preserves layering and satisfies the prohibition on fetching metadata solely for reporting.

**Alternatives considered**: Looking up every key after planning adds traffic and race opportunities. Aggregating only in cmd fails where service details are no longer exposed. Empty tenant metadata is not proof of a default-tenant resource.

## Decision 4: Use additive structured context without changing payload meaning

**Decision**: Add an optional `tenantContext` object alongside `payload` in full-contract `ResultEnvelope` values. Limited/raw structured views and audit report roots expose the same optional object. Command context carries the current immutable value so generic result renderers can include it without wrapping or reshaping payloads.

**Rationale**: Existing payloads include both structs and lists. An optional envelope member preserves their shapes and the outcome/class/command semantics while ensuring one common object.

**Alternatives considered**: Wrapping every payload in `{tenantContext,data}` breaks payload shapes. Flattened tenant fields violate the clarification requiring one nested object. Human warning strings do not belong in machine payload data.

## Decision 5: Keep YAML scope repository-native

**Decision**: Do not add a global YAML output mode. Preserve `config show` as the existing YAML-producing command and add `tenantContext` to its sanitized YAML document when showing effective configuration. Operations reports continue to support Markdown and JSON only.

**Rationale**: The renderer supports JSON, one-line, keys-only, and tree; ops report tests explicitly reject YAML. FR-016 applies to existing structured surfaces, not a new output feature.

**Alternatives considered**: Adding `--yaml` across mutation commands is unrelated scope expansion. Printing diagnostics around YAML corrupts the document.

## Decision 6: Preserve output routing and confirmation behavior

**Decision**: Human tenant lines use existing human line/warning render helpers and appear after planning but before preview confirmation or mutation. Quiet and keys-only modes never receive these lines. Cross-tenant and unknown warnings are repeated in compact destructive confirmation text when a prompt exists; they remain informational and non-blocking.

**Rationale**: Existing render helpers route human diagnostics away from machine stdout and honor quiet. Showing evidence before the irreversible step is the feature's safety purpose.

**Alternatives considered**: Unconditional stdout breaks JSON, YAML, and keys-only output. New prompts on non-interactive deploy/run flows would change interaction rather than reporting.

## Decision 7: Keep audit schemas additive and truthful

**Decision**: Add `tenantContext` to every affected ops report, including orphan purge. Retain optional legacy `tenantId` for v1 consumers only when it truthfully represents a named filter or creation target. Omit it for unfiltered discovery; never fill it through `ViewTenant()`. Markdown reports render the operation-specific context and warnings.

**Rationale**: Current enrichment can record `<default>` for unfiltered work. The nested object fixes that ambiguity while an additive field preserves v1 decoding compatibility. Removing the legacy field entirely would require a broad report migration.

**Alternatives considered**: Keeping contradictory `tenantId: <default>` makes audit evidence untrustworthy. Immediately bumping all reports to v2 is unnecessary for optional additive data and corrected omission.

## Decision 8: Stay version-neutral across Camunda adapters

**Decision**: Implement aggregation above generated/versioned clients. Exercise shared logic and representative command paths across the 8.7–8.10 support matrix; make no adapter changes unless implementation discovers a version-specific missing tenant field.

**Rationale**: Existing domain models already carry tenant IDs. Camunda 8.7 configuration normalization maps absent tenant to its default-only semantics, while later versions retain empty discovery as unfiltered.

**Alternatives considered**: Generated-client edits are unnecessary. Duplicating aggregation in every versioned service creates avoidable divergence.
