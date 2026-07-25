# Research: C89 Real-State Semantic Integration Coverage

## Decision: Add A Real-State Layer Separate From Baseline And Volume Targets

**Decision**: Introduce `real-state` integration targets rather than extending the 255 baseline or 256 volume targets.

**Rationale**: Baseline targets prove command breadth and representative behavior. Volume targets prove paging, progress, reports, and pipeline semantics. The remaining gaps require specific Camunda runtime state and more destructive assertions, so they need a focused layer that can be run intentionally.

**Alternatives considered**: Folding real-state assertions into volume targets was rejected because it would make already-slower targets harder to reason about. Creating external scripts was rejected because the Go harness already captures command output, evidence files, profile selection, and proposal records.

## Decision: Focus On Camunda 8.9 First

**Decision**: Implement the foundation against Camunda 8.9 profiles selected from the default local config, while keeping observed and affected version fields in evidence.

**Rationale**: The current goal is a strong 8.9 foundation. Camunda minor releases happen regularly, so scenario names, proposal records, and evidence must remain version-aware, but expanding this feature into a broad version matrix would delay the real-state foundation.

**Alternatives considered**: Building simultaneous 8.7/8.8/8.9/8.10 coverage was rejected for this feature because it would overload scope. Hard-coding 8.9 assumptions without version evidence was rejected because it would make future minor-release extension expensive.

## Decision: Prefer C8volt Commands For Setup, With Proposal-Recorded API Fallback

**Decision**: Real-state tests should create data through c8volt commands when possible. If a state cannot be created through c8volt, direct Camunda API setup is allowed only when paired with command-extension proposal evidence.

**Rationale**: Integration tests should validate the product as operators use it. Direct API setup can be practical for rare runtime states, but it should reveal a missing c8volt capability rather than become hidden permanent scaffolding.

**Alternatives considered**: Direct API setup for all fixtures was rejected because it avoids testing c8volt setup commands. Blocking all API setup was rejected because some job, listener, retention, or partial-failure states may not yet be reachable through c8volt.

## Decision: Prefer Embedded BPMN Fixtures, Then Propose New Fixtures

**Decision**: Use existing embedded BPMN models first. Do not mutate existing embedded fixtures to force new behavior; record embedded BPMN proposals for missing listener, BPMN error, repair, retention, or partial-failure models.

**Rationale**: Existing fixtures are shared assumptions for other tests. New behavior should be explicit and versioned so later releases can extend it safely.

**Alternatives considered**: Editing existing embedded fixtures was rejected because it can change unrelated test behavior. Importing undocumented one-off BPMN without proposal records was rejected because the missing reusable fixture would remain invisible.

## Decision: Prove Real State With Before And After Evidence

**Decision**: Every real-state mutation scenario must capture enough before-state and after-state evidence to prove the command outcome, or explicitly classify the outcome as submitted, no-wait, retained, cleanup-failed, skipped, unsupported, or proposed.

**Rationale**: This matches c8volt's "done is done" standard. It is not enough to prove that a command accepted flags or returned success when the cluster state did not change or could not be observed.

**Alternatives considered**: Trusting command exit codes alone was rejected because it misses accepted-but-not-complete, partial failure, and report/accounting regressions.

## Decision: Make Dirty-Cluster Tolerance A First-Class Assertion

**Decision**: Assertions must identify suite-owned resources and use containment, scoped selectors, and recorded keys rather than exact global counts.

**Rationale**: The selected cluster may be clean, dirty, or left over from a previous run. The suite must remain useful in all those states, and the user explicitly allows mutation against the chosen disposable cluster.

**Alternatives considered**: Requiring cleanup before every run was rejected because it does not reflect real operator validation. Exact global count assertions were rejected because dirty clusters make them brittle.

## Decision: Treat Proposal Aggregation As A Testable Contract

**Decision**: The aggregate proposal reports must include every known command-family and embedded BPMN setup gap, including gaps discovered by the real-state layer.

**Rationale**: Proposal evidence is the bridge between tests and future product/fixture work. If a family writes a proposal only in its own slice but the aggregate misses it, maintainers lose the map of remaining live-state gaps.

**Alternatives considered**: Keeping proposals only in per-family reports was rejected because it makes cross-suite planning and release review harder.
