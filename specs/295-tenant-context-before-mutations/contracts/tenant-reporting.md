# Contract: Tenant Visibility Before Ops Mutations

## Affected command surface

- `ops purge all-process-definitions` and alias `apd`
- `ops purge orphan-process-instances`
- `ops purge process-instances-with-incidents`
- `ops execute retention-policy`
- `ops repair incident`
- `ops repair process-instance`

Existing command/flag validation, aliases, output support, authorization, and mutation semantics are unchanged. Do not add support for a mode that a command currently rejects.

## Permitted human output order

1. Complete local input and report-path validation and resolve discovery versus explicit-key mode.
2. On durable stderr, display applicable override provenance/warnings, then the selection scope.
3. Begin existing discovery activity, scope/progress reporting, resolution, and impact validation.
4. After validated scope is available, display affected tenants and applicable unknown-tenant warning on durable stderr.
5. Ask the existing confirmation question when required.
6. Perform existing mutations and operational confirmation.
7. Render the ordinary final result without repeated tenant context; persist complete audit evidence when requested.

Auto-confirm skips step 5 only. Each applicable tenant line appears once across the invocation. Existing operational progress remains in its current order. No additional request or discovery pass is introduced.

### Selection labels and warnings

Reuse existing wording:

- Named filter: `selection scope: <tenant> only`
- No filter: `selection scope: unfiltered across accessible tenants`
- Explicit keys: `selection scope: explicit resource keys; tenant filter not applied`
- Clearing configured filter with an empty tenant: `--tenant "" overrides the configured tenant filter; selection is unfiltered`
- Clearing configured filter with all-tenants: `--all-tenants overrides the configured tenant filter; selection is unfiltered`

Existing configured-tenant labels and changed named-tenant informational lines remain intact. Absent/equal overrides remain silent. Explicit-key operations retain existing omission of inapplicable discovery override commentary.

### Affected scope

One known tenant gives one information-level `affected tenants: <tenant>` line. Multiple known tenants give the same label once at warning level, with unique deterministically ordered values. Unknown metadata gives one existing warning with the unknown target count, in addition to any known-tenant summary. Do not embed an extra severity prefix in message text. Empty validated scope adds neither affected tenants nor an unknown warning.

A validated preview may report affected scope even though dry-run performs no mutation. Failed discovery/validation must not be described as a complete scope. Preserve the existing no-work, force-blocked, declined, and mutation-failure outcomes.

## Protected modes and audit data

Use `opsProgressChannelForMode` for early tenant output. JSON, automation, quiet, and supported keys-only modes receive no new human tenant preflight output on either stream. Existing error diagnostics and quiet failure behavior remain unchanged. JSON stays in the existing shared envelope where supported. Keys-only, if supported, remains exactly one key per line.

Reports retain the existing complete `tenantContext` and evidence representation, including known tenant IDs, unknown counts, cross-tenant status, and applicable existing warnings. Legacy audit `tenantId` must not falsely identify unfiltered or explicit-key work as named-filter work. JSON and Markdown report writers ignore human emission flags. Human-only override provenance does not become a new audit field.

## Internal/public progress notification contract

`tenant_scope` is a new additive event kind with an optional typed tenant-evidence payload in the existing ops progress envelope. It is emitted synchronously after successful scope construction and before mutation. The facade maps and copies facts only. Existing progress consumers can ignore the new kind; other event meanings and final serialized command/report contracts remain unchanged.

## Acceptance matrix

| Dimension | Required coverage |
| --- | --- |
| Commands | All six with auto-confirm; APD alias exercised; both repair keyed/search paths |
| Confirmation | Interactive accept and decline; `-y`/`--auto-confirm`; existing automation implicit confirmation |
| Selection | Named configured filter; unfiltered; changed named override; explicit empty override; `--all-tenants`; absent/equal override; explicit keys where supported |
| Evidence | One named tenant; multiple known tenants; default-tenant evidence; unknown-only; known plus unknown; multiple plus unknown; duplicates; empty scope |
| Modes | Supported ordinary human, verbose/debug, JSON, quiet, automation, and other advertised modes; unsupported modes retain validation |
| Lifecycle | Dry-run; no work; invalid input; discovery/impact failure; force-blocked; mutation failure; repeated service callbacks within interactive flow; repeated command execution |
| Output proof | Selection exists at first discovery/activity; affected lines exist at first mutation; prompt sees full context; applicable lines each occur once |
| Data proof | Audit JSON/Markdown and structured context remain complete despite deduplication; copied evidence cannot mutate service plan |
| Backend proof | Request counters unchanged; mutation target set unchanged; repair variable updates included as first mutation |

Use representative combinations for the broader matrix while requiring command-level auto-confirm ordering coverage for every listed command. Service-level tests cover every distinct notification placement.
