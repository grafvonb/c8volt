# Ralph Memory

Feature: 362-late-business-id-assignment
Started: 2026-11-02T14:21:10Z

## Codebase Patterns

- Process assignment uses the existing process service dependency in every facade constructor; aggregate c8volt/contract.go embeds process.API, so no extra constructor or aggregate method wiring is needed.
- Service coordinator: internal/services/processinstance/business_id_assignment.go. Public thin API and copied converters: c8volt/process/business_id_assignment.go and convert.go. Native adapter and exact-key waiter own submission/observation.
- NormalizeProcessInstanceKeys requires 16 decimal digits after trimming, deduplicating in first-input order. Keys in the planning examples are descriptive, not valid execution fixtures.
- Existing pool helpers return zero slots for unscheduled work; the coordinator prepopulates not_submitted/skipped results and preserves those slots, joining caller cancellation explicitly.
- Assignment result Status is not_submitted/accepted/confirmed/rejected/unknown; Confirmation is skipped/confirmed/unconfirmed. Requested, prior and later BID pointers are independent. Error is machine evidence only.

## Decisions

- T002 complete: --set-bid dispatches before variable parsing into cmd/update_processinstance_assignment.go; cmd/cmd_views_processinstance_assignment.go owns final formatting. No service/facade changes were needed.
- Assignment reads complete cmd.InOrStdin() input with a bounded scanner. Existing readKeysIfDash rejects empty/terminal stdin and is unsuitable here. Deliberate empty assignment stdin is a ready plan, with items/requestedKeys as empty arrays; no-key without '-' is invalid.
- Assignment approval consumes the service-copied plan, never current flag storage. Frozen-approval regression mutates key/value flags during confirmation and verifies the original wire request. Explicit administration options preserve backend authorization and worker/fail-fast controls.
- JSON previews/empty results succeed even with no-wait; only actual accepted submissions yield an accepted envelope. Failed execution renders partial evidence once, then exits with the original class. Keys-only prints preview targets in dry-run and accepted/confirmed targets after execution.
- Human preview/result lines use renderHumanLine; aggregate ERROR uses the configured logger and bounded counts/class. Raw errors are only machine evidence; HTTP diagnostics stay in the existing redacted DEBUG path. Normal no-wait results retain accepted/skipped evidence.


- T001 complete: optional shared validator permits nil for future omitted job intent; the direct adapter requires nonnil. Creation still permits whitespace-only IDs.
- Planning validates all keys/value before reads, reads only exact keys for preview facts, and retains Ready=false with incomplete evidence on failures. Empty explicit plans are Ready=true. Execution validates readiness and one-to-one ordered membership, snapshots values and scope, then uses existing worker limits.
- Plans are invocation-local public data, not authorization tokens. Inconsistent plans fail before writes; structurally valid caller-created plans still receive adapter capability/value/timing checks.
- 8.8/8.9 reject planning, prepared execution and direct mutation before I/O. v810 accepts only the native 204 empty acknowledgement, reuses common completion submission classification, and never invokes RetryCamundaMutation.
- Confirmation runs only after known acceptance with configured backoff and exact-key/value checks. Unknown/rejected submission performs no reconciliation reads; accepted observation failures retain acceptance plus an error.
- README documents library behavior/source compatibility. CLI docs are unchanged because T002 owns CLI delivery.

## Gotchas

- Assignment fake lost responses classify as internal (exit 1), while HTTP 400 remains invalid_input (exit 2). Unknown assignment evidence is separate from the error class.
- New assignment command tests live in update_processinstance_assignment{,_logging}_test.go, with owned HTTP servers, separate configured/inherited writers and actual terminal stdin. The helper clears inherited C8VOLT overrides before executing the explicit temporary config.
- CLI docs regeneration also synced the T001 library README additions into docs/index.md. README now describes CLI assignment as delivered; atomic completion remains pending.


- Generated assignment response parsing returns nil plus a parse error for malformed advertised JSON, even if an HTTP status was received. Preserve this as unknown, not authoritative rejection.
- Test fixtures with JSON content type must supply valid problem JSON when testing typed HTTP failures; empty 503 JSON tests parse failure instead of ErrUnavailable.
- Existing process service/public interface doubles in process, resource and cmd need alignment when adding methods. Unrelated fixtures embed the expanded interface; assignment fixtures explicitly implement its new operations.
- The shared HTTP read retry transport only retries GET/HEAD. Assignment POST bypasses the service mutation retry helper. Error strings remain in machine results; no new human logger/output boundary exists in T001.
- Real-engine eligibility, durability, uniqueness configuration and timing were not tested; all dependencies were isolated fakes.

## Reusable Commands

- env GOFLAGS= go test ./cmd -run 'Test.*(BusinessIDAssignment|UpdateProcessInstance)' -count=1
- env GOFLAGS= GORACE=atexit_sleep_ms=0 go test ./cmd -run 'Test.*(BusinessIDAssignment|UpdateProcessInstance)' -race -count=1
- env GOFLAGS= GORACE=atexit_sleep_ms=0 go test ./cmd -race -count=1 (passed iteration 2 in 565.222s; allow sufficient wall time).
- env GOFLAGS= make docs-content; git diff --check (passed iteration 2).
- Negative mutation checks temporarily replaced assignment ERROR logging with a direct configured-stderr print and raw err.Error() logging. TestBusinessIDAssignmentLogging/plain-time/configured/failure rejected both. Restored production code passed targeted race validation.


- env GOFLAGS= go test ./internal/domain -run TestValidateBusinessID -count=1
- env GOFLAGS= go test ./internal/services/processinstance/... ./c8volt/process -run BusinessIDAssignment -race -count=1
- env GOFLAGS= go test ./internal/domain ./internal/services/processinstance/... ./c8volt/process ./c8volt/resource -count=1
- env GOFLAGS= go test ./... -run '^$' -count=1
- gofmt on touched Go files; git diff --check.
- All commands above passed in iteration 1. The race selection exercises nine new service/facade test functions plus their subtests; domain selection includes creation regression and assignment validation.
- Full make test remains the T004 final delivery obligation, not performed in T001.

## Do Not Repeat

- Do not replay assignment, redirect a child, infer acceptance from later state, or backfill observed job/task/listener identity.
- Do not log raw result.Error strings at normal or verbose human levels in assignment or completion. Preserve those fields for machine evidence and render bounded aggregate diagnostics through configured command logging.
- Do not run integration targets/tags or probe live services for implementation validation.

## Current Handoff

- Next dependency-ready work is T003 [US2]: atomic BID-bearing scalar/bulk completion service, public models/conversions, copied approved payloads, adapter one-shot submission and evidence-aware reconciliation. Recheck tasks.md before selection. T001/T002 complete US1; T003/T004 remain open.
- Reuse the assignment validator and preserve plain completion behavior. T004 owns completion CLI delivery and the final repository-wide make test gate. No live-engine validation was run or is needed for implementation completion.
