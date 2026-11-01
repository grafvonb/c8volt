# Ralph Memory

Feature: 361-business-id-selection
Started: 2026-11-01T11:22:16Z

## Codebase Patterns

- Native BID search uses copied optional `BusinessId *string` criteria in PI filters and task requests. `common.ValidateBusinessIDSearch` checks presence, adapter capability, then nonempty valid UTF-8, without creation-length rules. v810 encodes the exact scalar; v88/v89 reject before HTTP.
- `ValidateProcessInstanceSearchFilter` exists on all PI adapters and the thin public `process.API`. Shared search invokes it before shortcut selection. `get pi` also invokes the public delegate before process-definition selector validation, so an empty definition lookup cannot bypass unsupported BID rejection.
- CLI BID presence comes from the command-local scalar flag through `searchBusinessIDFlag`, never shared global flag values. Only `get pi`/`get ut` register it in T002. Shared PI filter helpers accept an optional command; later consumers must explicitly wire their own validated BID modes.
- A supplied BID disables the direct-incident index in both CLI eligibility and service normalization. A forced library shortcut retains its direct-incident predicate and detail filters, then performs native PI selection and local direct-incident checks before applying the selected limit.
- BID query failures use `handleBusinessIDSearchError`: bounded configured ERROR, known selected count for collected enrichment, original error/envelope/exit retained. Task BID flag validation runs after bootstrap so JSON validation errors use the shared envelope and configured logging.

- Process discovery is owned by `internal/services/processinstance/search.go`; `search_paging.go` now owns shared collection/count traversal, metadata validation, cursor/offset progress and terminal probes. Keep facade visitors mechanical.
- Public/domain `ProcessInstanceSearchPagesResult.Completion` uses `exhausted`, `limit_reached`, `visitor_stopped`, `incomplete`. Never accept zero/unknown values as completion evidence. A limit takes precedence even on a terminal page; unique selection must require exhaustion.
- Distinct selected keys are counted after local filtering. Raw page size drives advancement; sparse pages advance via fresh cursors or offsets. Counted totals deduplicate keys; compatible exact totals retain the no-extra-read path.
- Direct incident-index search loads previously unseen PI keys per incident page, filters them before consuming the selected bound, and invokes its query visitor once. It preserves partial loaded items on a later incident/load error.
- Facade paging maps partial items/completion before converting errors; total facade now preserves partial counts with the converted error. Existing query JSON payloads are unchanged.

## Decisions

- T002 delivers US1 only: native PI/task queries, copied process/ops filter conversions, native task snapshots, both shortcut guards, no-I/O capability checks and query output contracts. It does not add BID flags to mutations, walk or ops.
- Public `process.API.ValidateProcessInstanceSearchFilter` is an intentional additive interface requirement, needed to reject BID before related definition lookups without moving adapter policy into CLI code. README documents custom implementations and unkeyed literal compatibility.

- T001 is the sole Phase 2 work unit. No BID filter, selection API, traversal expansion or mutation lifecycle from later stories is implemented here.
- Terminal and limit evidence take precedence over a visitor returning Stop for those same conditions; deliberate early stops retain `visitor_stopped`. Ordinary CLI continuation/prompt policy remains unchanged, including stopping indeterminate query pages.
- Validate per-page continuation facts, not a fixed cross-page total: legacy paged mutations can legitimately shrink backend totals while running. Exact count-only metadata can be valid without any returned rows.
- Indeterminate completion uses observed distinct raw keys, never offsets advanced across sparse pages. An ambiguous empty page with an unmet reported count and no cursor fails incomplete. Explicit authoritative no-more responses retain their existing sparse-page interpretation.
- Metadata errors are bounded and never include cursor values. Existing service errors retain original error identity/classification through the facade.

## Gotchas

- v810 PI search serializes `processInstanceFilter`, not the generated filter directly; update its fields and `isEmpty()` together. Both facade process/ops conversion directions must copy BID.
- Native variable equality is serialized as a `$eq` object, even though BID is a scalar. Variable/effective-variable empty fixtures need `hasMoreTotalItems:false` as well as `totalItems:0`.
- Cobra Args validation precedes command logger/bootstrap. For new BID user-task input, defer validation to Run before stdin or native requests so invalid JSON yields exactly one envelope; legacy omitted-BID validation behavior stays intact.

- The current query quiet mode retains nonempty ordinary rows; empty human summaries are suppressed. Do not silently change this while adding BID.
- Existing capped-total tests had terminal empty pages still promising 10,000 matches. Their terminal estimates now reconcile to the three observed results; rejecting contradictory terminal metadata is intentional.
- Repeated offset page signatures and repeated/cyclic cursors fail incomplete. A zero-item selected page with valid continuation must keep going.
- Public struct fields are additive but unkeyed literals need updating; custom facade implementations must explicitly supply completion before later selection consumers trust them. README documents this.
- `runBusinessIDCommand` in the command tests is reusable for actual root execution with real terminal stdin, separate streams, configured/inherited logger destinations and original exit codes, despite its feature-specific name.

## Reusable Commands

- T002 focused tests: `go test ./internal/services/common ./internal/services/processinstance ./internal/services/usertask ./c8volt/process ./c8volt/task ./c8volt/ops -run 'Test.*BusinessID(Search|Selection)' -count=1`; command coverage is `go test ./cmd -run '^TestBusinessIDSelection' -count=1` (list new names first).
- T002 durable command fixtures are in `cmd/business_id_selection_test.go`: scalar/AND/tenant requests, sparse pages, modes/empty completion with real terminal stdin, exclusivity, direct-incident fallback and configured logging matrix. Reuse `runBusinessIDCommand`; it rejects writer bypasses.

- Prerequisites: `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`.
- Use the isolation wrapper from `quickstart.md`: strip `C8VOLT_`, `CAMUNDA_`, `ZEEBE_`, `GO_WANT_`, Kubernetes/make overrides; set `GOFLAGS=''`, `GOENV=off`. The iteration's temporary wrapper is `/tmp/c8volt-361-validation.py`; recreate it if absent.
- Owner checks: `go test ./internal/services/processinstance/... ./c8volt/process -run 'Test.*(BusinessID|Search|Selection|Children|Traversal|DryRun|SelectedMutation|Delete|Discover|Convert|Walk|Plan|Update|Resolve|Suspend|Resume)' -count=1`.
- Command checks: `go test ./cmd -run 'Test.*(ProcessInstance.*(Search|Paging)|GetProcessInstance|ConfirmationEmptySelectorResults)' -count=1`.
- List newly added test names with `-list` before running them. All test dependencies are owned mocks/HTTP fakes/PTY fixtures.
- Generate docs with `make docs-content`; full isolated breadth is `make test PKG=./...` (race enabled, integration tags disabled).

## Do Not Repeat

- When changing command help, update `cmd/command_contract_test.go` and `docsgen/main_test.go` wording assertions with the metadata. The first full race run found only these two stale assertions; both were corrected and passed targeted race reruns. All runtime tests in that run passed. The final affected-owner/command race run also passed after the last runtime change.

- Do not drop prior results on later failures, count duplicate keys toward limits, or infer exhaustion from an empty selected page.
- Do not validate exact count-only requests by requiring row traversal: the count can be returned without rows.
- Do not equate offset progress with observed matches when resolving indeterminate completion.
- Do not implement paging in commands/facades, change ordinary query prompt policy, run live checks, or spawn implementation agents without authorization.

## Current Handoff

- T001 and T002 are complete. T003 (US2 complete BID walk selection and traversal) is next dependency-ready; recheck against `tasks.md`. Reuse the native filter policy and no-I/O validator before selection/frozen-plan shortcuts, the trustworthy PI paging engine, and copied filter conversions. No blocking dependency is known. Eight feature tasks remain.
