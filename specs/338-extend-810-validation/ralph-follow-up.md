# Separate Ralph follow-up

Suggested issue title: `fix(ralph): stop repeated blocked iterations without task progress`

The #338 run completed nine implementation tasks in its first nine iterations,
then repeated the same external acceptance blocker through iteration 100.
Successful agent process exits reset the failure counter even when no task
completed. Progress-log edits do not resolve the blocker.

Add explicit blocked-result handling and a bounded no-progress counter to
`.specify/extensions/ralph/scripts/bash/ralph-loop.sh`. Stop with a distinct
blocked/no-progress summary; leave the task unchecked and retain its prerequisite.
Do not interpret process exit zero as task completion or automatically authorize
live execution. Preserve successful iterations that legitimately implement work
without closing a task, but bound repeated identical blockers.

Acceptance:
- An explicit external blocker stops without another identical iteration.
- Repeated exit-zero iterations with no task or implementation progress reach
  a small configurable limit; progress/memory-only edits do not reset it.
- Real implementation progress resets the no-progress counter.
- Existing nonzero-exit circuit breaker and strict completion gate still work.
- Shell tests use fake agent results; no model calls or live Camunda runs.

This is a separate runner defect. No Ralph source changes or external issue
creation were performed as part of the #338 acceptance work.
