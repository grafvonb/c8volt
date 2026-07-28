# Ops-Scale Progress Coverage Inventory

This inventory records the Phase 7 audit for high-volume command families against
the shared CLI progress contract.

## Coverage Matrix

| Command family | Commands | High-volume trigger | Existing coverage | Contract gaps | Follow-up slice |
| --- | --- | --- | --- | --- | --- |
| Proof ops analysis | `ops analyse slow-process-instances` | Broad `--bpmn-process-id` or `--pd-key` discovery, then process-instance enrichment | Shared preflight, page progress, frozen-scope counters, ETA gating, and machine-output safety are implemented for process-definition search mode | Help/docs polish remains; explicit-key mode intentionally skips broad preflight | Phase 8 polish tasks |
| Basic process-instance inspection | `get process-instance` | Search filters, broad process-definition selectors, `--with-incidents`, `--with-vars`, `--with-elements`, `--with-listeners` | Service-owned page traversal exists; verbose paging diagnostics and interactive continuation prompts are stdout-safe | No shared preflight before broad enrichment; page progress uses older command-local summary instead of `OpsPageProgress`; frozen enrichment has no exact counters/ETA; command-local limit/progress helpers remain from pre-progress behavior | Basic inspection rollout |
| Basic incident inspection | `get incident` | Incident search filters, broad BPMN/process selectors, `--pi-keys-only` pipelines | Service-owned incident page traversal exists; verbose paging diagnostics and prompt gating are stdout-safe | No preflight for broad incident searches; page progress is older generic wording without total-certainty/page-count model; no frozen progress for process-instance-key extraction; machine-mode safety needs shared progress assertions after integration | Basic inspection rollout |
| Basic job inspection | `get job` | Job search filters over failed/active/high-volume jobs | Service-owned job page traversal exists; verbose paging diagnostics and prompt gating are stdout-safe | No preflight; job page progress has only has-more wording and no total certainty; no shared progress-channel tests for JSON/keys-only/quiet/automation after rollout | Basic inspection rollout |
| Basic element inspection | `get element` | Runtime element search filters and `--with-listeners` enrichment | Service-owned element page traversal exists for plain search; verbose paging diagnostics and prompt gating are stdout-safe | No broad preflight; listener-enriched search collects full results without frozen-scope listener progress; page progress has no shared total-certainty wording | Basic inspection rollout |
| Process-definition inspection | `get process-definition` | Listing many deployed definitions, latest-version scans, optional `--stat` | Read-only listing and selector validation exist; `--stat` has lower-level activity in version services | No command-level `--batch-size`/`--limit` controls; no shared page traversal/preflight for broad definition listing; `--stat` progress is not in the shared ops progress model | Process-definition rollout |
| Process-instance cancellation | `cancel process-instance` | Search-selected cancel with dependency expansion and optional waits | Uses `PlanProcessInstanceMutationPages`; page prompts and destructive confirmation preserve current safety semantics | No preflight before broad discovery/planning; cancel mutates page-by-page instead of showing a whole-run frozen-scope confirmation; planning, mutation, and wait progress are not emitted as shared frozen counters | Process-instance mutation rollout |
| Process-instance deletion | `delete process-instance` | Search-selected delete with dependency expansion and optional force/wait | Uses `PlanProcessInstanceMutationPages`; search delete freezes page plans before final mutation confirmation | No preflight before broad discovery/planning; delete planning and mutation progress are not exposed as shared frozen counters; machine-mode safety needs regression tests after progress routing | Process-instance mutation rollout |
| Process-instance traversal | `walk process-instance` | Large keyed family traversal plus optional incident/variable/element/listener enrichment | Explicit-key only, so broad selector preflight is not required; output modes are constrained and automation is unsupported | Large family/enrichment work has only command completion output and no frozen traversal/enrichment counters; JSON/stdout safety needs tests if progress is added | Walk/run/smoke rollout |
| Bulk process-instance run | `run process-instance` | Large `--count` with create and wait confirmation | Explicit count is known before work; process-instance bulk services have internal periodic progress logging | No shared preflight/consequence text for large counts; bulk create/wait progress is not routed through `OpsProgressEvent` or command progress-channel gating; keys-only/JSON safety needs assertions after integration | Walk/run/smoke rollout |
| Smoke workflow | `ops execute smoke-test` | Large `--count`, deployment, creation, family walks, cleanup | Has destructive confirmation prompt, audit report, and coarse command activity | No phase-specific progress for deploy/create/walk/cleanup; no shared preflight for large count; no frozen counters for created instances or cleanup scope | Walk/run/smoke rollout |
| Retention workflow | `ops execute retention-policy` | Broad process-instance retention discovery, delete planning, deletion | Service owns discovery/planning/deletion and reports discovery completion/user-limit status; command has coarse activity and final audit report | No shared preflight before discovery/planning; no page progress from discovery; no frozen counters/ETA for delete planning or deletion; prompts happen after full dry-run planning | Ops purge/retention rollout |
| Orphan purge workflow | `ops purge orphan-process-instances` | Broad child discovery plus orphan checks and delete planning | Service-owned orphan discovery already has a progress callback shape; command has coarse activity and audit report | Existing orphan progress is not mapped to `OpsProgressEvent` or command channels; no preflight; no frozen counters for orphan checks, delete planning, or deletion | Ops purge/retention rollout |
| Incident-based purge workflow | `ops purge process-instances-with-incidents` | Broad incident discovery, process-instance dedupe, delete planning, deletion | Service owns incident discovery, frozen candidate set, delete plan, and audit report | No shared incident preflight/page progress; no frozen counters for dedupe/planning/deletion; output-mode tests need to cover progress after integration | Ops purge/retention rollout |
| All-process-definition purge workflow | `ops purge all-process-definitions` | Broad process-definition discovery and affected process-instance checks | Service owns paged process-definition discovery, delete impact planning, version support, and audit report | No shared process-definition preflight/page progress; no frozen counters for impact checks or deletion; 8.9-only support needs explicit tests with progress | Process-definition rollout |
| Incident repair workflow | `ops repair incident` | Broad incident search or many explicit incident keys, variable/job/resolve/confirm steps | Dry-run preflight/confirmation exists before mutation when needed; audit report records step statuses | Preflight is command-specific, not shared scope wording; search page progress and frozen repair counters are missing; keyed bulk repair needs exact counters without broad prompts | Ops repair rollout |
| Process-instance repair workflow | `ops repair process-instance` | Broad incident-bearing process-instance search or many explicit keys, incident repair fan-out | Dry-run preflight/confirmation exists before mutation when needed; audit report records discovery, targets, duplicate/skipped keys, and steps | Process-instance search progress is not shared; frozen target/incident repair counters are missing; direct-incident search needs candidate-vs-frozen wording | Ops repair rollout |

## Audit Notes By Task

### T045 Basic Get Commands

- `get process-instance`, `get incident`, `get job`, and `get element` already keep remote page advancement in facades/services and keep JSON/keys-only stdout clean by collecting before JSON and rendering keys one per line.
- Existing verbose paging uses `printPISearchProgress` or `printSearchPageProgress`, which lacks the shared total-certainty, page-count, and frozen-scope model from `cmd/ops_progress.go`.
- Broad searches that can trigger enrichment should add preflight before expensive per-item enrichment begins, with explicit-key lookups remaining concise.

### T046 Process-Instance Actions

- `cancel process-instance` and `delete process-instance` route search-derived mutation planning through `PlanProcessInstanceMutationPages`, so follow-up work should extend service/facade progress facts instead of reintroducing command-owned paging.
- `delete process-instance` can freeze page-level dry-run plans before one mutation confirmation; `cancel process-instance` currently confirms and mutates per page for search-selected flows.
- `walk process-instance` is explicit-key only, so broad selector preflight is not required, but large family traversal and optional enrichment still need exact phase counters.
- Bulk `run process-instance` has an operator-provided `--count`; follow-up progress should treat that as a known frozen scope and preserve JSON/keys-only output contracts.

### T047 Ops Workflows

- Retention, orphan purge, incident purge, all-process-definition purge, and repair workflows already build durable report models and coarse activity messages.
- Orphan discovery has a service progress callback, but it is not the shared ops progress model and is not routed through command progress-channel gating.
- Purge and repair workflows need candidate-vs-frozen wording because incident discovery, orphan checks, dependency expansion, and force/state checks can all change the final affected scope.

## Rollout Order

1. Basic inspection rollout: make read-only searches use shared preflight/page progress while preserving incremental human and machine output contracts.
2. Process-instance mutation rollout: add destructive preflight and frozen planning/mutation progress around existing service-owned mutation planning.
3. Ops purge/retention rollout: map existing discovery/report scopes to shared progress events and command channels.
4. Ops repair rollout: align command-specific dry-run confirmation with shared preflight and add frozen repair counters.
5. Walk/run/smoke rollout: add exact counters for explicit large work without introducing broad prompts for small keyed work.
