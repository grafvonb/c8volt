# Research: Runtime Listener Jobs Under Elements

## Decision: Attach Listener Jobs During Element Enrichment

Extend the existing process-instance element enrichment path so requested listener jobs are attached to `ProcessInstanceElement` rows by matching `elementInstanceKey`.

**Rationale**: All in-scope output shapes put listeners under elements, and the existing process-instance activity model already carries requested-but-empty `elements` arrays and renders `elements:` below process-instance rows. Extending that path keeps `get pi`, `walk pi`, and element-oriented views consistent while preserving the command-layer boundary.

**Alternatives considered**:

- Add a top-level listener enrichment model beside elements: rejected because the spec requires listener rows below owning elements and clarified that unmatched jobs are omitted.
- Build listener rendering only in `cmd`: rejected because multiple commands need the same ownership and JSON behavior, and command code must not call generated clients or versioned service implementations directly.

## Decision: Use Runtime Job Search For Listener Source Data

Use the existing job service search capability with `processInstanceKey` and listener job kind filters for `EXECUTION_LISTENER` and `TASK_LISTENER`.

**Rationale**: The current domain and public job models already expose job key, state, retries, type, worker, kind, listener event type, process instance key, element instance key, element ID, tenant, and error details. The generated v8.8/v8.9 clients already support job search filters for process instance, element instance, kind, and listener event type. Camunda 8.7 job search already returns the repository's established unsupported-version style error.

**Alternatives considered**:

- Read static BPMN listener metadata: rejected because the feature explicitly targets runtime listener jobs and operator-visible retries/failures.
- Require one job lookup per element: rejected because process-instance scoped job search can collect the listener jobs needed for all elements on one selected process instance.

## Decision: Fetch Listener Jobs Per Selected Process Instance

For each process instance selected for enrichment, search listener jobs once for the process instance, filter to listener kinds, then group matched jobs by `elementInstanceKey`.

**Rationale**: This matches the issue's recommended direction and scales with selected process instances rather than selected elements. It also preserves existing traversal and process-instance selection order because listener enrichment is applied after element/process selection is complete.

**Alternatives considered**:

- Search jobs separately for each element instance key: rejected because it can multiply remote calls on dense processes.
- Search jobs globally without process-instance filters: rejected because it risks broad tenant-visible scans and makes ownership filtering less predictable.

## Decision: Omit Unmatched Listener Jobs

Runtime listener jobs without a matching element instance key are omitted from listener-enriched output.

**Rationale**: Clarification on 2026-07-23 selected element-owned output only. This prevents a new top-level unmatched section from changing output contracts and avoids implying ownership where none can be proven.

**Alternatives considered**:

- Render unmatched jobs in a separate section: rejected by clarification and because it would introduce a new output surface not required by the issue.
- Fail when unmatched jobs exist: rejected because unmatched data should not block valid element-owned diagnostics.

## Decision: Preserve Requested-But-Empty JSON Semantics

When listeners are requested, each element object in structured output includes `listeners`, using an empty array for elements with no matched listener jobs. When listeners are not requested, listener fields are omitted.

**Rationale**: This mirrors existing element requested-but-empty conventions and lets automation distinguish unrequested enrichment from requested enrichment with zero matches.

**Alternatives considered**:

- Omit empty listener arrays: rejected because it weakens the requested-versus-unrequested contract.
- Add a process-level requested flag: rejected because existing JSON contracts express enrichment through field presence.

## Decision: Validation Fails Before Remote Listener Lookup

Reject `--with-listeners` without an element context and reject keys-only combinations before any element or listener lookup starts.

**Rationale**: c8volt favors deterministic CLI behavior and clear local validation. These combinations cannot represent nested listener rows and should not spend remote calls before failing.

**Alternatives considered**:

- Allow `--with-listeners` to imply `--with-elements` everywhere: rejected because the spec requires an element context and existing flags keep enrichment explicit.
- Allow keys-only to print listener job keys: rejected because keys-only for these commands prints process or element keys and cannot represent nested listener ownership.

## Decision: Extend Documentation And Command Metadata

Update help text, examples, command capability metadata, README guidance, and generated CLI docs for the new flag and output behavior.

**Rationale**: The constitution requires user-visible command behavior to match documentation, and this feature changes public CLI and structured output contracts.

**Alternatives considered**:

- Rely on tests and issue text only: rejected because operators and automation authors discover behavior through help, README, and CLI reference docs.
