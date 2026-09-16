# Research: Consistent Listener Timestamps

## 1. Timestamp availability and adapter ownership

**Decision**: Copy `CreationTime` and `EndTime` directly from generated `JobSearchResult` into domain jobs in `internal/services/job/v88/convert.go`, `v89/convert.go`, and `v810/convert.go`, in `fromJobSearchResult`. Leave v87 unsupported.

**Rationale**: The checked-in generated models in `internal/clients/camunda/{v88,v89,v810}/camunda/client.gen.go` already contain both optional `*time.Time` fields. All three service converters drop them today. The generated creation-time comment says it is present for jobs created after 8.9, so a field's presence in a client does not guarantee its availability in a response. Preserve whatever is actually supplied, rather than hard-coding a version cutoff. `internal/services/job/v87/service.go` explicitly rejects job retrieval/search as requiring 8.8 or newer.

**Alternatives considered**: Regenerate clients (unnecessary); force timestamps on older jobs or infer from deadlines (incorrect); version-gate otherwise supplied values (would discard valid facts); add unsupported-version fallback (outside scope).

## 2. Lossless domain and public representations

**Decision**: Add `CreationTime *time.Time` and `EndTime *time.Time` with `json:"creationTime,omitempty"` and `json:"endTime,omitempty"` to domain `Job` and `RuntimeListenerJob`, public `job.Job`, and public `RuntimeListenerJob` in `element`, `process`, and `ops`. Copy both fields through each existing conversion.

**Rationale**: `internal/domain/job.go:RuntimeListenerJobFromJob` projects jobs into listener records. `c8volt/job/client.go:fromDomainJob` serves job retrieval and list/page conversion. Each of `c8volt/element/convert.go`, `c8volt/process/convert.go`, and `c8volt/ops/convert.go` owns `fromDomainRuntimeListenerJob`. Updating every boundary prevents the original data-loss bug from moving downstream. The existing deadline already uses an optional time pointer. Omitted and explicit-null source fields decode to nil; both remain omitted in public JSON.

**Alternatives considered**: Strings (unnecessary parsing and inconsistent with deadline); non-pointer times (cannot reliably distinguish absence); a public shared listener type migration (unrelated compatibility work).

## 3. Enrichment and operational behavior

**Decision**: Keep enrichment queries, ownership matching, ordering, pagination, and duration calculations unchanged. Verify that timestamps survive existing enrichment results.

**Rationale**: `internal/services/element/enrichment.go` and `internal/services/processinstance/enrichment.go` already project jobs through `RuntimeListenerJobFromJob` and attach complete listener records. Slow analysis under `internal/services/ops/slow_process_analysis.go` consumes listener enrichment. The missing values require mapping changes, not additional discovery or calculations.

**Alternatives considered**: Fetch historical times separately (adds requests and changes availability semantics); derive listener duration or worker execution time (not requested and not supported by creation/end semantics).

## 4. Consistent human rendering with stable columns

**Decision**: Add a small command-view helper in `cmd/cmd_views_listener.go` that returns exactly three optional columns in `s:`, `e:`, `d:` order from creation time, end time, deadline, state, and timezone-offset setting. Use it from the existing element, process-activity, and slow-analysis listener row functions. Return empty strings for unavailable or ineligible columns.

**Rationale**: `flatRowElementListenerWithTimezone`, `flatRowProcessInstanceElementListenerWithTimezone`, and `flatRowOpsSlowProcessAnalysisListenerWithTimezone` repeat the same unconditional deadline rendering today. Process get and walk share the process-activity renderer. `cmd/cmd_views_flat.go:formatFlatRows` aligns by column position, so omitting entries from individual rows would misalign error and timestamp columns in mixed-state results. A narrow helper centralizes the predicate without introducing a new public abstraction or changing surrounding row grammar.

Use `toolx.FormatTime` unchanged. `toolx/timestamp.go` formats full date/time at millisecond precision and optionally a numeric timezone offset. The issue's shortened time-only example illustrates semantics; it does not authorize replacing the repository's full-date display format. Show `d:` only for exact state `ACTIVATED`. Keep supplied `s:` and `e:` independent of state. Keep the ordinary job renderer in `cmd/cmd_views_job.go` unchanged.

**Alternatives considered**: Duplicate the predicate in three renderers (risks renewed drift); refactor all listener row fields (broader than needed); change `toolx` formatting (affects unrelated timestamps); suppress deadlines in models (breaks programmatic data).

## 5. Validation and documentation

**Decision**: Extend existing fixture-driven service, facade, and command tests. Cover both listener kinds, mixed states and missing-field combinations, JSON omission, all four commands, and duration invariance. Update README and the four command descriptions, then regenerate documentation with `make docs-content` during implementation.

**Rationale**: Relevant coverage already exists in the three versioned job `service_test.go` files, `c8volt/{job,element,process,ops}/client_test.go`, `internal/services/processinstance/enrichment_test.go`, `internal/services/ops/slow_process_analysis_test.go`, and `cmd/{get_element_test.go,get_processinstance_test.go,walk_test.go,ops_analyse_slow_process_instances_test.go}` plus the three view test files. Existing subprocess/fixture helpers support execution-path checks. `app.show_timezone_offset` is the existing display control; no new flag is needed.

**Alternatives considered**: Only testing view helpers (would miss discarded transport fields); relying solely on live completed listeners (availability can vary); running runtime tests for this documentation-only planning change (prohibited by constitution principle III).

## Resolution

All technical questions are resolved from repository evidence. No new dependency, generated-client update, schema migration, network research, or user clarification is required. Phase 1 may proceed.
