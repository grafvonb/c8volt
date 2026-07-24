# Phase 0 Research: CLI Debt Refactor

## Decision: Start With A Checked-In Command Assessment

**Rationale**: The issue covers all 55 command nodes, but the implementation risk varies by command family. A checked-in assessment creates a stable baseline for paging ownership, output contracts, automation support, progress behavior, mutation behavior, execution style, and high-volume risk before refactoring starts.

**Alternatives considered**:

- Refactor the obvious duplicated files immediately. Rejected because it could miss command-family differences and weaken behavior guarantees.
- Generate an automated static audit framework. Rejected because the issue explicitly excludes a generic static audit framework and the current need is a concrete baseline.

## Decision: Refactor Basic Search Paging From Lower Risk To Higher Risk

**Rationale**: Job and element search have similar paging, limit trimming, continuation, total-count, and incremental rendering behavior. Incident search adds domain-specific filters and process-instance-key output. Process-instance search adds local filtering, enrichment, prompting, and query-strategy concerns. Ordering work in that sequence keeps slices independently testable while learning from smaller surfaces first.

**Alternatives considered**:

- Move all search paging in one large refactor. Rejected because it increases regression risk across multiple output modes.
- Leave job/element duplication alone and focus only on process-instance search. Rejected because shared behavior is clearest in job/element and should shape the service/facade boundary before the hardest command is touched.

## Decision: Backend Mechanics Belong Below `cmd`, Streaming Decisions Stay In `cmd`

**Rationale**: Page traversal, cursor/offset advancement, limit trimming, total fallback, query strategy, local compatibility filtering, frozen discovery, and mutation planning are backend/workflow mechanics. Commands should still own flag validation, confirmation/prompt policy, render mode selection, stdout/stderr rendering, and incremental streaming boundaries because those are CLI behavior.

**Alternatives considered**:

- Move all paging and rendering into internal services. Rejected because services should not own stdout/stderr, prompt, or render-mode behavior.
- Keep all current command paging helpers. Rejected because command-owned backend traversal is the central layering debt.

## Decision: Preserve Domain-Specific Ops Workflows Unless Mechanics Are Identical

**Rationale**: Ops workflows encode different safety and report semantics such as incident discovery, process-definition discovery, orphan detection, frozen candidate sets, duplicate candidate reporting, dry-run reports, and destructive confirmation. Similar helper names are not enough to justify extraction.

**Alternatives considered**:

- Create one generic ops discovery abstraction. Rejected by the specification and by the risk of hiding workflow-specific safety behavior.
- Avoid all ops refactoring. Rejected because the issue requires a selective review for real duplication, layering problems, and slow serial execution.

## Decision: Use Existing Bounded Concurrency Helpers For Independent Work

**Rationale**: `toolx/pool` already provides bounded worker execution with fail-fast behavior and deterministic result slots. Existing service tests cover worker, fail-fast, and no-worker-limit semantics. New performance work should reuse these helpers or local equivalents only when existing helpers do not fit.

**Alternatives considered**:

- Add a new worker-pool package. Rejected because existing `toolx/pool` fits most independent lookups, enrichment, planning, and mutation fan-out.
- Parallelize every page fetch. Rejected because backend pagination is often sequential by cursor/offset and uncontrolled request fan-out would violate operator safety.

## Decision: Define One CLI Progress Policy

**Rationale**: Activity indicators, verbose durable progress, discovery summaries, prompts, and machine-output silence currently differ across commands. A policy allows command families to remain distinct while making the differences intentional and testable.

**Alternatives considered**:

- Standardize every command on one progress format. Rejected because basic read commands and ops workflows have different user-facing semantics.
- Leave progress behavior undocumented. Rejected because output consistency is a primary feature goal and impacts automation safety.

## Decision: Performance Characterization Must Be Targeted And Practical

**Rationale**: The feature must preserve or improve throughput for thousands of resources. Fake-latency tests, benchmark-style tests, and targeted smoke scenarios are appropriate because many regressions come from unnecessary serial calls or lost concurrency rather than CPU-only performance.

**Alternatives considered**:

- Require full production-scale integration benchmarks for every command. Rejected because it would slow delivery and depends on external runtime availability.
- Rely only on unit tests. Rejected because unit tests can miss throughput regressions caused by sequential request patterns.

## Decision: Documentation And Capability Metadata Are Part Of The Contract

**Rationale**: Command help, generated CLI docs, examples, and `capabilities --json` guide operators and automation authors. Any behavior or wording changes around `--batch-size`, `--limit`, output modes, automation, and progress must be validated at the source metadata and generated-doc level.

**Alternatives considered**:

- Update only generated docs after code changes. Rejected because generated docs should follow source command metadata and docsgen tests should protect expected wording.
- Treat capability metadata as secondary. Rejected because it is the machine-readable command contract.
