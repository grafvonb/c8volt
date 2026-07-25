# C89 Real-State Integration Coverage Matrix

This feature-local artifact is planning and evidence control for `257-c89-real-state-integration`. It is not generated CLI documentation and should not be copied into `docs/`.

| Topic | Current Evidence Level | Target Real-State Proof | First Follow-Up |
| --- | --- | --- | --- |
| Aggregate proposal wiring | Partial live evidence, repair gaps only in slice reports | Aggregate command and embedded BPMN proposals include every known family gap | Wire ops repair proposals into aggregate reports |
| Real `get job` rows | Mostly flag/help/unit and proposal-backed behavior | Non-empty suite-owned Camunda 8.9 job rows returned by `get job` | Add a minimal active-job fixture slice |
| Job retries, timeout, fail, no-wait | Partial command execution and output checks | Observable job post-state after update paths, or explicit no-wait/submitted evidence | Extend active-job fixture with one mutation at a time |
| `update job --throw-bpmn-error` | Proposal-backed | BPMN error-capable job drives an observable error path | Propose or add embedded BPMN error fixture |
| Incidents with related jobs | Active incident evidence exists, related-job evidence is shallow | Incident queries and repair paths show related job keys where behavior depends on them | Add incident-with-job fixture and post-query evidence |
| Listener jobs and `--with-listeners` | Flag/dependency coverage, non-empty listener evidence missing | Listener-capable model produces listener jobs or listener element evidence | Propose or add listener embedded BPMN fixture |
| Deterministic retention candidates | No-match or very high retention windows | Completed suite-owned process instances become real retention candidates | Add conservative completed-instance fixture and retention assertions |
| Real purge semantics | Preview/no-match and report checks | Confirmed purge/delete removes or reports suite-owned incidents, orphan state, and cleanup failures | Split purge targets by candidate type |
| Process-definition delete semantics | Partial command behavior | Suite-owned process definitions are deleted or reported as blocked with dependents | Add suite-owned deployment lifecycle assertions |
| Cancel/delete/resolve post-state | Baseline and volume coverage | Confirmed destructive actions verify post-state for real candidates | Add mixed target sets by family |
| Partial failure and fail-fast | Limited mixed-input coverage | Mixed valid, missing, malformed, stale, and already-mutated targets prove stop/continue reporting | Add one family-neutral mixed-target helper |
| Ops report parity | Volume report coverage exists | Real-state scenarios prove report/stdout parity on non-empty candidates | Reuse volume report assertions with real candidates |
| Version extensibility | Affected versions recorded broadly | Current 8.9 foundation is explicit and future minors can add targeted rows | Keep scenario names and proposal versions explicit |

## Evidence Rules

- A topic is `live-covered` only when the suite observes real Camunda state before and after the relevant command.
- A topic is `partially live-covered` when real state exists but not for every critical flag or mutation path.
- A topic is `no-match only` when the current test proves filtering, parsing, or report shape but not non-empty candidates.
- A topic is `proposal-backed` when missing c8volt commands or embedded BPMN models block reliable setup.
- A topic is `not yet started` when no feature-local test or proposal exists.
