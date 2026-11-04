# Contract: Variable Update Previews

Applies to `c8volt update process-instance` / `update pi` and `c8volt update user-task` / `update ut`, including existing aliases.

## Detailed planned writes

Retain current variable ordering, semicolon-separated detail layout, value rendering, and resource-key prefix. Insert destination annotations after variable names:

```text
~ customerTier (pi:123): "silver" -> "gold"
+ incident (pi:123): "buba"
```

For user tasks:

```text
~ customerTier (inherited, pi:123): "silver" -> "gold"
~ approvalLimit (inherited, element:456): 100 -> 200
~ reviewed (local, element:789): false -> true
+ incident (local, element:789): "buba"
```

These fragments apply to both normal confirmation plans and dry-run previews. Single-resource display eligibility remains unchanged. With multiple resources, details remain verbose-only. Unchanged-requested and untouched entries retain their existing syntax and are not converted into planned writes.

## Compact bulk behavior

Keep the existing header, counts, outcome wording, and `inherited target scopes:` prefix. A representative task summary is:

```text
inherited target scopes: pi:123, element:456
```

Preserve raw-key deduplication and first-seen ordering across task changes. Local-only bulk updates do not gain an inherited-target line. PI bulk updates retain their existing compact summary; detailed root annotations appear under verbose output.

## Incomplete legacy metadata

When a user-task plan lacks owner process identity, preserve `(inherited scope <key>)` and the existing bare summary key for unknown inherited destinations. Known local writes still identify the element scope. Do not guess a root, claim an unknown scope is an element, add warnings, perform a lookup, or reject the plan. Fully collected normal task plans carry the identity required for typed labels.

## Compatibility

- No commands, aliases, flags, or supported flag combinations change.
- JSON payloads, envelope status, omission rules, null/empty collections, and keys-only results remain identical for equivalent inputs. The new public metadata field is explicitly excluded from JSON.
- Preserve mode precedence, quiet suppression, tenant context, confirmation prompt text and stderr routing, default/EOF handling, auto-confirm, automation, and no-wait behavior.
- Preserve existing dry-run key-selection behavior for explicitly selected resources. A selected resource with unchanged values is not automatically equivalent to an empty selected scope; do not broaden this feature into output-contract repairs.
- Dry-run, no-op, and declined execution add no mutations. Scope labeling adds zero backend requests in any mode.
- No new activity, verbose service diagnostics, or debug messages are required.
- Final result rendering and accepted/confirmed/failure statuses remain unchanged.

## Public library surface

`task.UserTaskVariablePlan.ProcessInstanceKey` is an additive Go metadata field with `json:"-"`; both converters preserve it. No public method signatures or internal service interfaces change. Callers constructing plans without it retain previous execution behavior.

## Acceptance coverage

Use exact assertions for detail fragments, full human summaries, inherited bulk scope ordering, and unchanged/untouched formatting. At command execution level, cover single/bulk, normal/dry-run, verbose, local/root/intermediate destinations, shared targets, no-op/empty payload, errors/aborts, and supported machine/quiet combinations. JSON decoding must reach EOF after one envelope; capture stdout and stderr independently. Real terminal stdin is required for confirmation-path and prompt-free no-op checks.
