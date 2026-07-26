# Data Model: Slow Process Timeline Readability

This feature reuses the slow-process analysis data model from `specs/244-slow-process-analysis/data-model.md`. It adds human rendering concepts only; JSON and keys-only output contracts remain unchanged.

## Analysis Result

Existing render-independent result for slow-process analysis.

### Existing Fields Used By This Feature

| Field | Type | Notes |
|-------|------|-------|
| `items` | Analyzed Process Instance[] | Ordered root results rendered in every output mode. |
| `count` | int | Final process-instance count. |
| `items[*].timeline` | Timeline Row[] | Complete analyzed detail rows after existing detail filters. Default human output summarizes this list; full-timeline human output renders it directly. |

### Validation Rules

- The result remains complete before human summarization.
- JSON output renders the existing result without summary-only fields.
- Keys-only output renders only process-instance keys from `items`.

## Human Timeline Display Mode

Command-local display choice for human output.

| Mode | Trigger | Behavior |
|------|---------|----------|
| `hotspot_summary` | Default human output | Render root rows with a `slowest elements:` section containing selected hotspot rows and hidden-row summary when rows are omitted. |
| `full_timeline` | `--with-full-timeline` | Render the complete chronological timeline using the existing detail row style. |

### Validation Rules

- The display mode affects human output only.
- The display mode does not change process-instance selection, root sorting, duration calculation, detail filtering, JSON output, or keys-only output.
- The British and American command spellings use the same display-mode behavior.

## Timeline Row

Existing analyzed element or transition row.

### Existing Fields Used By Summary Selection

| Field | Type | Required For Summary | Notes |
|-------|------|----------------------|-------|
| `kind` | enum | Yes | Distinguishes element rows from transition rows. |
| `elementInstanceKey` | string | No | Hidden in default summaries unless needed for incident identity. |
| `elementId` | string | Yes for elements | Identifies the BPMN element. |
| `type` | string | Yes for elements | Identifies the runtime element type. |
| `state` | string | Yes for elements | Determines active-row inclusion. |
| `duration` | string | No | Human-readable duration when available. |
| `durationMillis` | int64 | Conditional | Used with root duration to determine process-duration share. |
| `durationAvailable` | bool | Yes | Rows without measurable duration are not selected by duration share alone. |
| `hasIncident` | bool | Yes for elements | Forces element row visibility in default summary. |
| `incidentKey` | string | No | May be shown when needed to identify an incident row. |
| `processDurationShare` | int | Conditional | Existing rounded share value; summary selection should treat completed contributors at or above 1% as visible. |

### Summary Selection Rules

- Completed element contributors with process-duration share at or above 1% are visible in default human output.
- Active element rows are visible even when their current share is below 1%.
- Incident-bearing element rows are visible even when their share is below 1%.
- A row that matches multiple visibility reasons appears once.
- Completed rows below 1%, zero-duration transition rows, and other instant or very-fast rows are omitted from default human output unless they match an active or incident rule.
- Full-timeline mode renders the complete timeline rows supplied by the existing analysis result.

## Hotspot Summary

Human-only projection of an analyzed process instance.

| Field | Type | Notes |
|-------|------|-------|
| `root` | Analyzed Process Instance | The root row rendered before summary rows. |
| `visibleRows` | Timeline Row[] | Rows selected by duration-share, active, or incident rules. |
| `hiddenRowCount` | int | Number of analyzed timeline rows omitted from default human output. |
| `hiddenRowKinds` | string | Compact human wording for omitted instant, fast, or timeline rows. |

### Validation Rules

- `hiddenRowCount` excludes the root row.
- No hidden-row summary appears when no rows are omitted.
- Hidden-row summary text names `--with-full-timeline`.
- Summary selection does not mutate or reorder the underlying analysis result used by JSON and keys-only output.
