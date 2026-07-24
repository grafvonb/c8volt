# Contract: CLI Debt Refactor

This contract defines the user-facing and maintainer-facing behavior that implementation tasks must preserve or introduce. It complements command-specific tests and generated command documentation.

## Assessment Contract

The implementation must create a checked-in assessment artifact that covers all 55 command nodes.

Each command record must include:

- Command path and aliases
- Command family
- Mutation classification
- Contract support level
- Automation support level
- Supported output modes
- Paging behavior
- Mutation behavior
- Activity indicator behavior
- Durable progress behavior
- JSON and keys-only constraints
- Ownership of paging, discovery, query strategy, rendering, confirmation, and mutation planning
- Serial or bounded-concurrent execution style
- Performance risk for thousands of resources

The assessment must also identify:

- Basic read commands with duplicated paging mechanics
- Process-instance mutation commands with command-owned paging or planning mechanics
- Ops workflows where similar code is intentionally retained because semantics differ
- High-volume workflows requiring performance characterization

## CLI Output Contract

Changed commands must continue to satisfy these output rules:

- Human output remains compact and scan-friendly.
- Verbose progress is durable and written away from machine stdout.
- Activity indicators are transient and use the shared activity writer.
- JSON output emits one valid machine-readable document.
- Keys-only output emits one key per line and no additional text.
- Quiet mode suppresses nonessential human output.
- Automation mode does not prompt unexpectedly and does not emit incidental progress into machine output.

## Paging And Limit Contract

Changed paged workflows must satisfy these semantics:

- `--batch-size` means per-page request size.
- `--limit` means total user cap or frozen-scope cap, depending on command family.
- Page traversal handles zero results, one page, multiple pages, final partial pages, backend total metadata, and local result filtering.
- Cursor and offset advancement remain version-aware and do not skip or duplicate visible items.
- Warning-stop and partial-complete states are distinct from complete discovery.

## Progress Policy Contract

Activity indicators:

- Used for long-running opaque work.
- Disabled by no-indicator, quiet, automation, and JSON log constraints.
- Written only through the shared activity writer.

Verbose durable progress:

- Used for paged searches and long-running mutations when verbose output is enabled.
- Includes page size, page count, cumulative count, more-data state, and next action where applicable.
- Includes worker progress when concurrent work materially affects operator understanding.

Ops discovery summaries:

- Distinguish complete discovery from user-limited discovery.
- Include batch size, limit, pages, candidates seen, and candidates frozen when relevant.
- Keep default human output compact while preserving full detail in JSON, report, or verbose output.

Machine output:

- JSON and keys-only output must remain parseable and free from progress noise.
- Automation must not prompt unexpectedly.

## Ownership Contract

The implementation must preserve these ownership boundaries:

- `cmd`: flags, validation, command metadata, prompt and confirmation policy, render-mode selection, stdout/stderr rendering, help examples, and command contract annotations.
- Public facades under `c8volt/<area>`: mapping public inputs to service inputs, delegating to internal services, mapping outputs back, and converting errors through public facade error conversion.
- Internal services: paging loops, cursor or offset advancement, limit trimming, query strategy, local compatibility filtering, frozen discovery, mutation planning, polling, retries, worker pools, and version-neutral workflow behavior.
- Version-specific services: Camunda version-specific request and response differences.
- Generated clients: no manual refactoring unless a later explicit client-generation task requires it.

## Performance Contract

Changed high-volume workflows must:

- Characterize current and changed behavior with fake-latency tests, benchmark-style tests, targeted smoke scenarios, or documented manual validation.
- Use bounded concurrency for independent lookups, enrichment, planning, confirmation checks, and bulk operations when it improves throughput safely.
- Preserve operator controls for workers, batch size, limits, fail-fast behavior, and worker-limit overrides.
- Avoid uncontrolled request fan-out against Camunda APIs.
- Preserve deterministic output order where users or machine consumers rely on it.

## Documentation And Metadata Contract

When behavior, flags, examples, aliases, output contracts, or help text change:

- Update source command metadata and help text.
- Update command contract tests.
- Regenerate generated CLI docs with `make docs-content`.
- Update README or docs examples if user-facing behavior is described there.
- Verify `capabilities --json` reports automation and output-contract support accurately.

## Compatibility Contract

Existing user-visible behavior is the baseline. Any intentional behavior change must include:

- A compatibility note in the implementation plan or task artifact.
- Targeted tests covering old and new expectations where practical.
- Documentation updates in the same unit of work.
- Explicit confirmation that JSON, keys-only, quiet, automation, prompt, and progress behavior remain safe.
