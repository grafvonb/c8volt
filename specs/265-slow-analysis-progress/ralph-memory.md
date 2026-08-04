# Ralph Memory

Feature: 265-slow-analysis-progress
Started: 2026-08-04T16:54:15Z

## Codebase Patterns

- `cmd/ops_progress.go` owns shared progress channel selection, compact human preflight/page/frozen-scope/ETA formatters, durable stderr printing through `printOpsDurableLine`, and machine-output-safe gating via `opsProgressChannelForMode`.
- `cmd/ops_analyse_slow_process_instances.go` wires broad slow-analysis search progress in `configureOpsSlowProcessAnalysisPreflight`; current callback handles Preflight, Page, FrozenScope, and ETA events, updates workflow activity through `printOpsSlowProcessAnalysisProgress`, and prompts through `ConfirmPreflight`.
- `printOpsSlowProcessAnalysisProgress` currently updates `logging.ActivityImportanceWorkflow` whenever `channel.TransientAllowed` is true, then writes durable lines only when `opsSlowProcessAnalysisDurableProgressAllowed` permits verbose/debug stderr.
- `internal/services/ops/slow_process_analysis.go` already emits structured progress events only: preflight before confirmation, page progress after each discovery page, frozen-scope counters before and during enrichment, plus timing facts in the frozen-scope event.
- `internal/domain/ops_progress.go` defines the shared progress event model and `OpsDefaultETAMinimumSamples`/`OpsMinimumTimingElapsed`; command milestone pacing should use command-owned constants/state rather than adding CLI policy to domain or services.
- `toolx/logging/activity.go` supports priority activity scopes and updates; workflow importance outranks HTTP, wait, and batch activity, so nested runtime lookups should not hide slow-analysis workflow progress when updates use `ActivityImportanceWorkflow`.
- Existing tests in `cmd/ops_progress_test.go` cover formatter wording, ETA gating, and `opsProgressChannelForMode` stdout safety; slow-analysis progress tests around `cmd/ops_analyse_slow_process_instances_test.go` cover default activity updates, verbose durable stderr, JSON/keys-only/quiet/automation suppression, and preflight prompt gating.

## Decisions

- Keep milestone pacing in `cmd/ops_progress.go` as shared command-rendering policy. Slow-analysis wiring should instantiate/use that state, but internal services must remain structured-event-only.
- Default human durable milestones must require elapsed silence plus a changed progress signature; timer-only output is not allowed.
- Verbose/debug durable behavior should remain immediate and detailed through the existing `opsSlowProcessAnalysisDurableProgressAllowed` path.

## Gotchas

- Default human `opsProgressChannelForMode` has `DurableAllowed` and `StderrAllowed` true today, but slow-analysis page/counter durable writes are suppressed by `opsSlowProcessAnalysisDurableProgressAllowed` unless mode is verbose/debug.
- Preflight scope lines already write durably in human modes before confirmation; new pacing should apply to post-confirmation page/frozen-scope/ETA-style progress, not preflight prompt text.
- JSON, keys-only, quiet, and automation tests assert both stdout and stderr stay empty for progress callbacks; milestone gating must keep these modes silent even though default human can use stderr.
- The slow-analysis service emits an initial frozen-scope event with `Done: 0`; pacing signatures should avoid treating unchanged or timer-only repeats as forward progress.

## Reusable Commands

- `go test ./cmd -run 'TestOps.*Progress|TestFormatOps.*Progress|TestOpsETA' -count=1`
- `go test ./cmd -run 'TestOpsAnalyseSlowProcessInstances.*Progress|TestOpsAnalyseSlowProcessInstances.*Preflight' -count=1`
- `go test ./internal/services/ops -run 'TestSlowProcessAnalysis.*Progress|TestSlowProcessAnalysis.*Preflight' -count=1`
- `make test`

## Do Not Repeat

- Do not move milestone pacing, wording, stderr routing, or mode gating into `internal/services/ops/slow_process_analysis.go`.
- Do not add timer-driven "still working" lines without observed discovery or frozen-scope counter movement.
- Do not change command help or generated docs unless the shipped help wording changes; run `make docs-content` if it does.

## Current Handoff
- Next iteration should start Phase 2 foundational tasks T003-T006: add shared milestone pacing and output-mode gating tests in `cmd/ops_progress_test.go`, implement shared command-owned state/helpers in `cmd/ops_progress.go`, then run the targeted shared progress test command.
