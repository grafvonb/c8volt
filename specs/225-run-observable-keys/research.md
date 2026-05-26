# Research: Run Confirmation Observes Real Process Instance States

## Decision: Creation confirmation should wait for observable lifecycle states

**Rationale**: The current v8.7, v8.8, and v8.9 process-instance creation flows call `WaitForProcessInstanceState` with only `StateActive`. The issue requires successful confirmation when the created instance is observable as `active`, `completed`, `canceled`, or `terminated`. Reusing the existing waiter with a broader desired state set keeps retry, timeout, logging, and absent handling in the established service boundary.

**Alternatives considered**:

- Add a new `run --expected-status` flag. Rejected by the issue because strict lifecycle expectations belong in `expect pi`.
- Treat any non-error lookup as success without checking state. Rejected because absent or unknown/non-observable states must not confirm creation.
- Implement special-case command-layer polling. Rejected because wait mechanics belong in internal services, not `cmd`.

## Decision: Preserve strict `expect pi` behavior

**Rationale**: `expect pi` already accepts explicit states including `active`, `completed`, `canceled`, `terminated`, and `absent`. Broad run confirmation is a creation-success rule only; downstream strict assertions remain in `expect pi`.

**Alternatives considered**:

- Loosen `expect pi` to treat all observable states as success. Rejected because it would break the command's explicit assertion contract.

## Decision: Render `run pi` through existing process-instance list views for non-JSON modes

**Rationale**: `run pi` currently calls `renderCommandResult`, which only renders shared JSON envelopes for full-contract commands. The existing `listProcessInstancesView` already supports one-line, JSON, and keys-only rendering for `process.ProcessInstances`. Reusing it keeps output semantics aligned with `get pi`.

**Alternatives considered**:

- Add a dedicated run renderer. Rejected unless implementation proves the shared process-instance list renderer cannot satisfy run-specific needs.
- Write keys manually in `run_processinstance.go`. Rejected because `listOrJSONFlat` already owns keys-only list semantics.

## Decision: Treat deploy-run commands as consumers of shared creation confirmation

**Rationale**: `deploy --run` and `embed deploy --run` call `CreateProcessInstances`; once service confirmation accepts observable lifecycle states, those commands gain the behavior without separate command-specific polling. They do not need keys-only changes unless the implementation discovers they render created instances in scope.

**Alternatives considered**:

- Add deployment-specific confirmation logic. Rejected because it duplicates service behavior.

## Decision: Update command contract and generated docs when output behavior changes

**Rationale**: The repository constitution requires CLI docs and machine contract discovery to match executable behavior. `run pi --keys-only` must be discoverable via help/capabilities and documented with the pipeline pattern.

**Alternatives considered**:

- Update README only. Rejected because generated CLI docs and command metadata would drift.
