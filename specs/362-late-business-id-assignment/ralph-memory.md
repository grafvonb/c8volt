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

- T001 complete: optional shared validator permits nil for future omitted job intent; the direct adapter requires nonnil. Creation still permits whitespace-only IDs.
- Planning validates all keys/value before reads, reads only exact keys for preview facts, and retains Ready=false with incomplete evidence on failures. Empty explicit plans are Ready=true. Execution validates readiness and one-to-one ordered membership, snapshots values and scope, then uses existing worker limits.
- Plans are invocation-local public data, not authorization tokens. Inconsistent plans fail before writes; structurally valid caller-created plans still receive adapter capability/value/timing checks.
- 8.8/8.9 reject planning, prepared execution and direct mutation before I/O. v810 accepts only the native 204 empty acknowledgement, reuses common completion submission classification, and never invokes RetryCamundaMutation.
- Confirmation runs only after known acceptance with configured backoff and exact-key/value checks. Unknown/rejected submission performs no reconciliation reads; accepted observation failures retain acceptance plus an error.
- README documents library behavior/source compatibility. CLI docs are unchanged because T002 owns CLI delivery.

## Gotchas

- Generated assignment response parsing returns nil plus a parse error for malformed advertised JSON, even if an HTTP status was received. Preserve this as unknown, not authoritative rejection.
- Test fixtures with JSON content type must supply valid problem JSON when testing typed HTTP failures; empty 503 JSON tests parse failure instead of ErrUnavailable.
- Existing process service/public interface doubles in process, resource and cmd need alignment when adding methods. Unrelated fixtures embed the expanded interface; assignment fixtures explicitly implement its new operations.
- The shared HTTP read retry transport only retries GET/HEAD. Assignment POST bypasses the service mutation retry helper. Error strings remain in machine results; no new human logger/output boundary exists in T001.
- Real-engine eligibility, durability, uniqueness configuration and timing were not tested; all dependencies were isolated fakes.

## Reusable Commands

- env GOFLAGS= go test ./internal/domain -run TestValidateBusinessID -count=1
- env GOFLAGS= go test ./internal/services/processinstance/... ./c8volt/process -run BusinessIDAssignment -race -count=1
- env GOFLAGS= go test ./internal/domain ./internal/services/processinstance/... ./c8volt/process ./c8volt/resource -count=1
- env GOFLAGS= go test ./... -run '^$' -count=1
- gofmt on touched Go files; git diff --check.
- All commands above passed in iteration 1. The race selection exercises nine new service/facade test functions plus their subtests; domain selection includes creation regression and assignment validation.
- Full make test remains the T004 final delivery obligation, not performed in T001.

## Do Not Repeat

- Do not replay assignment, redirect a child, infer acceptance from later state, or backfill observed job/task/listener identity.
- Do not log raw result.Error strings at normal or verbose human levels in T002. Preserve those fields for machine evidence and render bounded aggregate diagnostics through configured command logging.
- Do not run integration targets/tags or probe live services for implementation validation.

## Current Handoff

- Next dependency-ready work is T002 [US1]: implement update pi --set-bid command lifecycle/views, output/error/empty/prompt acceptance matrix, source help/metadata and regenerated docs. Recheck tasks.md before selection. T001 is complete; T003/T004 remain open and belong to US2.
