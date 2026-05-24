# Data Model: Ops Paged Discovery Scope

## Discovery Scope Status

Records whether a frozen discovery scope represents the full matching population or an explicit operator cap.

**Fields**:
- `status`: workflow step status for discovery/frozen-set creation.
- `complete`: true when discovery reached the end of the matching population.
- `limited`: true when discovery stopped because `--limit` was reached.
- `limit`: operator-supplied total cap, omitted or zero when unlimited.
- `batchSize`: page-size value used for discovery.
- `pages`: number of backend pages inspected.
- `candidatesSeen`: number of raw candidates inspected before filtering or dedupe, when meaningful.
- `candidatesFrozen`: number of candidates retained in the frozen set.

**Validation rules**:
- `complete` and `limited` cannot both be true for the same scope.
- `limited` requires `limit > 0`.
- `batchSize` must be positive after normalization.
- `pages` must be zero only when keyed/frozen reuse avoided search discovery.

## Frozen Candidate Set

The immutable candidate collection approved for dry-run, confirmation, mutation, automation output, and reports.

**Fields**:
- Candidate keys for the workflow type: incident keys, process-instance keys, or process-definition keys.
- Candidate detail rows where the existing report model already carries them.
- Duplicate-key lists where dedupe occurred.
- Skipped candidate lists where a candidate could not produce a mutable target.
- Discovery scope status.

**Relationships**:
- Incident purge freezes incident rows and unique process-instance candidate keys.
- Repair incident freezes incident rows and unique incident/process-instance/job/variable-scope keys.
- Repair process-instance freezes selected process-instance keys and active incidents found for them.
- All-process-definitions purge freezes process-definition keys and process-definition detail rows.

## Discovery Page Request

The normalized page request sent to backend search services.

**Fields**:
- `size`: page size from normalized `--batch-size` or existing service default.
- `from`: offset cursor fallback for APIs that use offset paging.
- `after`: cursor for APIs that return forward cursors.

**Validation rules**:
- Use either cursor or offset semantics for a single request, not both.
- Advance with returned cursor when available; otherwise advance offset by the backend page size.

## Ops Report Discovery Section

The audit and machine-readable discovery section emitted by affected workflows.

**Fields**:
- Existing filters and candidate data.
- Discovery scope status fields.
- Existing notices for zero candidates, duplicates, skipped candidates, latest-only scope, and other semantic facts.

**Relationships**:
- Human one-line output, JSON output, automation output, and Markdown reports must all read from the same discovery or frozen-set status data.

## State Transitions

```text
Search requested
  -> Page discovery running
  -> Complete frozen scope
  -> Dry-run preview or confirmation
  -> Confirmed mutation reuses frozen scope

Search requested
  -> Page discovery running
  -> User-limited frozen scope
  -> Dry-run preview or confirmation
  -> Confirmed mutation reuses frozen scope

Frozen keys supplied after confirmation
  -> Reconstructed frozen scope
  -> Confirmed mutation reuses supplied keys without search
```
