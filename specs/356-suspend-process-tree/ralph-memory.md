# Ralph Memory

Feature: 356-suspend-process-tree
Started: 2026-10-26T18:30:05Z

## Codebase Patterns

- `process.API.PlanProcessInstanceSuspension` delegates to `internal/services/processinstance.PlanProcessInstanceSuspension`; the internal/versioned API is unchanged. Domain/public plans own RequestedKeys, Instances, and TargetKeys slices; conversion uses slices.Clone and toolx.MapSlice, preserving zero plans on failure and non-nil successful arrays.
- Planning guards capability, normalizes all seeds, performs one first-seed forced local dry-run preflight, then calls DescendantsResult in input order with copied explicit-key/suppression options. Complete overlapping seeds are skipped; first observation wins. Only ACTIVE is eligible, SUSPENDED is separate, other states are excluded, and unrecognized states normalize to UNKNOWN.
- Every traversal must be complete descendants of its requested seed, include that seed, and have valid canonical keys with matching Chain identities. Partial/unresolved results use ErrPrecondition; malformed discovery uses ErrMalformedResponse. Any error/cancellation returns a zero plan; no executable partial scope escapes.
- Existing SuspendProcessInstances remains the only collection executor. Public facade HTTP tests compose planning with copied TargetKeys, proving no descendant rediscovery, no ineligible mutations, retained confirmation/no-wait evidence, and no forced preview leakage.

## Decisions

- Iteration 1 completed only T001/US1. T002 owns the CLI flag, mode runner/views, terminal/logger/activity acceptance, README and generated docs. No production command behavior or output changed in T001.
- Keep the existing 1,000-direct-children lookup limit and explicit-key IgnoreTenant semantics. No walker/adapter/generated-client/pool modifications were needed.
- New public classification constants are ProcessInstanceSuspensionPlanEligible, ProcessInstanceSuspensionPlanAlreadySuspended, and ProcessInstanceSuspensionPlanExcluded. The plan/error must be checked before treating empty TargetKeys as a no-op.
- Custom process.API implementations now require the planner method. Repository concrete command/resource stubs explicitly reject unsupported planning; new planner service fakes provide explicit descendant callbacks.

## Gotchas

- v810 parentProcessInstanceKey search filters serialize as JSON strings, not {$eq: ...} objects. JSON-typed fake error responses need a valid JSON body; empty bodies cause generated JSON decoding errors before HTTP classification.
- ferrors.FromDomain maps domain failures to public sentinels and retains cause text, not the original domain sentinel chain. context.Canceled stays wrapped until CLI local normalization; context.DeadlineExceeded maps to ErrTimeout. Do not alter shared error conversion to satisfy incorrect fixture assumptions.
- Waiting timing validation must happen before discovery even for all-ineligible scopes. NoWait bypasses irrelevant timing validation; fake-server tests use negative delay to prove both paths.
- Reusing the existing API means descendant metadata includes StartKey and RootKey equal to the selected seed. A missing observation is malformed, not an UNKNOWN state.

## Reusable Commands

- Run tests with a minimal process environment: `env -i PATH="$PATH" HOME="$HOME" TMPDIR="${TMPDIR:-/tmp}" GOENV=off GOFLAGS= go test ...`. This removes inherited application profiles, URLs, tokens, helper-process flags and build tags; matching fixtures construct their own clients/configuration. Never enable integration tags.
- Passed: `go test ./internal/services/processinstance/... -run 'ProcessInstanceSuspension|Descendants' -count=1` using the minimal environment above. Covers planner and existing collection/worker/adapter/waiter/descendant controls.
- Passed: `go test ./c8volt/process -run 'ProcessInstanceSuspension' -count=1` using the same environment. Covers mapping, factory guards, explicit tenant handling, retained lookup size, no-I/O planning preflight, and plan-to-execution composition.
- Passed: `go test ./... -run '^$'` using the same environment, for all public-interface consumers. This was compile-only, not a live or integration execution.
- Passed: `GORACE=atexit_sleep_ms=0 go test ./c8volt/process ./internal/services/processinstance -run '^TestProcessInstanceSuspensionPlan' -race -count=1` using the same environment, for new fake-server/collection fixtures and planner ownership.
- Targeted gofmt and git diff --check passed. No live environment was used; real-engine behavior remains unverified.

## Do Not Repeat

- Do not move discovery/classification into the CLI or facade, widen internal versioned interfaces, mutate during planning, or let planning observations become submission/confirmation evidence.
- Do not expose --tree until T002 completes its entire operator contract; do not hand-edit generated docs.
- Do not run live integration/smoke validation or discover a live target. Existing service/facade suites already passed; do not rerun them solely for records or commits.

## Current Handoff

- Next dependency-ready work: T002 [US2], the complete --tree command workflow, focused views, output/PTY/activity regressions, source metadata/help, README and generated docs. T001 is complete; no blocking dependencies remain. Recheck tasks.md before selection and read contracts/cli.md for exact wording and output semantics.
