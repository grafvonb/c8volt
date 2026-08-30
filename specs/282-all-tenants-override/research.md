# Research: All-Tenants Tenant Override

## Decision 1: Apply the override in root configuration resolution

**Decision**: Add `--all-tenants` as a root persistent boolean flag. Resolve base configuration, active profile, environment, and the existing `--tenant` flag through the current Viper path first. When `--all-tenants` is active, capture the resolved configured tenant for private provenance and then set the effective `cfg.App.Tenant` to the existing empty-filter value before the config enters command context or remote services are installed.

**Rationale**: `cmd/root.go` and `cmd/root_config.go` already own configuration precedence. Existing read/search/service paths interpret an empty `cfg.App.Tenant` as unfiltered discovery, so a root-only override reaches every applicable command without facade or service changes. Applying it after configuration normalization is essential for Camunda 8.7, where an omitted tenant otherwise normalizes to `<default>`; applying it before the config enters context ensures commands and services receive the overridden value.

**Alternatives considered**:

- Bind `--all-tenants` as a new durable config field: rejected because the issue defines an explicit command-line override, not a profile/environment/configuration concept.
- Force the raw `tenant` flag to empty before Viper resolution: rejected because it obscures provenance and makes profile, environment, explicit-empty, and Camunda 8.7 behavior brittle.
- Add per-command tenant options: rejected because it duplicates central resolution and can drift across command families.

## Decision 2: Validate tenant choices before configuration and command work

**Decision**: Add one root-level validation function at the start of `PersistentPreRunE`, after flags are parsed and after the existing help bypass. When `--all-tenants` is true and `--tenant` was explicitly changed, including `--tenant ""`, return the established mutually-exclusive invalid-input error. Treat explicit `--all-tenants=false` as inactive. When active `--all-tenants` is used on a command classified as requiring one concrete destination, return an invalid-input error before configuration loading, service installation, stdin/file access, prompts, activity, report creation, or remote requests.

**Rationale**: Root validation gives every command the same timing and error class. The repository's `mutuallyExclusiveFlagsf`, `invalidFlagValuef`, `silenceUsageForError`, and bootstrap error rendering already preserve invalid-input exit and structured-envelope behavior.

**Alternatives considered**:

- Cobra flag-group exclusivity alone: rejected because project-owned error constructors provide the stable invalid-input class and explicitly handle an empty changed `--tenant` value.
- Validate inside each command runner: rejected because several runners initialize clients or inspect files before local validation, and future commands could omit the rule.
- Infer invalidity from state-changing command metadata: rejected because cancel, delete, resolve, repair, retention, and purge operations legitimately use all-tenant discovery.

## Decision 3: Use one annotation for runtime safety and capability discovery

**Decision**: Add a command annotation and enum-like capability value with two states:

- `accepted`
- `rejected_concrete_destination`

Commands default to `accepted`. Mark these four executable leaves as `rejected_concrete_destination`:

1. `deploy process-definition`
2. `run process-instance`
3. `embed deploy`, including its optional run step
4. `ops execute smoke-test`, including dry-run planning

The root validator and `CommandCapability` serializer MUST use the same annotation resolver.

**Rationale**: These are the complete production command call sites for deployment, process-instance creation, or the composite smoke-test creation workflow. Their backend paths convert an empty tenant to `<default>`, so they must reject rather than silently reinterpret the flag. A shared annotation keeps runtime behavior and machine-readable discovery synchronized.

**Alternatives considered**:

- Hard-code command paths in root validation: rejected because a separate capability mapping could drift.
- Put support state on the inherited flag metadata: rejected because inherited flags are shared while support is command-specific.
- Add a third `not_applicable` state to every utility and grouping command: deferred as unnecessary scope; accepted commands may expose the inherited flag even when they have no tenant-sensitive work, matching current `--tenant` scope.

## Decision 4: Reuse existing empty-tenant warning and protected output paths

**Decision**: Extend private `tenantOverrideProvenance` to identify that the empty effective tenant originated from `--all-tenants`. For a named configured tenant, the existing human tenant-context pipeline renders, in order:

1. `configured tenant: <tenant>`
2. warning: `--all-tenants overrides the configured tenant filter; selection is unfiltered`
3. `selection scope: unfiltered across accessible tenants`

If the configured tenant was already empty, omit override provenance chatter and keep only the ordinary unfiltered scope. Preserve the existing `--tenant ""` wording and behavior. Do not add command-line provenance to public `tenant.Context` or structured schemas.

**Rationale**: Feature #283 already centralizes warning classification, once-only rendering, durable progress severity, and suppression for JSON, keys-only, total-only, and quiet output. Reusing it provides the required safety signal without creating parallel renderers or changing machine payloads.

**Alternatives considered**:

- Emit only an informational line: rejected because clearing a named filter is an established warning-level broadening event.
- Add a new warning field to structured tenant context: rejected because the effective `filter: "none"` and existing `unfiltered_selection` warning already describe machine semantics; command-line provenance is intentionally private.
- Print directly from root bootstrap: rejected because it would bypass protected output-mode routing and duplicate command-context rendering.

## Decision 5: Keep authorization and backend layers unchanged

**Decision**: Implement no facade, domain-service, version-adapter, generated-client, or backend request changes. Applicable discovery commands consume the same empty tenant filter already produced by `--tenant ""`. Direct resource-key operations retain their existing backend-authorized behavior. `--all-tenants` MUST NOT map to `WithIgnoreTenant` or any equivalent authorization-bypass option.

**Rationale**: The feature changes how an operator explicitly selects an already-supported filter state. Existing service behavior is correct for all supported Camunda versions. `WithIgnoreTenant` controls a different direct-key/admin path and would broaden authorization semantics beyond the issue.

**Alternatives considered**:

- Add a new facade/service option: rejected because it duplicates the effective config value and increases cross-layer scope.
- Reuse `WithIgnoreTenant`: rejected because it conflates unfiltered discovery with backend-authorized direct-key behavior.
- Add tenant enumeration before search: rejected because “all tenants” means no tenant filter within authenticated visibility, not a client-side fan-out.

## Decision 6: Add an additive capability field and keep contract version v1

**Decision**: Add `allTenantsSupport` to each `CommandCapability`, using the two values defined above. Keep the capability document version at `v1`. Continue serializing the inherited `all-tenants` flag contract on commands, while the command-level field communicates whether execution accepts it. Update human capability summary/help so JSON and human discovery do not contradict each other.

**Rationale**: An inherited flag alone cannot tell automation that creation commands reject it. The new field is additive, existing field meanings and result envelopes are unchanged, and repository precedent keeps additive optional metadata under the current capability version.

**Alternatives considered**:

- Rely only on the flag description: rejected because it is prose and cannot distinguish individual commands reliably.
- Bump the capability document version: rejected because no existing field is removed, renamed, or reinterpreted.
- Use a boolean: rejected because `rejected_concrete_destination` explains why the flag is unavailable and avoids conflating safety rejection with missing implementation.

## Decision 7: Validate request shape, output isolation, and documentation together

**Decision**: Cover root resolution and provenance with unit tests; use command/subprocess tests for invalid-input exit, structured errors, early rejection, warning order, output-mode isolation, and zero remote work; use representative mock-backed discovery tests to prove tenant filters are omitted. Regenerate CLI documentation from source with `make docs-content`, update README and non-generated ops safety guidance, and update integration example root-flag parsing for the new boolean flag.

**Rationale**: The behavior crosses flag parsing, configuration precedence, help/capabilities, human warnings, and creation safety. Close tests prove each owner boundary, while generated documentation prevents all inherited help pages from drifting.

**Alternatives considered**:

- Test only root config helpers: rejected because it would miss command classification, exit behavior, output routing, and remote-work timing.
- Require a live multi-tenant cluster for all validation: rejected because current integration profiles do not guarantee an identity with asymmetric tenant visibility. Automated request-shape and unchanged-authorization tests remain deterministic; a real visibility check can stay as an optional gated scenario.
- Hand-edit generated `docs/cli/*`: rejected because the repository requires source metadata plus regeneration.

## Resolved Technical Context

- No new dependency, storage, public facade API, backend API, or generated client is required.
- The change is version-neutral across Camunda 8.7, 8.8, 8.9, and 8.10.
- The four concrete-destination leaves are complete based on production call-site inspection for deployment, process creation, and smoke-test execution.
- Capability metadata and runtime validation share one source of truth.
- All Phase 0 questions are resolved; no technical context remains undecided.
