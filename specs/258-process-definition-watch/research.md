# Research: Process Definition Watch Mode

## Decision: Add A Reusable Fixed-Interval Watch Runner

Watch-loop mechanics should live in a new reusable production helper under `toolx/watch`.

**Rationale**: The feature requires immediate first tick, fixed interval sleeps, context cancellation, overall timeout integration, consecutive transient failure handling, retry budget reset after success, and a clean terminal reason. Those mechanics are not process-definition-specific and should be reusable by future read-only commands without copying command-local loops.

**Alternatives considered**:

- Put the loop directly in `cmd/get_processdefinition.go`. Rejected because future watch commands would copy retry and cancellation behavior, increasing drift.
- Reuse `toolx/poller` as-is. Rejected because the existing poller waits for completion with backoff and success predicates, while watch mode must emit every successful snapshot on a fixed cadence.
- Put watch behavior in internal process-definition services. Rejected because interval/retry/cancellation mechanics are generic and should not know about process definitions.

## Decision: Keep Snapshot Collection In The Process-Definition Service/Facade Path

Each process-definition watch tick should call a facade-level snapshot operation that delegates complete page traversal to `internal/services/processdefinition`.

**Rationale**: The repository already has `SearchProcessDefinitionsPages` for service-owned traversal, page continuation, and limit trimming. Watch mode must not reintroduce command-local page loops or call generated Camunda clients from `cmd`.

**Alternatives considered**:

- Reuse the current command helper directly from each tick. Rejected because it mixes progress rendering with snapshot collection and makes human-only output validation harder to reason about.
- Call versioned services or generated clients from `cmd`. Rejected by repository layering rules.
- Use one backend page per snapshot. Rejected because the spec requires complete snapshots, including paged result sets.

## Decision: Treat `--watch` Without Selectors As Broad Discovery

Missing-selector watch should watch all process definitions while one-shot behavior keeps the existing fatal selector diagnostic.

**Rationale**: Clarification selected broad discovery for `get process-definition --watch`. Keeping the behavior watch-only preserves existing non-watch validation while making the shortest watch command useful for observing all deployed definitions.

**Alternatives considered**:

- Emit empty snapshots. Rejected by clarification.
- Keep the one-shot fatal selector error. Rejected by clarification and by the watch use case.

## Decision: Use 1 Second As The Default Watch Interval

When `--watch-interval` is omitted, watch mode should refresh every 1 second after the immediate first snapshot.

**Rationale**: Clarification selected `1s`. This is responsive for deployment visibility while remaining easy to override for lower-load environments.

**Alternatives considered**:

- Default to `2s`. Rejected by clarification.
- Require `--watch-interval`. Rejected because a usable default improves CLI ergonomics.
- Default to `5s`. Rejected because deployment visibility feedback would feel slower than the clarified expectation.

## Decision: Reuse Existing Command Retry Defaults

When no retry flag is provided, watch mode should use the existing command retry default.

**Rationale**: Clarification selected the existing default. This keeps the command aligned with root/config behavior and avoids introducing a parallel retry policy.

**Alternatives considered**:

- Stop after the first transient failure. Rejected because it makes watch brittle during brief Camunda or network blips.
- Retry indefinitely without regard to command defaults. Rejected because it ignores existing operator controls.
- Hard-code three transient failures. Rejected because it creates a second default unrelated to established config.

## Decision: Reject Machine-Oriented Watch Output

Watch mode should reject JSON, keys-only, quiet, automation, and XML output combinations before lookup work begins.

**Rationale**: Watch is a human/operator observation mode. Rejecting machine-oriented combinations preserves c8volt's simple script-safe contracts: `--json` remains one stable JSON document per command invocation, keys-only remains a pipeline format for finite command output, quiet/automation remain deterministic, and XML remains a single artifact.

**Alternatives considered**:

- Emit newline-delimited JSON objects. Rejected because it creates a special `--json --watch` contract and weakens the one-command/one-document expectation.
- Redraw the latest JSON in place. Rejected because terminal redraw is not a script-safe stdout contract.
- Allow keys-only streaming. Rejected because the clarified requirement is human-output-only watch behavior.

## Decision: Keep Human Watch Output Snapshot-Oriented

Default human watch output should render each snapshot as a compact result block, with retry/status details away from result stdout where applicable.

**Rationale**: Existing list output is compact and familiar. Watch mode needs a visible snapshot boundary for humans, and machine-oriented modes are rejected rather than adapted into watch streams.

**Alternatives considered**:

- Redraw an in-place dashboard. Rejected as out of scope and harder to validate in non-interactive terminals.
- Prefix every human result row with a timestamp. Rejected because it would make row scanning noisier.
- Use verbose-only durable progress. Rejected because default human watch still needs understandable snapshot boundaries.

## Decision: Reject `--watch --xml`

Watch mode must reject XML output before lookup work begins.

**Rationale**: XML output is a single artifact selected by key. Repeated XML documents would not form one stable artifact or a clean stream contract.

**Alternatives considered**:

- Emit multiple XML documents separated by newlines. Rejected because consumers cannot treat the result as one XML document.
- Allow only key-based XML watch. Rejected because it still conflicts with the artifact contract.

## Decision: Validation Should Prove Shared Mechanics And Command Contracts

Testing should combine focused `toolx/watch` unit tests, process-definition service/facade tests, command output-mode tests, documentation regeneration, and full repository validation.

**Rationale**: Watch correctness spans generic loop semantics, service-owned paging, facade conversion, command validation, and stdout/stderr contracts. Focused coverage keeps failures easy to localize before running the full suite.

**Alternatives considered**:

- Only add command subprocess tests. Rejected because retry timing and runner behavior would be hard to test deterministically.
- Only test the generic watch helper. Rejected because command flags and output contracts are the user-facing surface.
