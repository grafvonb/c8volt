# Ralph Memory

Feature: 248-slow-process-timeline
Started: 2026-07-22T15:43:37Z

## Codebase Patterns
- Slow-process command wiring is in `cmd/ops_analyse_slow_process_instances.go`; flags are package-level globals reset by `resetOpsSlowProcessAnalysisTestFlags` in `cmd/ops_analyse_slow_process_instances_test.go`.
- Slow-process rendering dispatch is in `cmd/cmd_views_ops_slow_process_analysis.go`; JSON and keys-only are selected before human rendering and must stay unaffected by human display flags.
- The historical full-detail renderer path uses `formatOpsSlowProcessAnalysisTimelineRow`, `formatOpsSlowProcessAnalysisElementRow`, and `formatOpsSlowProcessAnalysisTransitionRow`; after US1 it is reserved for US2 full-timeline dispatch.
- Renderer tests now have neutral fixture builders in `cmd/cmd_views_ops_slow_process_analysis_test.go`: `opsSlowProcessAnalysisRenderTestResult`, `opsSlowProcessAnalysisRenderTestRoot`, `opsSlowProcessAnalysisRenderTestElement`, and `opsSlowProcessAnalysisRenderTestTransition`.
- Default human slow-process output now renders `└─ slowest elements:` from `opsSlowProcessAnalysisDefaultHotspotSummary`; visible rows are element rows that are completed with `ProcessDurationShare >= 1`, active, or incident-bearing, sorted by share then duration.
- Default summary row formatting is handled by `formatOpsSlowProcessAnalysisSummaryRow`; it omits element instance keys, except `eik:<key>` when an incident row lacks an incident key, and keeps `inc!`/`inc!:<incidentKey>` markers.
- Hidden default detail is rendered as `hidden: N instant/fast timeline row(s); use --with-full-timeline` and counts omitted analyzed timeline rows only, not the root.
- `--with-full-timeline` is a command-local bool flag (`flagOpsAnalyseSlowProcessInstanceWithFullTimeline`) registered in `cmd/ops_analyse_slow_process_instances.go`; it is parsed into `opsSlowProcessAnalysisCommandRequest.WithFullTimeline` for command tests but is intentionally not added to facade/domain request structs.
- Full-timeline human rendering is dispatched only after JSON and keys-only handling in `renderOpsSlowProcessAnalysisResult`; it calls `renderOpsSlowProcessAnalysisFullTimeline`, prints `└─ elements:`, and reuses `formatOpsSlowProcessAnalysisTimelineRow` for chronological element/transition rows.
- US3 locks machine output with renderer tests that compare JSON and keys-only output with and without `flagOpsAnalyseSlowProcessInstanceWithFullTimeline`; those tests assert no `hidden:`, `slowest elements:`, or summary/full-timeline fields leak into machine output.
- Command parsing tests now prove `--with-full-timeline` is accepted while `--json` or `--keys-only` is active and does not alter selection, filters, root duration, batch size, or limit in `ops.SlowProcessAnalysisRequest`.

## Decisions
- No feature-artifact conflict found between spec, plan, CLI contract, and `specs/ralph-implementation-rules.md` during setup.
- Keep the service/domain/facade slow-process payload unchanged; `internal/services/ops/slow_process_analysis.go` builds the complete timeline, applies existing detail filters, and JSON exposes that complete render-independent result.
- US1 intentionally changed default human output only; JSON and keys-only dispatch still happens before human summary selection.
- US2 keeps `--with-full-timeline` human-only; it restores the pre-US1 chronological row style without changing selection, detail filtering, service payload, JSON, or keys-only output.
- US3 confirmed the existing renderer dispatch already checks JSON and keys-only before human full-timeline branching; no renderer implementation change was needed.
- US3 confirmed `c8volt/ops/model.go`, `c8volt/ops/convert.go`, and `internal/domain/ops_slow_process_analysis.go` have no human-only full-timeline or hidden-row fields.

## Gotchas
- Do not move summary or hidden-row data into `c8volt/ops/model.go` or `internal/domain/ops_slow_process_analysis.go`; those structs are part of the JSON/public payload and must not gain human-only fields.
- Existing full chronological `elements:` formatting helpers remain in `cmd/cmd_views_ops_slow_process_analysis.go` but are no longer used by the default human path; US2 should reuse them for `--with-full-timeline`.
- Command metadata/docs tests cover examples and flag contracts in `cmd/command_contract_test.go`; docs content under `docs/cli/` must be regenerated only after command metadata changes.
- Because command metadata changed for `--with-full-timeline`, the later polish phase must update `docsgen/main_test.go` expectations and run `make docs-content`; this iteration intentionally did not hand-edit generated docs.

## Reusable Commands
- `go test ./cmd -run 'TestRenderOpsSlowProcessAnalysisResultHuman' -count=1`
- `go test ./cmd -run 'TestRenderOpsSlowProcessAnalysisResult' -count=1`
- `go test ./cmd -run 'TestOpsAnalyseSlowProcessInstances|TestRenderOpsSlowProcessAnalysisResultHuman|TestCommandContractOpsAnalyseSlowProcessInstances' -count=1`
- `go test ./cmd -run 'TestRenderOpsSlowProcessAnalysisResult.*JSON|TestRenderOpsSlowProcessAnalysisResult.*KeysOnly|TestOpsAnalyseSlowProcessInstances' -count=1`

## Do Not Repeat
- Do not re-review the entire service/facade payload for T005 unless behavior changes require it; setup confirmed it remains complete and render-independent.

## Current Handoff
- Next iteration should start Phase 6 polish at T027/T028/T029. Update README/quickstart/docsgen expectations for compact default summaries and `--with-full-timeline`, then run docs generation and broader validation.
