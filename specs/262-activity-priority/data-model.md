# Data Model: Preserve High-Level Activity

## Activity Scope

Represents one active unit of transient work.

Fields:

- `id`: Unique identifier for balancing start/update/stop operations.
- `message`: Current visible candidate text for the scope.
- `importance`: Semantic activity level used for visibility selection.
- `startedOrder`: Monotonic order used to break ties between scopes with equal importance.
- `active`: Whether the scope can still be selected.
- `drawn`: Whether this scope's message has contributed to visible terminal output.

Validation rules:

- A scope without a non-empty message may be active but should not produce visible text beyond the spinner frame.
- Stopping a scope must be idempotent from the caller perspective.
- Scope selection must not leak stopped scopes.

State transitions:

```text
created -> active -> updated* -> stopped
```

## Activity Importance

Represents the relative user value of activity messages.

Values:

- `workflow`: Command or ops workflow progress such as discovery, preflight, frozen-scope counters, or high-level mutation progress.
- `batch`: Service-level bulk work such as creating, deleting, cancelling, updating variables, or deployment confirmation across a known set.
- `wait`: Waiter or poller activity such as waiting for process-instance state, incident resolution, or job retries.
- `http`: Individual Camunda request fallback activity.

Validation rules:

- `workflow` outranks `batch`, `wait`, and `http`.
- `batch` outranks `wait` and `http`.
- `wait` outranks `http`.
- `http` is visible only when no higher-importance active scope has a visible message.

## Visible Activity

Derived view of the currently displayed activity message.

Fields:

- `frame`: Spinner frame.
- `message`: Selected activity scope message.
- `width`: Terminal-safe width used for drawing and clearing.

Selection rules:

- Select the active scope with highest importance.
- For equal importance, select the most recently started active scope.
- Updates change the selected message only when they target the selected scope or cause a higher-importance scope to become selectable.
- When the selected scope stops, recompute from remaining active scopes.

## Activity Output Mode

Represents whether transient activity may be rendered.

Values:

- `human`: Transient terminal activity allowed when the stderr writer is interactive and indicators are enabled.
- `verbose`: Durable progress lines may be emitted by command renderers; transient activity remains terminal-only.
- `debug`: Diagnostic output may include low-level traces; transient activity remains terminal-only.
- `json`: No transient activity in stdout; output remains a single valid JSON result.
- `keys-only`: No transient activity in stdout; stdout remains one key per line.
- `quiet`: Non-error activity suppressed.
- `automation`: Deterministic unattended behavior with transient activity suppressed.

Validation rules:

- Machine-oriented modes must not receive spinner or activity text in stdout.
- Durable output is controlled by existing command renderers, not by the activity selector.

## Fallback Endpoint Label

Represents a resource-aware activity message for one known Camunda request path.

Fields:

- `method`: Request action category.
- `pathPattern`: Resource path family, normalized across version prefixes when applicable.
- `message`: Short operator-facing fallback text.

Validation rules:

- Message must not include host names, full URLs, access tokens, tenant secrets, or request bodies.
- Message should describe the resource action, not transport mechanics.
- Unknown paths may use generic fallback wording.

## Representative Command Family

Represents a behavior group used to validate shared activity behavior.

Fields:

- `family`: Command behavior category.
- `representativeCommands`: Commands chosen to exercise the shared behavior.
- `risk`: Why the family could regress the activity UX.

Families:

- Process-instance get/search: paging, totals, enrichment, orphan discovery.
- Process-instance mutation: cancel, delete, run with count, update variables.
- Process-definition mutation: delete impact, active process-instance handling, deployment confirmation.
- Wait/expect/resolve: process-instance state, incident resolution, job retries.
- Ops analysis/repair/purge: discovery, frozen scope, dependency expansion, nested enrichment.
- Simple Camunda-backed fallback: cluster, tenant, element, job, incident, resource lookup.
