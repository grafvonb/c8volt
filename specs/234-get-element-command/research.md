# Research: Runtime Element Instance Command

## Decision: Model `get element` after `get job` for direct lookup plus paged search

**Rationale**: `get job` already has the closest shape: a singular runtime object, Camunda 8.8/8.9 support, Camunda 8.7 unsupported behavior, direct lookup by key, search filters, paging, totals, JSON, and compact rows. Reusing that shape minimizes command surprises and keeps validation local to `cmd`.

**Alternatives considered**:
- `get incident`: strong paging and selector-validation precedent, but incident has extra process-instance-key output and incident-message behavior that would add unrelated concepts to element output.
- `get pi`: useful for page continuation and timestamp formatting, but process-instance enrichment and recovery behavior are broader than this standalone read-only command.

## Decision: Add a dedicated `c8volt/element` facade and `internal/services/element` service

**Rationale**: The spec explicitly calls for a reusable service/facade capability for a later `get pi --with-elements` story. A dedicated package keeps the public facade thin and avoids coupling runtime element inspection to jobs, incidents, or process-instance code.

**Alternatives considered**:
- Place element lookup under `c8volt/process`: runtime element instances are related to process instances, but the standalone command and future reuse would make process facade responsibilities broader and less clear.
- Implement directly in `cmd`: rejected by repository layering rules; paging, conversion, and version-specific behavior belong below the command layer.

## Decision: Use generated Camunda 8.8/8.9 element-instance operations

**Rationale**: The generated clients already expose `SearchElementInstancesWithResponse` and `GetElementInstanceWithResponse` for v8.8 and v8.9. Using them avoids hand-written HTTP calls and keeps version-specific request/response differences in adapter packages.

**Alternatives considered**:
- Hand-edit generated clients: rejected because the required operations already exist.
- Search by key only: rejected because the spec requires direct lookup and search filters.

## Decision: Return unsupported-operation errors for all element operations on Camunda 8.7

**Rationale**: The generated v8.7 client lacks runtime element instance lookup/search operations. The existing job service pattern returns domain unsupported errors with clear wording for 8.7, which maps cleanly through `c8volt/ferrors`.

**Alternatives considered**:
- Hide the command for v8.7: rejected because c8volt command availability is generally stable across configured versions, with runtime unsupported errors where necessary.
- Attempt partial behavior using older endpoints: rejected because it would not satisfy runtime element instance inspection and risks confusing static/variable element operations with runtime element records.

## Decision: Search filters use AND semantics and `--key` is exclusive

**Rationale**: This was clarified in the spec. AND semantics are predictable for operators and testable across service and command layers. Keeping `--key` mutually exclusive prevents direct lookup from silently ignoring filters.

**Alternatives considered**:
- Allow only one search selector: too restrictive for operational diagnosis.
- Let `--key` take precedence: surprising and weaker for validation because extra filters would be accepted but ignored.

## Decision: Preserve existing get-command output contracts

**Rationale**: Human output should use compact aligned rows, primary key first, short tags, existing timestamp helpers, incident markers, and final `found: N`. `--keys-only`, `--total`, and `--json` must stay script-safe and quiet.

**Alternatives considered**:
- Add element-specific summary or loop counts: explicitly out of scope.
- Include backend request/page diagnostics by default: rejected by CLI UX rules; diagnostics belong behind verbose logging.

## Decision: Treat exact reported totals as authoritative, otherwise count pages

**Rationale**: This follows the spec and existing `get incident`/`get job` behavior. It gives efficient totals when the backend provides exact counts while preserving correctness for lower-bound or paged totals.

**Alternatives considered**:
- Always page for totals: simpler but slower when exact totals are available.
- Always trust reported totals: unsafe when the backend reports capped or lower-bound totals.
