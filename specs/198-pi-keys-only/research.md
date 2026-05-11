# Research: get incident process-instance key output

## Decision: Add a command-local `--pi-keys-only` flag

**Rationale**: The requested behavior applies only to incident rows and changes which existing incident field is emitted. A command-local flag avoids changing the global `--keys-only` contract, which already means "the key for this resource" across commands.

**Alternatives considered**:

- Add a new global render mode: rejected because other commands do not have a second key identity.
- Reinterpret `--keys-only` for `get incident`: rejected because existing incident-key pipelines would break.

## Decision: Render from existing incident result items

**Rationale**: `get incident` already receives `incident.ProcessInstanceIncidentDetail` values containing both `IncidentKey` and `ProcessInstanceKey`. Rendering a different field is a view concern and does not require facade or service changes.

**Alternatives considered**:

- Add a second query for process instances: rejected because it would add remote calls without changing the selected incident set.
- Add a new result model: rejected because it duplicates existing incident row fields.

## Decision: Preserve duplicate process instance keys

**Rationale**: One process instance can have multiple incidents. The incident command is selecting incident rows, so output should remain one line per selected incident row even when the process instance key repeats.

**Alternatives considered**:

- Dedupe in `get incident`: rejected because it hides incident multiplicity and conflicts with the issue requirement.
- Add an opt-in dedupe flag: rejected for this focused version; shell users can pipe to `sort -u` when needed.

## Decision: Skip missing process instance keys

**Rationale**: The clarification gate selected skip behavior. Skipping avoids malformed blank-line pipeline input while allowing otherwise useful results to continue.

**Alternatives considered**:

- Fail the command: rejected by clarification answer.
- Emit blank lines: rejected because downstream key readers treat blank and malformed input differently and blank output is not useful.

## Decision: Treat `--pi-keys-only` as mutually exclusive with other output modifiers

**Rationale**: The flag is a complete line-output mode. Combining it with JSON, totals, incident keys, or message formatting creates ambiguous output expectations and risks unsafe pipelines.

**Alternatives considered**:

- Let later flags win: rejected because inherited/global output modes should not depend on argument order.
- Allow message flags: rejected because message formatting has no effect on key-only output.

## Decision: Keep delete process-instance cleanup at command boundary

**Rationale**: `cancel pi` already dedupes merged flag/stdin keys immediately. Aligning `delete pi` with the same pattern reduces duplicate planning counts while preserving service-layer dedupe and downstream safety.

**Alternatives considered**:

- Leave dedupe only in the service planning layer: rejected because command-local prompts and dry-run previews can briefly reflect duplicate requested keys.
- Dedupe `get incident --pi-keys-only`: rejected because that would violate the incident output contract.
