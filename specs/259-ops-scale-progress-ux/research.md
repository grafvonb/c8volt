# Research: Ops-Scale Preflight And Progress UX

## Decision: Reuse the existing activity sink/writer as the progress channel

**Rationale**: `toolx/logging.ActivitySink` and `ActivityUpdater` already route transient activity through `cmd.ErrOrStderr()` and are disabled when indicators are unavailable or explicitly disabled. `cmd/root.go` installs this writer into command context and HTTP clients already use it. Extending this path keeps progress out of stdout and keeps tests straightforward with `testx/activitysink`.

**Alternatives considered**: A new stdout progress stream was rejected because it would break JSON and keys-only contracts. A separate global progress writer was rejected because it would duplicate existing command context and terminal capability handling.

## Decision: Keep durable progress wording in `cmd`, not in generated clients or low-level HTTP code

**Rationale**: Operators need phase names such as `discovering process instances` and `loading runtime elements`, not endpoint names. `cmd` already owns render modes, prompts, quiet/automation checks, and stderr output. Services can emit structured progress facts through callbacks or result metadata, while `cmd` translates them into human wording.

**Alternatives considered**: Putting final wording in services was rejected because services do not know render mode or prompt policy. Putting progress in HTTP round trippers was rejected because HTTP request activity is too low-level for this product requirement.

## Decision: Model total certainty generically but map from existing per-resource metadata first

**Rationale**: Process instances, incidents, jobs, elements, and process definitions already preserve `ReportedTotal` and overflow state with exact/lower-bound semantics. A generic preflight model can normalize exact, lower-bound, estimated, and unknown totals while preserving existing domain-specific metadata.

**Alternatives considered**: Adding separate count calls for every command was rejected because it would make startup slower and can duplicate discovery. Treating all counts as exact was rejected because Camunda can report lower bounds or omit usable totals.

## Decision: Use "peek first page and reuse it" as the preferred preflight strategy

**Rationale**: The reported slow behavior starts with repeated process-instance search calls. When the first page includes count and cursor/overflow metadata, preflight can display scope and continue traversal from that same page. This avoids doing a count call followed by a fresh discovery call.

**Alternatives considered**: A preflight-only count request was rejected as the default because it can be as expensive as discovery and does not provide reusable resource rows. Skipping preflight was rejected because the feature is primarily about operator trust at scale.

## Decision: Make `ops analyse slow-process-instances` the first proof workflow

**Rationale**: The workflow already has process-definition discovery, a frozen process-instance set, and per-process-instance runtime element enrichment. That gives the first slice all important phases: preflight, paged discovery, frozen-scope processing, and final render, while remaining read-only.

**Alternatives considered**: Starting with destructive purge/repair workflows was rejected for the first slice because they add confirmation and mutation risk before the shared read-only progress contract is proven.

## Decision: ETA is optional and gated by sample quality

**Rationale**: ETA is valuable for thousands of resources but misleading early in a phase. The plan should introduce elapsed time and exact counters first, then show approximate rate/remaining time only after a minimum completed-work threshold and only when a known or frozen total exists.

**Alternatives considered**: Always showing ETA was rejected because early estimates are noisy. Never showing ETA was rejected because long enrichment phases need planning information for operators.

## Decision: Do not change machine stdout contracts for transient progress

**Rationale**: The repository already treats JSON, keys-only, quiet, and automation as script-safe surfaces. Progress belongs on the activity channel or stderr when allowed, and durable progress should be gated by verbose or explicit progress behavior where needed.

**Alternatives considered**: Embedding transient progress records in JSON results was rejected for this feature because it would change result schemas without helping interactive users. Printing progress to stdout was rejected because it breaks pipelines.

## Decision: Broader rollout should follow command-family slices

**Rationale**: The feature spans basic inspection commands, state-changing process-instance commands, and ops workflows. Slicing by command family keeps each step testable and avoids a risky cross-repo rewrite.

**Alternatives considered**: A single sweeping implementation was rejected because it would touch too many contracts at once. Leaving non-proof commands unplanned was rejected because the product requirement is generic.
