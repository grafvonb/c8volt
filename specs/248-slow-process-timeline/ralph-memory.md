# Ralph Memory

Feature: 248-slow-process-timeline
Started: 2026-07-22T15:43:37Z

## Codebase Patterns
- Slow-process command wiring is in `cmd/ops_analyse_slow_process_instances.go`; flags are package-level globals reset by `resetOpsSlowProcessAnalysisTestFlags` in `cmd/ops_analyse_slow_process_instances_test.go`.
- Slow-process rendering dispatch is in `cmd/cmd_views_ops_slow_process_analysis.go`; JSON and keys-only are selected before human rendering and must stay unaffected by human display flags.
- Existing human output currently renders every detail row under `└─ elements:` using `formatOpsSlowProcessAnalysisTimelineRow`, `formatOpsSlowProcessAnalysisElementRow`, and `formatOpsSlowProcessAnalysisTransitionRow`.
- Renderer tests now have neutral fixture builders in `cmd/cmd_views_ops_slow_process_analysis_test.go`: `opsSlowProcessAnalysisRenderTestResult`, `opsSlowProcessAnalysisRenderTestRoot`, `opsSlowProcessAnalysisRenderTestElement`, and `opsSlowProcessAnalysisRenderTestTransition`.

## Decisions
- No feature-artifact conflict found between spec, plan, CLI contract, and `specs/ralph-implementation-rules.md` during setup.
- Keep the service/domain/facade slow-process payload unchanged; `internal/services/ops/slow_process_analysis.go` builds the complete timeline, applies existing detail filters, and JSON exposes that complete render-independent result.
- Foundation introduced `opsSlowProcessAnalysisDefaultHotspotSummary` and `opsSlowProcessAnalysisHotspotMinimumProcessShare` as behavior-neutral renderer scaffolding; it currently copies all timeline rows and reports zero hidden rows.

## Gotchas
- Do not move summary or hidden-row data into `c8volt/ops/model.go` or `internal/domain/ops_slow_process_analysis.go`; those structs are part of the JSON/public payload and must not gain human-only fields.
- Existing renderer tests still assert the old `elements:` full timeline behavior until US1/US2 intentionally changes expectations.
- Command metadata/docs tests cover examples and flag contracts in `cmd/command_contract_test.go`; docs content under `docs/cli/` must be regenerated only after command metadata changes.

## Reusable Commands
- `go test ./cmd -run 'TestRenderOpsSlowProcessAnalysisResultHuman|TestOpsSlowProcessAnalysisDefaultHotspotSummaryScaffold' -count=1`

## Do Not Repeat
- Do not re-review the entire service/facade payload for T005 unless behavior changes require it; setup confirmed it remains complete and render-independent.

## Current Handoff
- Next iteration should start User Story 1 at T008/T009 by replacing old default human renderer expectations with `slowest elements:` summary tests, using the new fixture builders and preserving JSON/keys-only dispatch order.
