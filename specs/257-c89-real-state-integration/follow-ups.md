# C89 Real-State Integration Follow-Up Roadmap

This artifact consolidates the remaining integration-test follow-ups discovered across the 255 all-command suite, the 256 volume suite, and the 257 real-state suite. It is the authoritative planning source for future issue/spec creation after 257. Do not add new runtime-generated backlog files from tests.

## Design Rules

- Keep 255 and 256 artifacts as historical context; do not rewrite them as the active roadmap.
- Put every remaining command setup, embedded BPMN, product output, ops-report, and pipeline follow-up here or in `gaps.md`.
- Prefer one focused issue/spec per coherent setup or fixture capability.
- Future integration tests should record runtime truth only: live-covered, partially live-covered, dry-run-covered, skipped-prerequisite, no-match only, or failed.
- Close a follow-up only after a real Camunda 8.9 test proves the behavior or the gap is explicitly declared out of scope.

## Follow-Up Issue Candidates

| Group | Follow-Up Candidate | Source Context | Blocks | Suggested First Spec |
| --- | --- | --- | --- | --- |
| Embedded BPMN assets | Add a catchable BPMN error service-task fixture with `ErrorTimerCode` path | 255 update gap, 256 volume gap, 257 BPMN error gap | Confirmed `update job --throw-bpmn-error` process-state proof | Embedded BPMN fixture pack |
| Embedded BPMN assets | Add richer listener fixtures, including task-listener or listener timeline variants beyond execution-listener jobs | 255 walk and ops analyse gaps, 257 listener variants gap | Broader `--with-listeners` coverage for `walk`, `get element`, and `ops analyse` | Embedded BPMN fixture pack |
| Embedded BPMN assets | Add deterministic slow-duration process and element fixtures | 255 ops analyse gap, 256 ops analyse partial task | Non-empty duration threshold proof without timing flakiness | Embedded BPMN fixture pack |
| Embedded BPMN assets | Add representative variable-shape fixture with nested object, array, boolean, numeric, string, and null variables | 255 update gap, 256 update notes | Volume variable rendering, filtering, and mutation proof | Embedded BPMN fixture pack |
| Embedded BPMN assets | Add deterministic repairable incident/job fixture with controlled partial-failure branches | 256 ops repair gap, 257 repair partial failures gap | `ops repair` mixed related-job, stale incident, fail-fast, and report-row proof | Repair and resolve real-state depth |
| Embedded BPMN assets | Add durable standalone resolve fixture whose incident can be cleared without immediate recreation | 257 destructive gap | Confirmed `resolve incident` or `resolve process-instance` post-state with no active incident | Repair and resolve real-state depth |
| c8volt setup commands | Add setup support for activated jobs accepted by Camunda timeout mutation | 255 update gap, 257 job timeout mutation gap | Confirmed `update job --timeout` post-state proof | Job and incident setup commands |
| c8volt setup commands | Add setup support for suite-owned active incidents with controllable related jobs | 255 ops execute and repair gaps, 257 incident/job gaps | Smoke-test incident/job-state branches and repair edge cases | Job and incident setup commands |
| c8volt setup commands | Add setup support for aged completed process instances and process-definition metadata filters | 255 retention gap, 256 ops execute partial task, 257 retention gap | Aged retention deletion, no-wait retention, `--pd-version`, and `--pd-version-tag` proof | Retention and purge candidate setup |
| c8volt setup commands | Add deterministic process-definition delete, all-process-definition purge, and orphan purge candidates | 256 ops purge partial task, 257 purge/delete gap | Confirmed delete/purge blocked, deleted, retained, and absent post-state proof | Retention and purge candidate setup |
| c8volt setup commands | Add setup support for numeric resource IDs, user-task or element-instance targets, and tenant-specific disposable data where supported | 255 follow-up coverage notes | Positive/negative command proof without relying on dirty global data | Auxiliary target setup |
| Product output contract | Add stable key and ok identity fields to state-only `expect process-instance` JSON rows, matching incident expectation rows | 256 expect/resolve volume gap, 257 expect identity gap | Machine-output row correlation for state-only expectations | Expect output identity |
| Ops report semantics | Add deterministic cleanup-failed, retained-resource, overwrite-conflict, preserve-report, notice, and error branches | 256 ops execute/purge partial tasks, 257 report edge gap | Complete report accounting across destructive and ops workflows | Ops report edge semantics |
| Ops report semantics | Add `ops repair` mixed-target report parity for valid, stale, missing, malformed, and already-mutated resources | 256 ops repair partial task, 257 partial failure gap | Fail-fast/partial accounting in stdout and report rows | Repair and resolve real-state depth |
| Pipeline semantics | Complete stdin pipeline volume coverage for empty, duplicate, whitespace-padded, malformed, missing, valid, mixed, dry-run, and confirmed mutation inputs | 256 open US3 tasks | Clean keys-only producer to stdin consumer workflows for people and pipelines | Volume pipeline completion |

## Issue Creation Order

1. Embedded BPMN fixture pack, because it unlocks BPMN error, listener, slow-duration, variable-shape, repair, and resolve depth without changing every command at once.
2. Job and incident setup commands, because they close timeout, incident, smoke-test, and repair setup gaps.
3. Retention and purge candidate setup, because it closes aged retention, process-definition purge/delete, orphan purge, and cleanup edge states.
4. Repair and resolve real-state depth, because it depends on better incident/job fixtures and setup.
5. Volume pipeline completion, because it can reuse the stable destructive and real-state foundations.
6. Expect output identity and auxiliary target setup, because they are narrower product/API consistency follow-ups.

## Migration Notes

- 255 and 256 artifacts remain historical context; treat their historical setup-gap records as source context only.
- New issues should reference this file and `gaps.md`, not the legacy runtime-output pattern.
- When a follow-up is implemented, update `coverage-matrix.md`, `gaps.md`, and this file in the same work unit.
