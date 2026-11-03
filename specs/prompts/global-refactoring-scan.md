Perform a complete, read-only refactoring-preparation audit of this repository.

Baseline: v4.4.0
Comparison: baseline tag through the current HEAD, plus inspection of the current working tree.

The objective is to identify concrete opportunities to consolidate, clean up, group, reuse, and optimize code after accumulated feature changes. Complete the whole-codebase scan before reporting conclusions.

Authorization and boundaries:
- Read and follow applicable AGENTS.md instructions.
- Do not edit, create, delete, format, regenerate, or save files.
- Do not create or switch branches, stage changes, commit, push, reset, or revert.
- Do not implement fixes or launch implementation workflows.
- Do not run tests, benchmarks, generators, application commands, or environment-backed validation.
- Do not contact, inspect, provision, or mutate live environments.
- Use read-only repository inspection and in-memory analysis.
- Preserve existing dirty changes.
- Produce the report directly in the conversation.

Repository inventory:
1. Inspect the current branch, HEAD, working-tree status, and baseline tag.
2. Inventory all tracked source files and package/module boundaries.
3. Summarize changes since the baseline by area, separating production code, tests, generated code, scripts, tooling, and documentation.
4. Inspect the entire current codebase, including unchanged code that interacts with changed areas. Do not restrict the audit to the diff or previously known packages.
5. Identify generated, vendored, historical, and runtime artifacts. Inspect their ownership and generation boundaries; direct refactoring recommendations toward maintained sources.
6. Respect repository rules that distinguish feature artifacts, reusable prompts, and integration harness material.

Mandatory scan coverage:
- CLI construction, flags, validation, dispatch, output, errors, prompts, and execution modes.
- Public facades, models, converters, options, and compatibility surfaces.
- Domain models and reusable invariants.
- Internal service orchestration and version-specific adapters.
- HTTP transport, authentication, retries, pacing, polling, logging, and diagnostic redaction.
- Shared production helpers and test support.
- Unit, contract, subprocess, terminal, and integration test source.
- Configuration and default/precedence handling.
- API generation and mutation tooling.
- Build, release, documentation, CI, and repository scripts.

Analysis:
Use structural searches, import/dependency analysis, symbol-reference searches, and exact or normalized duplicate detection to locate candidates. Manually inspect candidates before reporting them.

A structural match is evidence for investigation, not proof that code should be merged. A symbol with no repository callers is not automatically safe to remove from a public API.

Specifically investigate:
1. Repeated JSON normalization, comparison, decoding, copying, and snapshotting.
   Check whether implementations differ in numeric precision, typed collections,
   custom marshalers, invalid/cyclic values, pointer ownership, or nil/empty behavior.

2. Multiple bulk execution abstractions, unused worker implementations, and dependency
   direction. Trace indirect dependencies through shared helpers as well as direct imports.

3. Divergent command error/exit paths.
   Trace representative failures through service, facade, rendering, logger, and exit.
   Check compact human reasons, retained machine evidence, partial results, original
   exit classes, and exactly-once envelope rendering. Calling a shared helper alone
   does not establish that its fallback behavior is correct.

4. Repeated collection searches and expensive local work.
   Look for nested scans, repeated association lookup, repeated normalization or
   serialization, unnecessary allocation, and redundant copying. Explain complexity
   using actual call paths; do not claim measured gains without benchmarks.

5. Similar lifecycle workflows such as suspend/resume, planning, submission evidence,
   confirmation, cancellation, and bounded observation.
   Separate reusable mechanics from action-specific semantics.

6. Repeated command summaries, outcome counting, bounded failure aggregation,
   reason ordering, and verbose formatting.

7. Version-independent logic duplicated inside version-specific adapters.
   Distinguish shared raw response decoding and traversal from genuinely different
   endpoint schemas, generated types, capability rules, and compatibility behavior.

8. Backend query strategy, fallback discovery, dependency expansion, or cross-resource
   workflows still owned by commands or facades.

9. Repeated deployment/start, repair, cleanup, selection, and frozen-plan preparation.
   Check existing internal owners before proposing new abstractions.

10. Repeated cross-version or cross-command test contracts and fixtures.
    Identify shared harness opportunities while retaining version-specific wire checks
    and independently meaningful acceptance coverage.

11. File cohesion and package organization.
    Identify files combining distinct planning, execution, lifecycle, model, or conversion
    responsibilities. File length alone is insufficient justification for splitting.

12. Dead scaffolding, duplicate configuration helpers, obsolete wrappers, and repeated
    build/release/documentation workflows.

Evidence requirements:
- Confirm each finding against the current implementation.
- Check whether it was introduced, extended, or already present at the baseline.
- Provide precise file/function references with verified line numbers.
- Explain the actual duplication, ownership problem, behavioral drift, or complexity.
- Identify existing helpers or services that could be reused.
- State the proposed owner and smallest cohesive refactoring boundary.
- Explain what behavior must remain distinct.
- Identify public API and CLI compatibility constraints.
- Separate confirmed findings from risks requiring isolated reproduction.
- Do not repeat findings already resolved in the current code.
- Do not impose a fixed finding count.
- Avoid speculative frameworks, cosmetic micro-abstractions, or universal helpers
  combining operations with different semantics.

Report:
Start with the scope, methods, baseline, and coverage limitations.

Provide prioritized findings. For each include:
- Finding and practical consequence.
- Evidence and source references.
- Origin relative to the baseline.
- Recommended consolidation, cleanup, grouping, reuse, or optimization.
- Dependencies on other findings.
- Compatibility constraints and implementation risk.
- Suggested isolated validation, without executing it.

Order recommendations by dependencies and practical value. Group tightly coupled work
into cohesive implementation outcomes, while keeping independently verifiable changes
separate. Include necessary fixtures and validation within each proposed outcome.

Explain intentional duplication that should remain, including generated mappings,
endpoint-specific contracts, different confirmation rules, and compatibility shims.

Conclude with:
- Recommended first implementation sequence.
- Any unresolved questions or reproduction needs.
- Confirmation that no files were changed or saved and no tests, live validation,
  implementation, Git mutations, or commits were performed.
