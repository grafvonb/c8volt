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

## Decision: Confirm default waits with incident-state observation and process-instance incident lookup

**Rationale**: The constitution requires operational proof unless an explicit opt-out exists. `resolve incident` should confirm supplied incident keys are no longer active or are reported resolved. `resolve pi` should confirm each selected process instance no longer has the initially discovered active incidents. `--no-wait` remains the opt-out.

**Alternatives considered**:

- Treat accepted resolution response as success by default: rejected because it weakens the repository's done-is-done behavior.
- Poll for every future incident on the process instance: rejected because the issue limits scope to incidents discovered at command start.

## Decision: Put result contracts in facade models and render through shared command view helpers

**Rationale**: Existing process-instance update, cancel, delete, and wait flows expose per-target result models with `OK`/`Totals` helpers and shared command rendering. Mirroring that shape gives consistent human output, JSON output, automation behavior, and test seams.

**Alternatives considered**:

- Render ad hoc maps in commands: rejected because it would duplicate output logic and make JSON contract tests brittle.
- Create a new top-level domain package for resolve: rejected because the feature is process/incident operational behavior and can reuse existing facade patterns.
