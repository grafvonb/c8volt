# Research: Volume And Semantic CLI Integration Coverage

## Decision: Keep Volume Targets Separate From Baseline Family Targets

**Decision**: Add dedicated volume Make targets and Go tests instead of extending the existing baseline family targets.

**Rationale**: Baseline family targets should remain quick, independently runnable, and useful for all-command regression checks. Volume tests need larger seeded datasets, longer timeouts, and broader destructive behavior, so they should be opt-in.

**Alternatives considered**: Folding volume scenarios into `integration-cli-get`, `integration-cli-update`, and other baseline targets was rejected because it would make routine checks slower and more destructive. Creating a separate external script suite was rejected because the existing Go harness already builds the CLI once, captures evidence, and enforces default local config rules.

## Decision: Use Configurable Conservative Dataset Sizes

**Decision**: Default volume datasets should be conservative and overridable by environment, for example with a volume-size variable used by all volume targets.

**Rationale**: The suite must prove paging and multi-target behavior while remaining practical on local disposable clusters. Operators can increase dataset size when they need deeper stress evidence.

**Alternatives considered**: Fixed large datasets were rejected because local Camunda clusters vary in capacity. Tiny fixed datasets were rejected because they cannot prove page, limit, worker, or progress semantics.

## Decision: Prove Semantics Through Observable CLI Outcomes

**Decision**: Validate critical flags through observable CLI behavior and follow-up reads, not by inspecting internal services.

**Rationale**: c8volt's public promise is CLI behavior for people and pipelines. Dry-run safety, no-wait wording, limit behavior, keys-only cleanliness, and report creation matter at the command boundary.

**Alternatives considered**: Reusing internal service tests was rejected because they do not prove root flags, stdout/stderr separation, prompts, stdin, profile selection, or report file behavior.

## Decision: Prefer Robust Worker Assertions Over Timing-Sensitive Concurrency Tests

**Decision**: Worker and no-worker-limit tests should assert accepted configuration, stable reporting, affected key accounting, and successful/partial outcomes over multi-key data without relying on exact runtime timing.

**Rationale**: Timing-sensitive integration tests become flaky on shared developer machines and Camunda clusters. The value is to prove the operator-visible semantics and batch accounting.

**Alternatives considered**: Comparing wall-clock duration between one worker and many workers was rejected because external cluster scheduling makes such checks unstable.

## Decision: Treat Stdin Pipelines As First-Class Volume Scenarios

**Decision**: Add explicit pipeline scenarios where keys-only producer output feeds stdin-consuming commands in dry-run and confirmed modes.

**Rationale**: c8volt is for people and pipelines. Keys-only output must remain clean under volume, and stdin consumers must handle empty, duplicate, malformed, missing, and valid keys predictably.

**Alternatives considered**: Continuing to record pipeline examples as blocked help examples was rejected because it does not prove the user-facing automation workflow.

## Decision: Capture Human Progress Without Depending On Terminal Animation

**Decision**: Prove progress using verbose/durable output and harness logs, and use transient indicator checks only where the harness can reliably allocate a terminal-like environment.

**Rationale**: The activity writer intentionally suppresses indicators in non-interactive and machine modes. Durable progress facts are more stable than spinner animation frames.

**Alternatives considered**: Requiring raw spinner frame assertions in every long-running command was rejected as brittle and platform-dependent.

## Decision: Validate Ops Reports As Product Artifacts

**Decision**: Volume ops scenarios must parse JSON reports and inspect Markdown reports for stable sections, workflow identity, discovery completeness, step status vocabulary, accounting, notices, errors, and final outcome.

**Rationale**: Ops commands are audited playbooks. Their reports are not incidental files; they are evidence that the command discovered, froze, planned, executed, and completed the workflow correctly.

**Alternatives considered**: Only checking that a report file exists was rejected because it would miss broken report semantics and stale or partial accounting.

## Decision: Preserve Proposal Reporting For Setup Gaps

**Decision**: When volume scenarios require data that current c8volt commands or embedded BPMN fixtures cannot create, record runtime truth in the test run and track missing setup or fixture needs in spec-owned artifacts.

**Current status**: This 256-era decision is deprecated by `specs/integration-test-responsibility.md`. Future volume or real-state integration work should record runtime truth in tests and maintain missing setup or fixture needs in spec-owned gap artifacts.

**Rationale**: The suite should reveal product gaps without silently skipping coverage or requiring hidden direct setup forever.

**Alternatives considered**: Expanding setup through direct Camunda APIs without spec-owned gap tracking was rejected because it hides useful future c8volt command opportunities.
