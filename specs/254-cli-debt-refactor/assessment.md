# CLI Debt Refactor Assessment

Feature: 254-cli-debt-refactor
Created: 2026-07-24

## Phase 1 Setup Findings

### Context And Rules Review

- Reviewed `specs/ralph-implementation-rules.md` against `specs/254-cli-debt-refactor/spec.md`; no conflict found.
- Binding ownership boundary for this feature: `cmd` owns flags, validation, prompts, render-mode selection, stdout/stderr rendering, command metadata, and help; facades map public inputs and errors; internal services own backend paging, traversal, frozen discovery, mutation planning, polling, retries, and worker execution.
- The feature's first deliverable is a checked-in assessment. Refactor tasks must not start until the full assessment structure and all 55 command-node classifications are complete.

### Basic Paging Implementations

Reviewed files: `cmd/get_job_search.go`, `cmd/get_element_search.go`, `cmd/get_incident_search.go`, and `cmd/get_processinstance_search.go`.

- `get job` and `get element` both implement command-owned offset page loops with the same mechanics: compute initial page request, call `Search*Page`, trim items to `--limit`, optionally stream one-line or keys-only output, collect JSON results, prompt when more data remains, and compute total fallback by walking pages.
- `get incident` follows the same command-owned pattern with incident-specific concerns: cursor fallback via `EndCursor`, `--process-instance-keys-only`, incident detail row rendering, and a shared continuation-state vocabulary borrowed from process-instance paging.
- `get process-instance` is the highest-risk basic read path. It owns backend page traversal, local filters, direct incident-index search strategy, enrichment invocation boundaries, incremental rendering, warning-stop handling, and verbose progress.
- Candidate mechanics for later ownership reduction: page walking, page-size normalization, total fallback, limit trimming, cursor/offset advancement, direct incident-index candidate lookup, and process-instance local compatibility filtering.
- CLI-owned behavior to preserve during later refactors: interactive prompts, one-line and keys-only streaming, JSON collection into one document, found summaries, warning text, and render-mode decisions.

### Process-Instance Mutation Paging

Reviewed files: `cmd/get_processinstance_paging.go`, `cmd/cancel_processinstance.go`, and `cmd/delete_processinstance.go`.

- `processPISearchPagesWithAction` centralizes some command-level cancel/delete paging mechanics: per-page search, local `--limit` trimming, continuation state, first-page confirmation, dry-run aggregation, partial-stop progress, and page-by-page action invocation.
- `cancel process-instance` still maps search pages directly to keys and submits page actions through command-owned planning/render functions. Confirmation safety depends on first-page impact counts and must be preserved.
- `delete process-instance` has an extra planning path for non-dry-run search mode: it walks all selected pages, builds delete previews without mutating, aggregates the frozen delete plan, checks non-final scope, prompts once, then deletes the frozen roots. This behavior is intentionally different from cancel and should not be flattened without preserving frozen-scope safety.
- Later refactor candidates: move search-selected discovery traversal and delete/cancel mutation planning below command ownership while keeping confirmation prompts, renderers, dry-run presentation, and final outcome handling in `cmd`.

### Ops Workflow Ownership

Reviewed files under `internal/services/ops/` plus command wiring and renderers under `cmd/ops_*.go` and `cmd/cmd_views_ops_*.go`.

- Ops services already own most workflow mechanics: incident purge freezes incident-derived process-instance candidates; all-process-definition purge freezes process-definition candidates; retention policy delegates retention discovery to `internal/services/processinstance`; repair workflows freeze either incident or process-instance targets before mutation; slow-process analysis performs read-only discovery and enrichment below `cmd`.
- Several ops paths already use bounded worker helpers for independent work: repair execution uses `toolx/pool.ExecuteSlice`, retention dependency expansion uses `pool.ExecuteSlice`, and process-definition purge delegates worker-aware planning to process-definition services.
- Similar discovery loops are not automatically equivalent. Incident purge tracks skipped incidents and duplicate process-instance candidates; all-process-definition purge tracks latest/all-version process-definition semantics and active-instance impact; retention policy tracks retention boundaries and non-final root skips; slow-process analysis tracks read-only duration/enrichment filters.
- Later refactor work should extract only mechanics with matching semantics. The safest near-term ops targets are progress-summary rendering consistency and targeted high-volume characterization, not a generic ops discovery abstraction.

### Output, Activity, And Capability Helpers

Reviewed files: `cmd/root.go`, `cmd/command_contract.go`, `toolx/logging/activity.go`, `cmd/cmd_views_ops_repair.go`, `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`, and `cmd/cmd_views_ops_purge_all_processdefinitions.go`.

- Root command creates the shared activity writer with `indicatorEnabled(cmd, cfg)` and installs it in context for HTTP and command activity sinks. Activity output is routed through stderr and gated by terminal interactivity.
- `toolx/logging.ActivityWriter` clears transient spinner output before normal writes, tracks nested activity references, supports message updates, and is disabled when no interactive terminal is present.
- Capability metadata is annotation-driven in `cmd/command_contract.go`; command support defaults are inferred from command tree shape unless commands set mutation, contract support, automation support, output modes, or required-flag metadata explicitly.
- Ops renderers already distinguish user-limited discovery in compact human output and hide complete discovery page detail unless `--verbose` is enabled. JSON/report paths retain complete discovery state through shared result rendering.
- Later policy work should align basic search progress with the ops pattern: default compact output, verbose durable page details away from stdout, and machine-output silence for JSON, keys-only, quiet, automation, and no-indicator scenarios.
