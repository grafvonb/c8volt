# Implementation Plan: Bulk Job Completion

**Branch**: `345-bulk-job-completion` | **Date**: 2026-10-20 | **Spec**: [spec.md](spec.md)

**Input**: [GitHub #345](https://github.com/grafvonb/c8volt/issues/345)

**Baseline**: `v4.3.4` / `019e4c0bd865b9cf675422b4d8c97c489c36f694`; intended PR base `main`.

## Summary

Extend `update job --complete` with collection keys, stdin, and job-search selection so an operator can explicitly acknowledge a backlog of pending canceling task listeners. Prepare the whole requested selection before mutation, freeze its unique keys, confirm once, then execute with bounded workers and one result per key.

Keep pagination, preparation, rechecks, completion submission, reconciliation, and aggregation in internal job services. Retain existing single-key behavior and output. Correct the shared job-search tenant omission in all three supported adapters. Add a bulk-only single-submission transport path so automatic retries cannot conceal an earlier uncertain completion attempt.

Completing a listener acknowledges its work without running its business logic. The command must state this consequence and must not infer process termination from job acceptance. This feature does not change cancellation commands, repair incidents, or depend on unreleased develop work.

## Technical Context

**Language/Version**: Go 1.26.0; repository toolchain Go 1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2, pflag 1.0.10, Viper 1.21.0, generated Camunda clients, `toolx/pool`, service call options, shared result envelopes, and testify. No new external dependency.

**Storage**: In-memory frozen candidates and outcomes; no persistent execution session or database.

**Testing**: Package tests with generated-client mocks and HTTP test servers; genuine terminal stdin through `testx/cmd_terminal_runner.go`; race-enabled suite for the implemented concurrency changes; separately controlled live hierarchy validation.

**Target Platform**: Existing supported CLI platforms; Camunda 8.8, 8.9, and 8.10. Preserve 8.7's existing unsupported job-operation result.

**Project Type**: Go CLI and public facade library.

**Performance Goals**: Account for at least 1,000 matching jobs across pages in one invocation. Memory O(selected jobs + one page); execution concurrency bounded by existing effective worker controls. No invented throughput or latency guarantee.

**Constraints**: Zero writes before complete preparation; tenant-safe discovery; no mutation in dry-run; deterministic results despite concurrency; preserve partial outcomes; no unbounded reconciliation; no generated-client edits required.

**Scale/Scope**: One command's completion mode, shared job-search tenant/page normalization, thin facade additions, internal coordinator, three supported adapters plus unsupported-version wiring, tests and command documentation.

## Constitution Check

| Principle | Before research | After design |
|---|---|---|
| Operational proof | Require separate submission and observation facts. | PASS: accepted responses explicitly skip independent confirmation; ambiguous submissions receive bounded observation; missing jobs do not prove completion. |
| CLI-first, script-safe | Preserve single-key and unrelated modes. | PASS: collection forms have one aggregate envelope; prompts use configured stderr; empty and partial results have explicit contracts. |
| Proportionate validation | Planning changes require lightweight checks only. | PASS: this run uses artifact checks; implementation starts with targeted tests, then race suite because shared APIs and concurrent execution change. Live proof remains a distinct release check. |
| Documentation matches behavior | Update command metadata and README with implementation. | PASS: help, capabilities, README, and `make docs-content` are part of the planned delivery. |
| Small, compatible changes | Reuse native layers and helpers on v4.3.4. | PASS: one coordinator, existing pools/options/envelope, no new dependencies or unrelated develop changes. |

No unresolved design clarification or constitution exception remains. The next release number and publication are separate release decisions, not implementation prerequisites.

## Project Structure

### Documentation

```text
specs/345-bulk-job-completion/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/requirements.md
└── contracts/
    ├── cli.md
    └── service.md
```

`tasks.md` is generated subsequently by `speckit-tasks`.

### Source Code

```text
cmd/
├── update_job.go                     # Flags, mode routing, existing ordinary path
├── update_job_request.go             # Legacy parsing and new grammar dispatch
├── update_job_batch.go               # New collection input and command lifecycle
├── get_job*.go                       # Reuse selector validation
├── cmd_views_job.go                  # Preview and aggregate/partial result views
└── *_test.go                         # Request, HTTP, envelope, exit and terminal tests
c8volt/job/
├── api.go / model.go                 # Additive public batch contracts
└── completion.go                    # Thin conversions and service delegation
internal/domain/
└── job*.go                          # Selection, prepared batch, submission and report
internal/services/job/
├── api.go                           # Prepare/execute contract
├── completion/                      # Shared coordinator; domain-only transport ports
├── v87/                             # Explicit unsupported implementations
├── v88/ / v89/ / v810/              # Tenant/page normalization and one-shot submission
└── existing factory and tests       # Keep version routing intact
README.md
docs/cli/                            # Regenerate from command metadata
```

**Structure decision**: The coordinator is a sibling of version packages and does not import their parent factory package. Version services delegate preparation/execution to it, injecting their existing reads and a private one-shot completion function. This prevents import cycles and keeps version mechanics in adapters. All interface implementations and test doubles change together.

## Design

### Selection and preparation

Parse completion input mode before deduplication. Exactly one scalar explicit key retains the legacy path. Repeated/comma-separated key forms, stdin, and filters use the batch path even when one unique key remains. Reject collection forms for every unrelated update mode.

Normalize predicates using existing job-search semantics. `--retries` is a pointer-valued predicate only for selector completion; zero is meaningful. Explicit keyed completion continues rejecting `--retries`. Tenant and pagination controls alone do not select jobs.

Fix tenant equality in versioned search filter builders using `common.EffectiveTenant`; generated clients already support it. Preserve key-only `GetJob` without discovery tenant filtering. Bulk selection additionally checks tenant and predicates locally.

The coordinator consumes normalized pages and owns traversal. Request stable ascending job-key ordering for bulk discovery. Advance using raw page metadata, never accepted/deduplicated counts. Continue sparse pages when metadata supports continuation. A contradictory or indeterminate end cannot become successful empty discovery: bounded read attempts must either establish exhaustion or return an incomplete preparation error. A positive unique-target limit provides a separate successful completion reason. Preserve existing overfull-response trimming.

Freeze first-seen target order after deduplication. Explicit keys retain first-input order. Read all explicit jobs before execution; any invalid or unreadable explicit target fails the entire preparation. A complete empty stdin/search selection succeeds without confirmation. Validate supported API capability before returning even an empty result.

Selection eligibility means matching requested scope, not guaranteed backend completability. Do not discard difficult matches to fill a limit with different jobs. Payload compatibility is checked across the prepared set: nonempty variables with any task listener fail before writes.

### Execution and outcomes

Use the same coordinator for explicit collections and search collections. Recheck each selected key immediately before submission. Already observed `COMPLETED` is a satisfied no-op even if a state predicate has changed; no submission occurs. Otherwise, changed scope, other terminal state, missing/unreadable observations, and unknown states have explicit unsatisfied reasons.

Known nonterminal search states are potentially completable; the backend decides actual acceptance. Do not infer a positive-retry prerequisite or infer an abandoned worker from `CREATED`. Recheck is not a lock. Preserve restrictions on listener variables at recheck as well.

Bulk makes one intentional completion submission per selected key; no automatic replay after timeout, transport error, 404, or 409. Adapters expose acceptance/rejection/uncertainty without hiding attempt evidence. Existing single-key transport behavior stays unchanged. Share request construction and response classification where practical without changing its default retry policy.

Successful submission reports accepted with independent confirmation skipped. For 404/409 or uncertain submission, reconcile with key-only reads within `cfg.App.Backoff.Timeout` and the caller's earlier deadline. Use existing backoff timing. `--no-wait` permits one reconciliation read but no polling; lack of evidence remains unresolved. Caller cancellation stops reads and preserves uncertainty. Never broaden this into process polling or retries/incident repair.

Preinitialize a report row for every frozen key. Reuse `toolx/pool.ExecuteSlice` and `DetermineNoOfWorkers`, applying call options at the owning workflow layer. Track started rows and overlay results by original index; zero-value pool slots are not successes. Default execution continues independent work. Fail-fast stops new scheduling when an unsatisfied outcome is known, while in-flight work retains its own result. Explicitly retain caller cancellation even if the pool returns nil error.

### Rendering and errors

Return report and error together. Batch views render one shared envelope including partial results, then use the existing exit-after-render path to prevent a second error envelope. Keep single-key payload unchanged. Envelope success does not imply observed process termination; accepted versus confirmed counts remain visible.

Use compact aggregate-first human output. Dry-run exposes context rows through verbose preview and all structured preview candidates, with count/filter/tenant/limit and listener consequence visible in ordinary preview. No per-key lifecycle chatter by default. Prompts use `cmd.ErrOrStderr()` and existing eligibility rules. JSON/automation mutation restrictions remain consistent with existing commands.

Empty normal/dry-run results go through the same mode-aware renderer. JSON remains visible with quiet; no confirmation, mutation or accepted/pending claim is emitted for empty scope. `update job` keeps its existing supported output modes; this feature adds no keys-only mode.

## Validation and Delivery

| Coverage | Requirements |
|---|---|
| Input forms, zero-valued retries, legacy mode and capability metadata | FR-001, FR-002, FR-012, FR-014 |
| Version-specific tenant JSON, key-only reads, sparse pages, deduplication, limits, preparation failure | FR-003–FR-005, FR-016 |
| Preview, empty/quiet/JSON/no-wait/automation, real terminal prompts | FR-006, FR-007, FR-012, FR-013 |
| Recheck races, one-shot transport, uncertainty, cancellation, fail-fast, report conservation | FR-008–FR-011, FR-015 |
| README/help generation and representative hierarchy recovery | FR-017, SC-006 |
| Deterministic 1,000-job multipage HTTP scenario and bounded concurrency | SC-001–SC-005, SC-007 |

Follow [quickstart.md](quickstart.md) for implementation validation. Keep interface additions, adapters and test doubles compiling within each work unit; implement one shared collection engine rather than rebuilding a single-item engine later. Group fixtures, tests, formatting and targeted checks with the behavior they verify. Do not make routine context reading or branch checks standalone implementation tasks.

Deliver against `main` from the pinned v4.3.4 baseline. Do not move the release tag, merge develop, or pull in #341. The baseline PR workflow excludes `main` targets, so verify feature-branch push CI and record local validation when preparing that PR; do not silently claim PR-triggered CI ran. Release numbering, forward integration and publication remain separate.

## Complexity Tracking

No constitution violation. The shared coordinator is justified by three adapters needing identical paging, frozen execution and reconciliation; duplicating those workflows or placing them in a facade would violate repository boundaries. No persistent run store or general recovery framework is needed.
