# C89 Real-State Integration Coverage Matrix

This feature-local artifact is planning and evidence control for `257-c89-real-state-integration`. It is not generated CLI documentation and should not be copied into `docs/`.

| Topic | Current Evidence Level | Target Real-State Proof | First Follow-Up |
| --- | --- | --- | --- |
| Gap artifact validation | not yet started | `gaps.md` includes every known command setup and embedded BPMN fixture gap that blocks deeper live coverage | Wire `integration-cli-real-state-gaps` to validate matrix and gap artifact consistency |
| Real `get job` rows | live-covered for Camunda 8.9 with suite-owned service-task listener jobs | Non-empty suite-owned Camunda 8.9 job rows returned by `get job` | Extend later slices to non-listener BPMN element jobs when available |
| Job retries, timeout, fail, no-wait | partially live-covered: retries confirmed, fail/no-wait submitted, timeout dry-run planned; activated timeout mutation is skipped-prerequisite | Observable job post-state after update paths, or explicit no-wait/submitted evidence | Add c8volt setup support for activated jobs so `update job --timeout` can be confirmed |
| `update job --throw-bpmn-error` | dry-run-covered with unchanged job-state verification on Camunda 8.9; confirmed mutation is skipped-prerequisite | BPMN error-capable job drives an observable error path | Add embedded catchable BPMN error fixture and c8volt setup path for confirmed mutation |
| Incidents with related jobs | live-covered for Camunda 8.9 through suite-owned failed listener jobs and ops repair dry-run related-job counts | Incident queries and repair paths show related job keys where behavior depends on them | Extend later destructive slice to confirmed repair post-state |
| Listener jobs and `--with-listeners` | live-covered for Camunda 8.9 through suite-owned execution-listener jobs in `get element`, `walk process-instance`, and `ops analyse slow-process-instances` | Listener-capable model produces listener jobs or listener element evidence | Extend later slices to task-listener jobs when an embedded fixture exists |
| Deterministic retention candidates | dry-run-covered with non-empty suite-owned completed candidates using `--retention-days 0`; confirmed deletion remains open | Completed suite-owned process instances become real retention candidates | Add confirmed retention deletion and post-state evidence |
| Real purge semantics | Preview/no-match and report checks | Confirmed purge/delete removes or reports suite-owned incidents, orphan state, and cleanup failures | Split purge targets by candidate type |
| Process-definition delete semantics | Partial command behavior | Suite-owned process definitions are deleted or reported as blocked with dependents | Add suite-owned deployment lifecycle assertions |
| Cancel/delete/resolve post-state | partially live-covered: cancel and delete prove dry-run safety plus confirmed post-state on suite-owned active process instances | Confirmed destructive actions verify post-state for real candidates | Extend the destructive target to resolve, purge, repair, and mixed target sets |
| Partial failure and fail-fast | Limited mixed-input coverage | Mixed valid, missing, malformed, stale, and already-mutated targets prove stop/continue reporting | Add one family-neutral mixed-target helper |
| Ops report parity | partially live-covered: retention target writes JSON report evidence for non-empty real candidates and proves dry-run/report agreement | Real-state scenarios prove report/stdout parity on non-empty candidates | Extend report parity to confirmed retention, purge, and repair scenarios |
| Version extensibility | 8.9 foundation explicit in matrix and `gaps.md` | Current 8.9 foundation is explicit and future minors can add targeted rows | Keep scenario names and gap versions explicit |

## Evidence Rules

- A topic is `live-covered` only when the suite observes real Camunda state before and after the relevant command.
- A topic is `partially live-covered` when real state exists but not for every critical flag or mutation path.
- A topic is `dry-run-covered` when the command plan is validated against real keys and follow-up state proves no mutation.
- A topic is `skipped-prerequisite` when missing c8volt commands or embedded BPMN models block reliable setup.
- A topic is `no-match only` when the current test proves filtering, parsing, or report shape but not non-empty candidates.
- A topic is `not yet started` when no feature-local runtime test or gap row exists.
