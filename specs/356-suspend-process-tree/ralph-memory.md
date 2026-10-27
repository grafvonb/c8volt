# Ralph Memory

Feature: 356-suspend-process-tree
Started: 2026-10-26T18:30:05Z

## Codebase Patterns

- `process.API.PlanProcessInstanceSuspension` delegates to `internal/services/processinstance.PlanProcessInstanceSuspension`; the internal/versioned API is unchanged. Domain/public plans own RequestedKeys, Instances, and TargetKeys slices; conversion uses slices.Clone and toolx.MapSlice, preserving zero plans on failure and non-nil successful arrays.
- Planning guards capability, normalizes all seeds, performs one first-seed forced local dry-run preflight, then calls DescendantsResult in input order with copied explicit-key/suppression options. Complete overlapping seeds are skipped; first observation wins. Only ACTIVE is eligible, SUSPENDED is separate, other states are excluded, and unrecognized states normalize to UNKNOWN.
- Every traversal must be complete descendants of its requested seed, include that seed, and have valid canonical keys with matching Chain identities. Partial/unresolved results use ErrPrecondition; malformed discovery uses ErrMalformedResponse. Any error/cancellation returns a zero plan; no executable partial scope escapes.
- Existing SuspendProcessInstances remains the only collection executor. Public facade HTTP tests compose planning with copied TargetKeys, proving no descendant rediscovery, no ineligible mutations, retained confirmation/no-wait evidence, and no forced preview leakage.

- Tree CLI orchestration lives in cmd/suspend_processinstance_tree.go: plan once, check error before no-op, clone targets before approval, and call the unchanged collection executor once. Preview/no-op never calls the mutation facade or starts execution activity. The tree view owns typed items+plan JSON and allowlisted verbose scope facts; exact-key payloads omit plan.
- Tree activity uses the existing local wrapper only after approval and closes before final output, including unwinding. Keys reuse the existing renderer. Human scope/summary/detail lines use renderHumanLine; operation failures reuse the compact grouped formatter and exitAfterRenderedResult. Planning failures use boundedCommandFailureReason while JSON retains original evidence.
- The tree HTTP test fixture wraps, rather than relaxes, the exact-key fixture. Discovery has separate request evidence; mutation/confirmation matrices run in both modes. Invalid-input/capability tests deliberately use the strict exact-key fake even with --tree so any forbidden discovery fails. PTY callbacks mutate hierarchy/state under the fixture mutex before approval to prove no rediscovery.

## Decisions

- Iteration 1 completed only T001/US1. T002 owns the CLI flag, mode runner/views, terminal/logger/activity acceptance, README and generated docs. No production command behavior or output changed in T001.
- Keep the existing 1,000-direct-children lookup limit and explicit-key IgnoreTenant semantics. No walker/adapter/generated-client/pool modifications were needed.
- New public classification constants are ProcessInstanceSuspensionPlanEligible, ProcessInstanceSuspensionPlanAlreadySuspended, and ProcessInstanceSuspensionPlanExcluded. The plan/error must be checked before treating empty TargetKeys as a no-op.
- Custom process.API implementations now require the planner method. Repository concrete command/resource stubs explicitly reject unsupported planning; new planner service fakes provide explicit descendant callbacks.

- Iteration 2 completed T002/US2, including shared US3/US4 acceptance, README, source metadata/help and regenerated docs. All feature tasks are complete. No shared pool, service, facade, traversal, generated-client, or global logger/activity behavior changed in this iteration.

## Gotchas

- v810 parentProcessInstanceKey search filters serialize as JSON strings, not {$eq: ...} objects. JSON-typed fake error responses need a valid JSON body; empty bodies cause generated JSON decoding errors before HTTP classification.
- ferrors.FromDomain maps domain failures to public sentinels and retains cause text, not the original domain sentinel chain. context.Canceled stays wrapped until CLI local normalization; context.DeadlineExceeded maps to ErrTimeout. Do not alter shared error conversion to satisfy incorrect fixture assumptions.
- Waiting timing validation must happen before discovery even for all-ineligible scopes. NoWait bypasses irrelevant timing validation; fake-server tests use negative delay to prove both paths.
- Reusing the existing API means descendant metadata includes StartKey and RootKey equal to the selected seed. A missing observation is malformed, not an UNKNOWN state.

- DEBUG discovery includes POST /process-instances/search. Count suspension diagnostics using the /suspension suffix, not all process-instance POSTs. CLI normalization maps context.Canceled to local_precondition; preserve this established exit behavior.
- The real PTY suite worked on this macOS host with redirected stdout, configured/inherited stderr, and no skips. Subprocess fixtures inherit the sanitized parent environment and supply only test-owned configuration/endpoints.

## Reusable Commands

- Run tests with a minimal process environment: `env -i PATH="$PATH" HOME="$HOME" TMPDIR="${TMPDIR:-/tmp}" GOENV=off GOFLAGS= go test ...`. This removes inherited application profiles, URLs, tokens, helper-process flags and build tags; matching fixtures construct their own clients/configuration. Never enable integration tags.
- Passed: `go test ./internal/services/processinstance/... -run 'ProcessInstanceSuspension|Descendants' -count=1` using the minimal environment above. Covers planner and existing collection/worker/adapter/waiter/descendant controls.
- Passed: `go test ./c8volt/process -run 'ProcessInstanceSuspension' -count=1` using the same environment. Covers mapping, factory guards, explicit tenant handling, retained lookup size, no-I/O planning preflight, and plan-to-execution composition.
- Passed: `go test ./... -run '^$'` using the same environment, for all public-interface consumers. This was compile-only, not a live or integration execution.
- Passed: `GORACE=atexit_sleep_ms=0 go test ./c8volt/process ./internal/services/processinstance -run '^TestProcessInstanceSuspensionPlan' -race -count=1` using the same environment, for new fake-server/collection fixtures and planner ownership.
- Targeted gofmt and git diff --check passed. No live environment was used; real-engine behavior remains unverified.

- Iteration 2 passed: minimal-environment go test ./cmd -run '^(TestSuspendProcessInstance|TestAllCommandHelpFocusesOnActions)' -count=1, followed by the same selection with -race and GORACE=atexit_sleep_ms=0. Both reported 789 successful test/subtest entries, no failures or skips. The race run also covers final strengthened zero-discovery and frozen partial-plan assertions.
- Iteration 2 passed: minimal-environment make docs-content and go test ./docsgen -count=1. Targeted gofmt, declaration ownership/output-boundary review and git diff --check passed. No service/facade rerun or repository-wide suite was needed for this command-only increment.
- Test JSON logs for this iteration are outside the repository at /tmp/c8volt-356-iteration2-tests.jsonl and /tmp/c8volt-356-iteration2-race.jsonl; durable evidence is recorded here and in progress.md.

## Do Not Repeat

- Do not move discovery/classification into the CLI or facade, widen internal versioned interfaces, mutate during planning, or let planning observations become submission/confirmation evidence.
- Keep --tree opt-in and preserve the exact-key schema/zero-discovery controls. Update source help and regenerate CLI docs; do not hand-edit them.
- Do not run live integration/smoke validation or discover a live target. Existing service/facade suites already passed; do not rerun them solely for records or commits.

## Current Handoff

- Feature complete; no handoff required.
