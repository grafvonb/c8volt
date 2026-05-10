# Research: Process Instance Variable Updates

## Decision: Add A New `update` Root Command

Create `cmd/update.go` and register `update process-instance` / `update pi` below it, following the same Cobra layout and mutation metadata patterns used by `run`, `delete`, `deploy`, and other state-changing command families.

**Rationale**: The issue asks for `c8volt update` as a new root command family. Keeping a dedicated root command preserves discoverability, command contract metadata, and future room for other update leaves without overloading `run` or `get`.

**Alternatives considered**:

- Add variable update flags to `get pi`: rejected because `get` is read-only and would violate command semantics.
- Add update behavior to `run --vars`: rejected because `run --vars` is creation-time input and must remain unchanged.

## Decision: Reuse Existing Key Selection And Worker Controls

Use the repository's existing key parsing, stdin `-`, deduplication, worker, fail-fast, and no-worker-limit patterns for the update target set.

**Rationale**: The issue requires the same key behavior and bulk controls operators already use. Reusing those paths reduces validation drift and keeps automation behavior predictable.

**Alternatives considered**:

- Implement command-local key parsing: rejected because it risks different stdin and deduplication behavior.
- Restrict the first implementation to one key: rejected because multi-key and stdin key input are required acceptance criteria.

## Decision: Implement Variable Mutation At The Process-Instance Service Layer

Add a shared process-instance service API method for updating element-instance variables, implement it in v8.8 and v8.9 using generated Camunda client methods for `/element-instances/{elementInstanceKey}/variables`, and make v8.7 return an unsupported-version error before mutation.

**Rationale**: Existing process-instance variable lookup already lives in versioned process-instance services. Keeping mutation there preserves version-specific API ownership and lets the facade/command stay version-neutral.

**Alternatives considered**:

- Call generated clients directly from `cmd`: rejected because it bypasses the service/facade error and version layers.
- Add a separate variable service for process-instance-scope updates: rejected because the update endpoint is element-instance scoped and the issue uses process instance keys as element instance keys.

## Decision: Confirm Through Existing Variable Lookup

After accepted mutation, reuse the same backend lookup path as `get process-instance --key <key> --with-vars`, then compare only requested variable names using normalized JSON value comparison.

**Rationale**: The constitution requires operational proof. The existing variable lookup path already filters process-scope variables, and normalized comparison avoids false failures from serialized JSON formatting differences.

**Alternatives considered**:

- Trust the mutation response only: rejected because the issue requires read-model confirmation.
- Compare raw JSON strings: rejected because returned values may be serialized with different formatting.
- Fetch unrelated variables into the success criteria: rejected because unrelated variables should not affect requested update confirmation.

## Decision: Keep `--no-wait` As Submitted/Accepted Output

When `--no-wait` is supplied, return after each mutation request is accepted and mark results as submitted rather than confirmed.

**Rationale**: This mirrors existing `run`, `deploy`, and delete no-wait behavior, gives automation a lower-latency path, and avoids claiming read-model visibility that was not checked.

**Alternatives considered**:

- Disable `--no-wait` for state-changing variable updates: rejected because the issue requires it.
- Poll briefly even with `--no-wait`: rejected because it makes the option ambiguous and less predictable.

## Decision: Update Docs Through Existing Generation Flow

Update Cobra help/examples and README, then regenerate generated CLI documentation with the repository's docs tooling.

**Rationale**: The new root command and flags are user-facing. The constitution requires docs to match shipped behavior, and generated docs should come from source metadata.

**Alternatives considered**:

- Hand-edit generated CLI docs only: rejected because it risks drift from command metadata.
