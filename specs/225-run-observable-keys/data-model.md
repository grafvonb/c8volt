# Data Model: Run Confirmation Observes Real Process Instance States

## Created Process Instance

Represents a process instance created by a run-style command.

**Fields**:

- `key`: Unique process instance key returned by creation or lookup.
- `bpmnProcessId`: BPMN process identifier when available.
- `processDefinitionKey`: Process definition key when available.
- `processVersion`: Process definition version when available.
- `state`: Actual observed lifecycle state after confirmation.
- `startDate`: Observed or request-time start timestamp.
- `startConfirmedAt`: Timestamp set when creation confirmation observes an accepted state.
- `tenantId`: Effective tenant associated with the created process instance.

**Validation rules**:

- A non-empty key is required for wait-based confirmation, except for existing v8.7 compatibility behavior where the API cannot return a usable key.
- `state` must reflect the observed process instance state when confirmation performs a lookup.
- Creation confirmation must not synthesize success for absent/not-found observations.

## Observed Lifecycle State

Represents the state that satisfies creation confirmation.

**Allowed confirmation states**:

- `active`
- `completed`
- `canceled`
- `terminated`

**Non-confirming states**:

- `absent`
- `unknown`
- any not-found/non-observable condition

**State transitions**:

```text
created request accepted
  -> observable active
  -> observable completed
  -> observable canceled
  -> observable terminated
  -> absent/not-found/unknown (not confirmed)
```

## Keys-Only Output

Represents line-oriented created process instance keys emitted by `c8volt run pi --keys-only`.

**Fields**:

- One process instance key per stdout line.

**Validation rules**:

- No human labels, summaries, warnings, or `found:` totals may appear on stdout in keys-only mode.
- Empty keys must not be emitted.
- Output must be directly usable as stdin for `c8volt expect pi --state <state> -`.

## Relationships

- `run pi`, `deploy --run`, and `embed deploy --run` create `Created Process Instance` values through the shared process facade/service path.
- `run pi --keys-only` renders `Created Process Instance.key` values only.
- `expect pi` consumes keys and applies strict `Observed Lifecycle State` expectations selected by the user.
