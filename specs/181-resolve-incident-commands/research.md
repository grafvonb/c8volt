# Research: Resolve Incident Commands

## Decision: `resolve` is a distinct root command family

**Rationale**: The feature resolves operational incidents and is not a field update. The repository already treats state-changing operator actions as verbs such as `cancel`, `delete`, `update`, and `run`; adding `resolve` keeps the command surface discoverable and avoids overloading `update`.

**Alternatives considered**:

- `update incident`: rejected because incident resolution is not a field update and the issue explicitly prefers a distinct root verb.
- `get pi --resolve-incidents`: rejected because it would mix inspection and mutation.

## Decision: Keep incident lookup and resolution in `internal/services/incident`

**Rationale**: Current process facade incident enrichment already routes lookup through `internal/services/incident`, while the issue explicitly forbids adding incident lookup or resolution behavior to `internal/services/processinstance`. The incident service API/factory is the correct service boundary for direct incident operations and process-instance incident discovery.

**Alternatives considered**:

- Add resolution to process-instance services: rejected by the issue and by the existing boundary.
- Call generated clients directly from commands: rejected because commands should use facade/service abstractions and tests can then cover version behavior cleanly.

## Decision: Support v8.8 and v8.9 resolution through generated incident endpoints, reject v8.7 before mutation

**Rationale**: Generated v8.8 and v8.9 clients expose `ResolveIncidentWithResponse` and process-instance incident search endpoints. The v8.7 incident service already rejects tenant-safe incident lookup as unsupported, so resolution should use the same unsupported-version discipline before mutation.

**Alternatives considered**:

- Try a best-effort v8.7 mutation path: rejected because unsupported versions must fail before mutation and the current incident service does not provide tenant-safe lookup.
- Use process-instance batch incident-resolution endpoints for `resolve pi`: rejected for the first iteration because the requirement says discover the active incidents at command start and resolve/report attempted incident keys.

## Decision: Confirm default waits by polling the same lookup paths used for planning and display

**Rationale**: The existing `update pi --vars` mutation path submits the mutation in the versioned service and, unless `--no-wait` is set, polls the same lookup path used for planning/display until the requested state is visible. Resolve commands should follow that pattern exactly: `resolve incident` polls incident lookup for each supplied incident key until it is no longer active, and `resolve pi` polls process-instance incident lookup until the initially discovered incident keys are no longer active for that process instance. Accepted mutation response alone is not confirmation. `--no-wait` remains the opt-out and returns submitted/accepted output without lookup polling.

**Alternatives considered**:

- Treat accepted resolution response as success by default: rejected because it weakens the repository's done-is-done behavior and diverges from `update pi --vars`.
- Poll for every future incident on the process instance: rejected because the issue limits scope to incidents discovered at command start.

## Decision: Add Lookup-Backed `--dry-run` Resolution Plans

**Rationale**: Issue #180 established the repository's newer mutation UX: state-changing commands should be able to load current state, build a compact pre-mutation plan, render dry-run output, and submit no mutation. Applying the same pattern to incident resolution lets operators review explicit incident keys and discovered process-instance incident sets before recovery actions run.

**Alternatives considered**:

- Make `--dry-run` echo only supplied keys without lookup: rejected because `resolve pi` must discover incidents to be useful and direct incident resolution should surface current state when the generated client supports lookup.
- Add interactive confirmation prompts with dry-run: rejected because issue #181 explicitly keeps interactive confirmation prompts out of scope.
- Let `--no-wait` change dry-run behavior: rejected because dry-run never submits a mutation, so there is nothing to wait for.

## Decision: Keep JSON Dry-Run Output Stable

**Rationale**: Issue #180 rejects JSON plus verbose output for state-changing dry-run plans so automation receives one stable payload shape. Resolve commands should follow the same rule and include dry-run status plus mutation-submission status in JSON plan/result payloads.

**Alternatives considered**:

- Let `--verbose` add fields to JSON dry-run output: rejected because it makes automation contracts mode-sensitive.
- Omit mutation-submission status from dry-run JSON: rejected because dry-run safety is easier to assert when the payload explicitly says no mutation was submitted.

## Decision: Put result contracts in facade models and render through shared command view helpers

**Rationale**: Existing process-instance update, cancel, delete, and wait flows expose per-target result models with `OK`/`Totals` helpers and shared command rendering. Mirroring that shape gives consistent human output, JSON output, automation behavior, and test seams.

**Alternatives considered**:

- Render ad hoc maps in commands: rejected because it would duplicate output logic and make JSON contract tests brittle.
- Create a new top-level domain package for resolve: rejected because the feature is process/incident operational behavior and can reuse existing facade patterns.
