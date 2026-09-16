# Data Model: Listener Timestamps

## Entities and fields

| Representation | Location | Change |
| --- | --- | --- |
| Job | `internal/domain/job.go` | Add optional creation and end times. |
| RuntimeListenerJob | `internal/domain/job.go` | Add the same fields for element-owned listener projections. |
| Job | `c8volt/job/model.go` | Expose both optional values to public job consumers. |
| RuntimeListenerJob | `c8volt/element/model.go`, `c8volt/process/model.go`, `c8volt/ops/model.go` | Expose the same optional values in each existing facade. |

Each new field uses the same representation everywhere:

| Go field | Type | JSON name | Meaning |
| --- | --- | --- | --- |
| CreationTime | `*time.Time` | `creationTime,omitempty` | Recorded job creation instant, not worker execution start. |
| EndTime | `*time.Time` | `endTime,omitempty` | Recorded job end instant; not inferred from state or deadline. |

Existing `Deadline *time.Time` with `deadline,omitempty` and `State string` remain unchanged. The state controls human deadline visibility only. Nil is absence, and fields are independent. No synthetic defaults, chronology correction, or conversion to local wall time is introduced. Existing source timestamp decoding errors retain their existing handling; this feature does not add an alternative parser.

## Mapping and relationships

1. Generated `JobSearchResult` in each v88/v89/v810 client supplies optional pointers.
2. Version-owned `fromJobSearchResult` maps them to domain `Job`.
3. Public job `fromDomainJob` preserves them for get, search, and paged results.
4. `RuntimeListenerJobFromJob` preserves them when a job becomes an element-owned listener record.
5. Existing element/process enrichment attaches records by element instance ownership. Unmatched jobs remain omitted and existing ordering is preserved. Slow analysis continues consuming these records without recalculating listener-based durations.
6. Each facade's `fromDomainRuntimeListenerJob` preserves both values in the public listener object.

Retain `Listeners *[]RuntimeListenerJob` semantics: nil means not requested; a pointer to an empty slice means requested with no matches. Adding timestamp fields must not change that distinction.

## Validation and lifecycle rules

- Non-nil creation/end values retain the same instant and timezone offset through mappings.
- Missing or null source values remain nil and are omitted from public JSON.
- Human rows show each available creation/end time independently of state.
- Only exact `ACTIVATED` plus non-nil deadline produces human `d:`.
- JSON preserves supplied deadlines in every state.
- No new state transitions, mutations, persistence, migrations, duration values, or validation of lifecycle ordering are introduced.
- v87 lookup remains unsupported; v88/v89/v810 responses with missing fields remain valid.
