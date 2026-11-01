# Ralph Memory

Feature: 361-business-id-selection
Started: 2026-11-01T11:22:16Z

## Codebase Patterns

- `ResolveProcessInstanceSelection` is a shared process-service function, exposed by a thin `process.API` method. It uses existing adapter APIs, validates capability before selection, copies criteria/observations (including nested variables), and returns ordered distinct keys, completion, page count, tenant evidence, `Origin: search`, and the evaluated `IgnoreTenant` flag. Check errors before using results: a unique-start limit remains bounded evidence, not uniqueness. A returned record without a technical key is malformed/incomplete, never an empty selection.
- `searchpaging` is the narrow single-page traversal subpackage shared by ordinary search/total and v810 child lookup; the parent package imports version factories, so v810 cannot import it directly. Do not add a second child paging loop. v88/v89 child behavior stays unchanged.
- v810 child lookup exhausts all pages at every depth with page size 1000 and no total cap. Walker/traversal results preserve observed chain/edges on failure; failures remain non-complete and stop recursion. Successful edges/DFS keys retain membership/order for later planners.
- BID walk uses command-local `--bid`, service-owned `unique_start`, and ordinary `collectOptions` for tenant-scoped discovery. Only after selection does it enter the existing key-authorized traversal/enrichment path. Zero exhausted matches use the empty view without any walk/enrichment call. JSON failures retain selection/traversal evidence and original error classification in one envelope; render write failures never attempt a second envelope.

- Native BID search uses copied optional `BusinessId *string` criteria in PI filters and task requests. `common.ValidateBusinessIDSearch` checks presence, adapter capability, then nonempty valid UTF-8, without creation-length rules. v810 encodes the exact scalar; v88/v89 reject before HTTP.
- `ValidateProcessInstanceSearchFilter` exists on all PI adapters and the thin public `process.API`. Shared search invokes it before shortcut selection. `get pi` also invokes the public delegate before process-definition selector validation, so an empty definition lookup cannot bypass unsupported BID rejection.
- CLI BID presence comes from the command-local scalar flag through `searchBusinessIDFlag`, never shared global flag values. Only `get pi`/`get ut` register it in T002. Shared PI filter helpers accept an optional command; later consumers must explicitly wire their own validated BID modes.
- A supplied BID disables the direct-incident index in both CLI eligibility and service normalization. A forced library shortcut retains its direct-incident predicate and detail filters, then performs native PI selection and local direct-incident checks before applying the selected limit.
- BID query failures use `handleBusinessIDSearchError`: bounded configured ERROR, known selected count for collected enrichment, original error/envelope/exit retained. Task BID flag validation runs after bootstrap so JSON validation errors use the shared envelope and configured logging.

- Process discovery is owned by `internal/services/processinstance/search.go`; `searchpaging/paging.go` now owns shared collection/count/child traversal, metadata validation, cursor/offset progress and terminal probes. Keep facade visitors mechanical.
- Public/domain `ProcessInstanceSearchPagesResult.Completion` uses `exhausted`, `limit_reached`, `visitor_stopped`, `incomplete`. Never accept zero/unknown values as completion evidence. A limit takes precedence even on a terminal page; unique selection must require exhaustion.
- Distinct selected keys are counted after local filtering. Raw page size drives advancement; sparse pages advance via fresh cursors or offsets. Counted totals deduplicate keys; compatible exact totals retain the no-extra-read path.
- Direct incident-index search loads previously unseen PI keys per incident page, filters them before consuming the selected bound, and invokes its query visitor once. It preserves partial loaded items on a later incident/load error.
- Facade paging maps partial items/completion before converting errors; total facade now preserves partial counts with the converted error. Existing query JSON payloads are unchanged.

## Decisions

- T003 delivers US2 only: shared collection/unique selection, complete v810 children, and BID walk. No mutation/ops BID flags or frozen mutation plans are introduced. The public API addition requires the command and resource facade stubs to implement selection; adapter interfaces need no new selector method because orchestration uses the existing shared service-function pattern.

- T002 delivers US1 only: native PI/task queries, copied process/ops filter conversions, native task snapshots, both shortcut guards, no-I/O capability checks and query output contracts. It does not add BID flags to mutations, walk or ops.
- Public `process.API.ValidateProcessInstanceSearchFilter` is an intentional additive interface requirement, needed to reject BID before related definition lookups without moving adapter policy into CLI code. README documents custom implementations and unkeyed literal compatibility.

- T001 is the sole Phase 2 work unit. No BID filter, selection API, traversal expansion or mutation lifecycle from later stories is implemented here.
- Terminal and limit evidence take precedence over a visitor returning Stop for those same conditions; deliberate early stops retain `visitor_stopped`. Ordinary CLI continuation/prompt policy remains unchanged, including stopping indeterminate query pages.
- Validate per-page continuation facts, not a fixed cross-page total: legacy paged mutations can legitimately shrink backend totals while running. Exact count-only metadata can be valid without any returned rows.
- Indeterminate completion uses observed distinct raw keys, never offsets advanced across sparse pages. An ambiguous empty page with an unmet reported count and no cursor fails incomplete. Explicit authoritative no-more responses retain their existing sparse-page interpretation.
- Metadata errors are bounded and never include cursor values. Existing service errors retain original error identity/classification through the facade.

## Gotchas

- Walk no longer marks `--key` unconditionally required: key-or-BID validation runs after bootstrap for configured diagnostics/one JSON envelope. Enrichment validators accept BID but retain their established output incompatibilities. Keep required-key metadata assertions aligned; use isolated subprocesses for no-selector errors because the command exits.
- The shared envelope success spelling is `succeeded`, with result data under `payload`. Listener enrichment can query `/v2/jobs/search` even with no element rows. Use owned fixture responses with authoritative empty page metadata.

- v810 PI search serializes `processInstanceFilter`, not the generated filter directly; update its fields and `isEmpty()` together. Both facade process/ops conversion directions must copy BID.
- Native variable equality is serialized as a `$eq` object, even though BID is a scalar. Variable/effective-variable empty fixtures need `hasMoreTotalItems:false` as well as `totalItems:0`.
- Cobra Args validation precedes command logger/bootstrap. For new BID user-task input, defer validation to Run before stdin or native requests so invalid JSON yields exactly one envelope; legacy omitted-BID validation behavior stays intact.

- The current query quiet mode retains nonempty ordinary rows; empty human summaries are suppressed. Do not silently change this while adding BID.
- Existing capped-total tests had terminal empty pages still promising 10,000 matches. Their terminal estimates now reconcile to the three observed results; rejecting contradictory terminal metadata is intentional.
- Repeated offset page signatures and repeated/cyclic cursors fail incomplete. A zero-item selected page with valid continuation must keep going.
- Public struct fields are additive but unkeyed literals need updating; custom facade implementations must explicitly supply completion before later selection consumers trust them. README documents this.
- `runBusinessIDCommand` in the command tests is reusable for actual root execution with real terminal stdin, separate streams, configured/inherited logger destinations and original exit codes, despite its feature-specific name.

## Reusable Commands

- T003 service fixtures: `TestBusinessIDSelectionCardinality`, `TestBusinessIDSelectionPartialFailure`, `TestBusinessIDSelectionCopies`, `TestBusinessIDSelectionVersionGate`, and `TestCompleteChildrenTraversal`; facade: `TestBusinessIDSelectionFacade`.
- T003 command fixtures: `TestBusinessIDSelectionWalk*` and the `walk` branch of `TestBusinessIDSelectionFailureLogging`. They cover real-terminal empty scopes, tenant transitions, cardinality, complete 1003-member JSON, late/cyclic failures, one-envelope output errors, and repeated enrichment failures with both logger formats/destinations and independent DEBUG.

- T002 focused tests: `go test ./internal/services/common ./internal/services/processinstance ./internal/services/usertask ./c8volt/process ./c8volt/task ./c8volt/ops -run 'Test.*BusinessID(Search|Selection)' -count=1`; command coverage is `go test ./cmd -run '^TestBusinessIDSelection' -count=1` (list new names first).
- T002 durable command fixtures are in `cmd/business_id_selection_test.go`: scalar/AND/tenant requests, sparse pages, modes/empty completion with real terminal stdin, exclusivity, direct-incident fallback and configured logging matrix. Reuse `runBusinessIDCommand`; it rejects writer bypasses.

- Prerequisites: `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`.
- Use the isolation wrapper from `quickstart.md`: strip `C8VOLT_`, `CAMUNDA_`, `ZEEBE_`, `GO_WANT_`, Kubernetes/make overrides; set `GOFLAGS=''`, `GOENV=off`. The iteration's temporary wrapper is `/tmp/c8volt-361-validation.py`; recreate it if absent.
- Owner checks: `go test ./internal/services/processinstance/... ./c8volt/process -run 'Test.*(BusinessID|Search|Selection|Children|Traversal|DryRun|SelectedMutation|Delete|Discover|Convert|Walk|Plan|Update|Resolve|Suspend|Resume)' -count=1`.
- Command checks: `go test ./cmd -run 'Test.*(ProcessInstance.*(Search|Paging)|GetProcessInstance|ConfirmationEmptySelectorResults)' -count=1`.
- List newly added test names with `-list` before running them. All test dependencies are owned mocks/HTTP fakes/PTY fixtures.
- Generate docs with `make docs-content`; full isolated breadth is `make test PKG=./...` (race enabled, integration tags disabled).

## Do Not Repeat

- T003 full race breadth found the resource facade stub, two stale suspend/resume 1000-child metadata assertions and loss of walk selector completion after removing required-key annotation. All were corrected and passed focused race reruns. Walk now suggests both selectors locally via `completeWalkPISelector`; no mandatory flag annotation is needed. Final selection/command race checks also passed after the malformed-key guard. The full suite was not repeated for the narrowly verified follow-up changes.

- When adding to `process.API`, update both `cmd/process_api_stub_test.go` and `c8volt/resource/client_test.go`. Untagged files beneath `integration/` include harmless parser/config and test-owned HTTP fixtures; real engine test entrypoints require the integration build tag. Do not confuse those offline helpers with live validation.

- When changing command help, update `cmd/command_contract_test.go` and `docsgen/main_test.go` wording assertions with the metadata. The first full race run found only these two stale assertions; both were corrected and passed targeted race reruns. All runtime tests in that run passed. The final affected-owner/command race run also passed after the last runtime change.

- Do not drop prior results on later failures, count duplicate keys toward limits, or infer exhaustion from an empty selected page.
- Do not validate exact count-only requests by requiring row traversal: the count can be returned without rows.
- Do not equate offset progress with observed matches when resolving indeterminate completion.
- Do not implement paging in commands/facades, change ordinary query prompt policy, run live checks, or spawn implementation agents without authorization.

## Current Handoff

- T001–T003 are complete. T004 (US3 frozen cross-family BID cancellation) is next dependency-ready; recheck against `tasks.md`. Use the shared collection resolver, exhaustive v810 child traversal, retained edges/observations and existing dry-run family expansion. Preserve search tenant options separately from later key-authorized execution. No blocking dependency is known. Seven feature tasks remain.
