# CLI Contract: Runtime Listener Jobs Under Elements

## Command Surface

The feature adds `--with-listeners` to these command paths:

```text
c8volt get element -k <element-instance-key> --with-listeners
c8volt get element --pi-key <process-instance-key> --with-listeners
c8volt get pi -k <process-instance-key> --with-elements --with-listeners
c8volt walk pi -k <process-instance-key> --with-elements --with-listeners
c8volt ops analyse slow-process-instances -k <process-instance-key> --with-listeners
```

`walk pi` supports listener enrichment in all existing element-enabled walk modes:

```text
c8volt walk pi -k <process-instance-key> --with-elements --with-listeners
c8volt walk pi -k <process-instance-key> --children --with-elements --with-listeners
c8volt walk pi -k <process-instance-key> --parent --with-elements --with-listeners
c8volt walk pi -k <process-instance-key> --flat --with-elements --with-listeners
```

## Flag Contract

| Flag | Type | Required | Compatibility |
|------|------|----------|---------------|
| `--with-listeners` | bool | No | Requires an element context and output mode that can represent nested listener details. |
| `--with-elements` | bool | Required for `get pi` and `walk pi` listener enrichment | Provides the owning element rows for listener nesting. |
| `--key`, `-k` | string/string slice by command | Required for keyed examples | Identifies process instance or element instance target. |
| `--pi-key` | string | Required for element search listener enrichment when `--key` is absent | Selects elements by owning process-instance key. |
| `--children` | bool | No | May combine with `--with-elements --with-listeners` for walk descendants. |
| `--parent` | bool | No | May combine with `--with-elements --with-listeners` for walk ancestry. |
| `--flat` | bool | No | May combine with `--with-elements --with-listeners` for flat family walk output. |
| `--json` | bool | No | Emits listener arrays under element objects only when listeners are requested. |
| `--keys-only` | bool | No | Must fail when combined with `--with-listeners`. |

## Validation Contract

- `get pi --with-listeners` fails unless `--with-elements` is also set.
- `walk pi --with-listeners` fails unless `--with-elements` is also set.
- `walk pi --keys-only --with-listeners` fails before traversal or listener lookup.
- `get element --keys-only --with-listeners` fails before element or listener lookup.
- `ops analyse slow-process-instances --keys-only --with-listeners` fails before analysis lookup.
- `--with-listeners` may not be combined with count-only modes such as `--total`.
- Camunda 8.7 returns the established unsupported job-search/listener-lookup error style.

## Human Output Contract

Listener rows are nested under the element row that owns them.

Element-with-listeners grammar:

```text
<element-row>
└─ listeners:
   ├─ <jobKey> <kind> <listenerEventType> <state> type:<type> retries:<retries> [worker:<worker>] [deadline:<deadline>] [ec:<errorCode>] [err:<errorMessage>]
   └─ <jobKey> <kind> <listenerEventType> <state> type:<type> retries:<retries>
```

Process-instance activity output nests listeners inside the existing `elements:` section:

```text
<process-instance-row>
└─ elements:
   ├─ <element-row>
   │  └─ listeners:
   │     └─ <listener-row>
   └─ <element-row>
```

When variables, incidents, and elements are requested together, existing top-level activity section order is preserved:

```text
vars:
incidents:
elements:
```

For `walk pi`, listener rows must remain inside the owning process instance's `elements:` block. Child process instance rows remain process-tree rows and must not appear as children of listener rows.

For slow-process analysis, listener rows appear under element timeline rows when requested. Transition rows never receive listener rows.

## Empty And Unmatched Listener Behavior

- Human output may omit an empty `listeners:` block for elements with no listener jobs.
- Structured output includes an empty `listeners` array for each element when listener enrichment is requested.
- Runtime listener jobs without a matching element instance key are omitted from enriched output.

## Error Contract

- Validation errors must name the incompatible flag combination or missing element context.
- Remote listener lookup errors fail the whole command and must not render partial success output.
- Unsupported-version errors must match nearby job or element lookup unsupported-version style.
