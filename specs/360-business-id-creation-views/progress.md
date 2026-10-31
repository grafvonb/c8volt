# Ralph Progress Log

Feature: 360-business-id-creation-views
Started: 2026-10-31 12:43:38

---


---
## Iteration 1 - 2026-10-31 12:50
**Work Unit**: US1 — T001 public business-ID creation complete; story partial (CLI T002 remains)
**Tasks Completed**:
- [x] T001: Deliver business-ID-aware public single, sequential and counted creation, native mappings, validation and successful partial results.
**Tasks Remaining in Work Unit**: T001 complete; 1 task remains in US1 (T002). Five feature tasks remain overall.
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- c8volt/process/bulk.go
- c8volt/process/business_id_test.go
- c8volt/process/client.go
- c8volt/process/convert.go
- c8volt/process/convert_test.go
- c8volt/process/model.go
- docs/index.md
- internal/domain/business_id.go
- internal/domain/business_id_test.go
- internal/domain/processinstance.go
- internal/services/processinstance/bulk.go
- internal/services/processinstance/bulk_test.go
- internal/services/processinstance/business_id_test.go
- internal/services/processinstance/v810/business_id_test.go
- internal/services/processinstance/v810/convert.go
- internal/services/processinstance/v810/convert_test.go
- internal/services/processinstance/v810/service.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v88/service_test.go
- internal/services/processinstance/v89/service.go
- internal/services/processinstance/v89/service_test.go
- specs/360-business-id-creation-views/progress.md
- specs/360-business-id-creation-views/ralph-memory.md
- specs/360-business-id-creation-views/tasks.md
**Learnings**:
- Initial regression tests failed as expected: sequential errors lost prior successes; counted errors exposed zero-value slots. The validator test first failed because its implementation was absent. All final checks passed.
- Confirmed nonzero BusinessID inventory in every changed area. Targeted domain/service/public creation checks passed; the same service/public selection passed under the race detector. A final public BusinessID race rerun passed after test assertion cleanup.
- Full affected-package tests passed for internal/domain, internal/services/processinstance/... and c8volt/process; sanitized `go build ./...` passed. All dependencies were strict mocks or test-owned HTTP servers; no live engine or integration test was run. Full repository race validation remains T006's gate.
- HTTP tests assert generated request bodies for both selectors, omission/exact values, Unicode boundaries, zero mutation on invalid/unsupported input, response-versus-observation identity, failed waiting, and actual conflict preservation through public calls. Bulk tests cover snapshots, bounded out-of-order completion, stable partial order, fail-fast and cancellation without fabricated entries.
- Error-path review: adapters return original domain/transport errors; collection services retain successful evidence; facades convert both results and errors. No command files, direct diagnostic prints or error-format logging were added. T002 owns remaining command rendering/logger/exit corrections and their complete execution matrix.
- Targeted gofmt and `git diff --check` passed. `make docs-content` passed; generated differences are README-derived public API guidance and build metadata in docs/index.md, with no CLI reference changes.
---
