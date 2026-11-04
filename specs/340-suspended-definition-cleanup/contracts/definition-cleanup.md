# Contract: Definition Statistics, Deletion, and Purge

Applies to existing definition lookup/list/statistics/watch, definition deletion, and `ops purge all-process-definitions`. No new command, flag, public method, or report schema version is introduced.

## Statistics and compatibility

| Observation | Required result |
| --- | --- |
| v810: active=2, suspended=3 | Existing active field stays 2; suspended is 3 and supported; cleanup observes unfinished with a combined query |
| v810: suspended=0 | Support is true even if the zero-valued numeric field is omitted by existing JSON conventions |
| v88/v89 | Suspension is unavailable; fields follow unsupported omission conventions; no unsupported request is sent |
| Required observation fails | Return existing error class; do not substitute zero, stale values, or unsupported status |
| No statistics requested | Preserve ordinary lookup/list output and request behavior |

Human statistics append `su:` only when supported and use the same zero-as-dash formatting as neighboring buckets. Existing statistics watch tracks both value and support, including clearing removed/zero values and detecting suspended-only changes. Watch timing, paging, terminal rendering, change markers, and baseline ownership remain unchanged. No definition lifecycle column or DRAINING display is added.

## Selection and execution

| Mode | Contract |
| --- | --- |
| Normal checks, no force | Refuse at the existing preflight boundary if unfinished count is positive; no mutation after that refusing preflight |
| Force | Discover ACTIVE/SUSPENDED union on 8.10, use current root planner, cancel roots, perform required unfinished verification, delete instance history and definitions through the existing workflow |
| Dry-run | Preview the same applicable unfinished scope and existing dependency expansion, with zero mutation submissions |
| No-wait | Preserve accepted-versus-confirmed reporting; required prerequisites for later destructive steps still execute |
| No-state-check where already supported | Preserve skipped preview and existing bypass/read boundaries; do not create a new bypass flag for purge or an implicit cancellation expansion |
| Empty scope | Existing successful empty payload/wording; no mutation, no mutation confirmation, no accepted-work claim |
| Failed cancellation or required verification | No dependent deletion; preserve existing errors and partial results |
| Additional work remains at verification | Continue existing bounded verification/fail; do not add that work to cancellation automatically |

Preserve existing single-item versus collection preflight boundaries. This feature does not make direct collection calls transactional and does not add cross-invocation deduplication.

## Scope and counting

Keep process-definition candidate selection, latest/version behavior, tenant options, limit semantics, and already-frozen candidate handling. Extend only instance eligibility underneath it. Existing root/family expansion and overlap handling remain in their current owner. Counts describing selected instances and expanded affected families must remain distinct.

Combined queries retain membership during ACTIVE/SUSPENDED transitions. They do not prevent new starts, eliminate exporter lag, or provide a snapshot across pages. Discovery with genuine advancing continuation continues even through an empty page. Unresolved completion/progress errors must not become an exact zero or a complete destructive scope. Do not silently widen tenant or definition criteria.

## Output and reports

- Active fields/keys retain active-only meanings. Add suspended/unfinished fields described in [data-model.md](../data-model.md); update both normal and duplicated ops conversions.
- Human blockers, previews, progress and reports say unfinished where they refer to ACTIVE plus SUSPENDED, with a compact active/suspended breakdown when available. Unsupported data is not rendered as measured zero.
- Resource and purge summaries, Markdown reports, JSON reports, and affected/blocked key projections must agree. Do not introduce per-key lifecycle chatter into default output.
- Preserve JSON envelope cardinality, partial-result payloads, original error/exit classification, keys-only bytes, quiet precedence, and existing null/empty/omission conventions.
- Diagnostics use the configured command-context logger; result payloads use existing stdout renderers; prompts remain plain text on configured stderr. Changed wording is not permission to log raw errors or transport details.
- Preserve accepted nullable deletion responses, existing batch polling where a key is available, and current final deletion confirmation. Do not reinterpret request acceptance as confirmed purge or redesign lifecycle evidence.

## Required isolated contract evidence

Use controlled active-only, suspended-only, mixed, terminal-only, and empty sets. Include both mixed parent/child directions, overlapping roots, state changes during reads, incomplete/capped/sparse paging, read/cancel/drain/history errors, no-state-check/no-wait controls, older-version restrictions, and tenant/limit controls.

Command execution coverage captures stdout and stderr separately. Applicable changed error paths require configured/inherited logger routing and severity, compact repeated-error aggregation, no raw HTTP data at normal or verbose levels, quiet errors, exactly one JSON envelope plus EOF, preserved partial result/exit, and redacted DEBUG via flag and configured level using plain and structured log formats. Assertions must detect logger bypass and raw-error exposure. Reuse existing tests and extend only affected paths.

For changed empty/prompt paths, retain real terminal stdin with separate streams, exact human empty wording, zero-byte keys-only output, one JSON envelope, no confirmation or mutation for empty scopes, and nonempty/abort regressions. Do not substitute a pipe-only test for terminal behavior.

No live integration run is an implementation prerequisite.
