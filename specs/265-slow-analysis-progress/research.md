# Research: Slow Analysis Progress After Confirmation

## Decision: Pace Durable Human Milestones In Shared Command Progress

**Rationale**: Issue #265 asks for milestone throttling and pacing in shared command progress code, not in `ops analyse slow-process-instances` policy. `cmd/ops_progress.go` already owns progress channel selection, human formatting, ETA gating, and stderr-safe durable output helpers. Keeping milestone pacing there lets slow analysis and future command families reuse the same behavior without moving human rendering into services.

**Alternatives considered**:

- Put the pacing state in `cmd/ops_analyse_slow_process_instances.go`: rejected because it solves only the proof command and conflicts with the issue requirement for shared command progress policy.
- Put durable milestone policy in internal services: rejected because services should emit structured events only and should not know about human modes, stderr, transient activity, or operator-facing wording.
- Add timer-driven output independent of progress events: rejected because it can print "still working" when no observable work has advanced.

## Decision: Default Human Milestones Require Elapsed Silence And Forward Progress

**Rationale**: The clarification answer chose elapsed time plus progress. A milestone should appear only when enough time has passed since the last visible durable milestone and the latest discovery or frozen-scope counter has advanced. This gives operators proof that c8volt is moving while preventing noisy timer-only chatter.

**Alternatives considered**:

- Elapsed time only: rejected because it can print without proof of progress.
- Counter progress only: rejected because fast progress can produce too many durable lines, and slow progress may still be invisible for too long if thresholds are count-only.
- Phase changes only: rejected because a single phase such as runtime element loading can last long enough to look frozen.

## Decision: Preserve Existing Verbose And Debug Durable Detail

**Rationale**: `opsSlowProcessAnalysisDurableProgressAllowed` currently allows durable progress only in verbose and debug modes, while default human mode uses transient workflow activity. This feature should add sparse default human milestones without reducing verbose/debug detail or changing diagnostic expectations.

**Alternatives considered**:

- Route all progress events durably in default human mode: rejected because it would make broad runs noisy and duplicate verbose/debug behavior.
- Remove verbose/debug durable progress after adding default milestones: rejected because diagnostics would regress.

## Decision: Keep Activity Updates On Every Allowed Progress Event

**Rationale**: Transient activity is already how default human mode sees continuous progress in capable terminals, and prior activity-priority work preserves high-level workflow activity over nested HTTP or wait activity. Sparse durable milestones should complement this path for non-spinner or redirected terminals, not replace it.

**Alternatives considered**:

- Durable milestones only, no transient updates: rejected because it would reduce interactive feedback in normal terminals.
- Start a new timer activity scope in slow analysis: rejected because the existing callback already updates workflow activity at the correct importance.

## Decision: Documentation Updates Are Conditional On Help Text Changes

**Rationale**: README and generated docs already describe high-volume progress, mode safety, and slow-analysis progress at a broad level. If implementation changes only internal milestone pacing and tests, no documentation wording may need to change. If command help wording changes, `make docs-content` is required by repository rules.

**Alternatives considered**:

- Always edit docs: rejected because it can create unnecessary churn when shipped behavior remains covered by existing wording.
- Never edit docs: rejected because user-visible help changes must be regenerated and aligned.
