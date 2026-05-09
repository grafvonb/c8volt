# Research: Get Incident Command

## Decision: Add `get incident` Under The Existing Get Command Family

**Rationale**: The feature is read-only incident inspection and belongs beside `get process-instance`, `get job`, and other lookup/list commands. The aliases `incidents` and `inc` make the command discoverable without adding a separate root.

**Alternatives considered**:

- Add a root `incident` command: rejected because it would split read-only lookup from the established `get` family.
- Add more `get pi` flags: rejected because the issue requires a first-class incident command and process-instance incident flags have distinct semantics.

## Decision: Keep Process-Instance Incident Concepts Out Of `get incident`

**Rationale**: `--with-incidents`, `--incidents-only`, and `--direct-incidents-only` describe process-instance rendering or process-instance result filtering. A direct incident command should expose incident fields as plain filters, such as `--state`, `--error-type`, and `--error-message`.

**Alternatives considered**:

- Reuse `--incident-*` names on `get incident`: rejected because the command context is already incident-specific and the issue asks for plain filter names.
- Add `--with-incidents` for consistency with `get pi`: rejected because it would make an incident command read like a process-instance view extender.

## Decision: Extend `internal/services/incident` For Search/List

**Rationale**: Existing keyed incident lookup, process-instance related incident lookup, and resolution already route through `internal/services/incident`. Adding an explicit incident search/list service method keeps version-specific request shape decisions inside the service boundary and gives command/facade tests a stable seam.

**Alternatives considered**:

- Call generated clients directly from `cmd`: rejected because it would duplicate version selection and make compatibility tests brittle.
- Put search under `internal/services/processinstance`: rejected because the data being searched is incident data and the issue says to reuse incident boundaries.

## Decision: Reuse `ProcessInstanceIncidentDetail` As The Incident Detail Model

**Rationale**: The existing domain and public facade incident detail model already carries incident key, creation time, process context, flow-node context, state, error type, error message, tenant, and optional job key. Reusing it avoids a parallel hierarchy and lets existing render helpers evolve toward shared incident line formatting.

**Alternatives considered**:

- Create a new standalone incident model: rejected because it would duplicate fields and conversion logic.
- Return raw generated client payloads from the facade: rejected because JSON and human rendering should remain stable across generated client versions.

## Decision: Use Server-Side Filters Only When Semantics Are Safe

**Rationale**: v8.9 should use server-side filters for tenant, state, error type, process context, flow node, and creation time because those fields have exact or bounded semantics. Error-message filtering requires case-insensitive substring behavior, which should be applied locally unless the backend guarantees compatible behavior.

**Alternatives considered**:

- Always use backend `$like` for messages: rejected because backend case sensitivity is not guaranteed by the issue.
- Always filter everything locally: rejected because safe server-side narrowing reduces result volume and follows the issue's preferred v8.9 path.

## Decision: Make v8.8 Compatibility Explicit

**Rationale**: The issue records v8.8.9 rejecting a scoped related incident search request containing `filter`. The new search implementation must avoid blindly reusing that request shape. When v8.8 cannot express a required filter safely, the service should page a tenant-safe broader result set and apply the missing filter locally.

**Alternatives considered**:

- Reuse the failing related search shape and rely on tests: rejected because real runtime behavior is already known to fail.
- Disable all v8.8 search: rejected because some top-level incident search paths may still be tenant-safe and useful.

## Decision: Count Totals After The Final Filter Pipeline

**Rationale**: `--total` is useful only if the count reflects the same results the user would see. Backend totals are exact only when every filter is server-side compatible. If local message filtering or compatibility filtering runs, c8volt must page and count locally.

**Alternatives considered**:

- Always display backend totals: rejected because local post-filtering can change the result set.
- Reject `--total` with local filters: rejected because operators need counts for the same filters they use for lists.

## Decision: Render Incident Age From `creationTime`

**Rationale**: Recent `get pi --with-incidents` work added `creationTime`; this feature requires human rows to also include an age derived from it. Missing or unparsable `creationTime` should degrade the age field without crashing output.

**Alternatives considered**:

- Omit age when listing incidents: rejected by the issue.
- Put age in JSON as a derived field: rejected for now because JSON should preserve full incident fields and avoid derived display-only drift unless existing conventions require it.
