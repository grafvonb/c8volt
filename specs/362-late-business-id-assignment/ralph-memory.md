# Ralph Memory

Feature: 362-late-business-id-assignment
Started: 2026-11-02T14:21:10Z

## Codebase Patterns

- Process assignment coordinator: `internal/services/processinstance/business_id_assignment.go`; thin public API: `c8volt/process/business_id_assignment.go`; native adapter and exact-key waiter own submission/observation. Existing process dependencies and aggregate interface embedding supply constructor/API wiring.
- Assignment command dispatches before variable parsing into `cmd/update_processinstance_assignment.go`; final views live in `cmd/cmd_views_processinstance_assignment.go`. Read full configured stdin with a bounded scanner: the ordinary readKeysIfDash helper rejects terminal/empty stdin and is unsuitable for deliberate empty assignment scope.
- Job completion retains its existing request/outcome/plan/batch files. Scalar `planUpdateJob` freezes BID and nested variables with `UpdatePlan.FreezeCompletionPayload` before approval. `jobAssignmentOutcomeView` preserves scalar evidence on both success and failure; main dispatch exits after rendering instead of invoking another renderer. Batch preparation/abort reports retain requested identity and not-submitted evidence.
- Public `job.ValidateCompletionBusinessID` delegates optional value validation to the shared domain validator without I/O. CLI parsing copies flag storage and rejects assignment outside completion. Scalar old-version validation precedes job lookup; bulk capability validation remains service-owned.
- Human previews/results/verbose rows use renderHumanLine; compact failures use the configured ERROR logger. JSON uses one envelope with partial evidence and original error classification. Prompt text stays plain configured stderr. Empty job scopes preserve established human wording without an extra assignment line.

## Decisions

- T001–T004 complete. Native assignment/atomic completion are one-shot; never replay, redirect to parents, fall back without BID, or infer assignment from a later completed job. Native 204 acknowledges acceptance; independently observed snapshots are never backfilled.
- Shared assignment validation permits nil only for omitted job intent; direct assignment requires nonnil. Reject blank/invalid UTF-8/>256 UTF-16 units; preserve accepted bytes and creation's existing whitespace behavior.
- Direct plans validate all input before reads, freeze stable distinct exact keys/value, and require authoritative readiness plus one-to-one ordered membership before writes. Explicit completed empty plans are valid. Plans are invocation-local data, not authorization tokens. v88/v89 reject assignment before service I/O, including forged/empty prepared requests.
- Direct confirmation runs only after known acceptance, polling exact key/value with existing backoff. Unknown/rejected dispatch never triggers reconciliation reads. Accepted observation failure retains acceptance plus error; no-wait records skipped confirmation.
- Job preparation copies BID, nested variables and selection. Distinct job keys remain separate even in one process. Precompleted jobs cannot satisfy assignment; unknown/rejected dispatch followed by COMPLETED keeps independent observation and original failure. Plain completion retry/reconciliation remains unchanged.
- Scalar approved plans privately freeze key/mode/payload. Conversion rejects altered plan attachments or changed request key/mode, and uses approved payload despite original input edits. Public UpdateJob rejects assignment intent rather than dropping it. Existing listener-variable restrictions remain enforced.
- New CLI assignment supports scalar, repeated/comma keys, newline stdin and existing selector completion. Assignment alone is not selection; keyless retries zero is exact, keyed retries remain rejected. Bulk keys-only and JSON+verbose remain unsupported. No-wait/empty/dry-run cannot invent accepted submissions.

## Gotchas

- Process key normalization requires 16 decimal digits; planning examples are descriptive. GetJob uses POST /jobs/search with exact jobKey, not GET /jobs/{key}.
- Pool helpers leave zero slots for unscheduled work; coordinators prepopulate not-submitted/skipped outcomes and join caller cancellation explicitly.
- CopyJSONMap converts float64 to json.Number. Copy nested variables conditionally for BID-bearing scalar requests to preserve omitted-input compatibility.
- Generated response parsing can lose HTTP status evidence on malformed advertised JSON; retain unknown classification. HTTP error fixtures need valid problem JSON. Lost responses classify internal (exit 1); HTTP 400 is invalid_input (exit 2).
- New public struct/private approval fields affect unkeyed external literals; use keyed literals. New process API methods require custom implementations/doubles to align; job.API gained no methods.
- Real-terminal fixtures defaulted to 8.8; the shared terminal completion fixture now uses 8.10 so it exercises ordinary and assignment payloads. Assignment logger/terminal tests cover configured/inherited destinations, redirects, plain/structured formats, quiet/WARN, DEBUG flag/config, one envelope+EOF, accept/decline/EOF and empty scopes.
- Docs regeneration also synchronized T003's library README additions into docs/index.md. Generated clients were not changed.

## Reusable Commands

- `env GOFLAGS= go test ./cmd -run 'Test.*(UpdateJob|BulkCompletion)' -count=1` passed iteration 4 (24.992s); final selection has 97 top-level tests including helper entry points. Last-added TestUpdateJobAssignmentInputGuards passed separately (1.013s). Eight new assignment test functions supplement expanded existing matrices.
- `env GOFLAGS= go test ./c8volt/job -run 'Completion|WorkerOutcome' -count=1` passed iteration 4 (0.667s).
- `env GOFLAGS= make test` passed iteration 4: 87 tested packages, 19 packages without tests; cmd race suite 560.531s. The target clears no flags itself, so explicitly clear GOFLAGS; integration live tests require tags. Untagged integration-directory tests validate fixtures/examples only. No inherited C8VOLT/MAKE overrides were present.
- `env GOFLAGS= make docs-content`, gofmt on touched Go files, `git diff --check`, and feature local Markdown link checks passed.
- Negative command tests rejected direct stderr error emission, raw runErr.Error() logging, and a human summary written to stdout; production code was restored before final targeted/full validation.
- Earlier iterations passed direct-assignment domain/service/facade/command targeted tests, affected service/facade race suites, repository compilation, and full cmd race validation. Detailed historical commands/results remain in progress.md; current coverage is in quickstart.md.

## Do Not Repeat

- Do not expose raw joined/transport/generated errors at normal or verbose human levels. Machine error fields retain original evidence; existing redacted DEBUG transport owns HTTP diagnostics.
- Do not run live integration targets/tags, inspect/provision clusters, or probe real engines during implementation. Eligibility, uniqueness configuration, durability and timing remain unverified by these isolated checks.
- Do not serialize scalar approval plans as authorization tokens, mutate generated clients, or add backend orchestration to command/facade layers.

## Current Handoff

- Feature complete; no handoff required.
