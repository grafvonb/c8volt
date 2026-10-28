# Tasks: Process-Definition Lifecycle State

**Feature**: `348-definition-lifecycle-state` · [spec.md](spec.md) · [plan.md](plan.md)

The original three work units remain completed. This compact record preserves their IDs and acceptance ownership; original implementation evidence is retained in [progress.md](progress.md). The 2026-10-28 simplification keeps behavior while consolidating duplicate tests and documentation according to the current plan.

## US1 — Definition state in existing reads (P1, MVP)

- [x] T001 [US1] Preserve optional lifecycle through `internal/domain/processdefinition.go`, `internal/services/processdefinition/v810/convert.go`, `c8volt/process/model.go` and `c8volt/process/convert.go`; display it through `cmd/cmd_views_processdefinition.go`, maintain command output/error contracts in `cmd/get_processdefinition.go`, update the existing CLI observer in `integration/cli/real_state_destructive_test.go`, and provide tests/source/generated docs.

Acceptance: known/unfamiliar values preserved; missing/null/older-version state omitted; lookup/list/page/latest/watch result propagation; human/JSON and existing keys/quiet/empty/XML/error contracts; zero lifecycle-specific requests. Shared state/stat column indices stay aligned. JSON serialization is tested at the model and command-envelope boundaries, not repeatedly in every conversion shape. Covers FR-001–007, FR-013, FR-015–018; SC-001–002, SC-005–006.

## US2 — Independent statistics (P1)

- [x] T002 [US2] Verify independent state/count observations in `cmd/get_processdefinition_suspended_test.go`, `cmd/cmd_views_processdefinition_test.go`, `internal/services/processdefinition/v810/service_test.go` and `c8volt/process/convert_test.go`; document the draining/suspended example in `README.md` and source help.

Acceptance: DRAINING with active=0/suspended=2 survives lookup/list in human/JSON; mixed rows align; missing/unsupported/error statistics retain their existing behavior and requests. Keep the four focused command cases and existing adapter/mixed-row coverage instead of multiplying every state by every count population and output mode. Covers FR-004, FR-007–009, FR-016–017; SC-003, SC-005–006.

## US3 — Existing watch and configured diagnostics (P2)

- [x] T003 [US3] Include state in existing watch comparison/highlighting in `cmd/cmd_views_processdefinition.go`, correct configured watch logging in `cmd/get_processdefinition_watch.go`, retain bounded read errors in `cmd/get_processdefinition.go`, and validate using existing command, timeout, output-failure and terminal fixtures.

Acceptance: state/stat changes count once; first/unchanged/omitted/failed/skipped/DELETED/absent observations preserve markers and baseline semantics. Keep configured logging severity/destinations/formats, INFO filtering, propagated write errors, redacted DEBUG, one JSON envelope and original exits. No new watcher or lifecycle workflow. Covers FR-010–015; SC-004–005.

## Execution and validation

Dependency order remains `T001 → T002 → T003`; overlapping display/fixture/docs ownership does not justify parallel implementation. Each unit includes fixtures, tests, formatting, source docs and generated docs. Follow `AGENTS.md`, the [contracts](contracts/cli.md), and [quickstart](quickstart.md); Ralph additionally follows `specs/ralph-implementation-rules.md`.

Use isolated tests only. No live-service discovery, provisioning, integration execution or explicit no-wait test additions. The existing draining scenario remains available to a separately requested live workflow; its original compile-only evidence does not establish live behavior. Preserve original #346/#347 coverage and completed progress. Commits associated with this issue end in ` #348`.
