# Implementation Plan: Suspend Process Instances

**Branch**: `346-suspend-process-instances` | **Date**: 2026-10-25 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/346-suspend-process-instances/spec.md`, derived from issue #346.

## Summary

Add `suspend process-instance` / `suspend pi` for explicit single/multiple keys and stdin. Preview exact targets locally, preserve the existing confirmation/automation contract, submit through the existing Camunda 8.10 single-instance endpoint, and default to bounded observation of SUSPENDED. No-wait reports acceptance without observation. Parent and child selections stay independent; no hierarchy lookup, root escalation, search selection, or relative mutation occurs.

Use a dedicated suspension result that separates submission evidence from observed state and confirmation. Keep collection execution, retry evidence, and waiting in internal services; public facades only map and delegate. Reuse the existing worker pool, backoff configuration, retry policy, error conversion, output envelopes, and command logger. A focused suspension waiter avoids changing cancellation's absence/terminal semantics. Update source help, README, generated CLI documentation, and the existing Bash suspension setup fixture without executing live tests.

## Technical Context

**Language/Version**: Go 1.26.0, repository toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2, pflag 1.0.10, Viper 1.21.0, generated v810 Camunda client, slog logging, repository `toolx/pool`, retry/backoff helpers, and testify 1.11.1. No new dependencies.

**Storage**: No new persistent storage. Per-call selected keys, outcome reports, and observation evidence only; existing process-instance state/date mapping remains unchanged.

**Testing**: Isolated unit and generated-client stub tests, test-owned HTTP servers, facade stubs, CLI subprocess tests, and real-terminal stdin tests using existing `testx` helpers. Planning-only validation is documentation review, path/link checks, and whitespace checks.

**Target Platform**: Existing Go CLI/library platforms. Real-terminal coverage uses the existing macOS/Linux test support; unsupported platform skips must be reported accurately.

**Project Type**: CLI and public Go facade library with version-neutral contracts and Camunda version adapters.

**Performance Goals**: O(n) selection/result storage for n unique supplied keys; existing bounded worker default or explicit worker controls; no hierarchy/discovery requests. One logical suspension per unique target, subject only to existing submission retry policy. Confirmation is bounded per target by configured timeout/max attempts and caller context; no-wait performs zero confirmation reads.

**Constraints**: Camunda 8.10 only; 8.8/8.9 guards before unsupported I/O even in preview. Preserve rejected/uncertain/accepted-but-unconfirmed evidence, original failure classes/exits, and one JSON envelope. No mutation from polling; no raw HTTP details in normal/verbose human output. No live-environment implementation validation.

**Scale/Scope**: One new verb and one child command with alias, two public methods, additive domain/public reports, one service-owned collection path, one supported adapter and explicit unsupported adapters, and a focused waiter. Native batch, search selection, resumption, definition lifecycle changes, cancellation/deletion behavior changes, and client regeneration are excluded.

## Constitution Check

The initial gate passed before design: the requested behavior fits the existing CLI/facade/service architecture, requires no constitutional exception, and can be validated without a running environment. Post-design review confirms the same gates below.

| Principle/gate | Initial result | Post-design evidence |
| --- | --- | --- |
| I. Operational proof over intent | Pass | Confirm only after valid 204 plus matching SUSPENDED observation. Explicit no-wait reports acceptance; rejected/unknown submissions never reconcile to success. |
| II. CLI-first, script-safe interfaces | Pass | Stable explicit-key grammar, shared envelope once, documented keys-only acceptance, quiet/mode precedence, configured stdout/stderr/logger separation, original exit classes. |
| III. Validation proportional to change | Pass | Current work uses document checks only. Implementation starts with targeted tests, then one full race suite justified by multi-package interface changes and concurrent collection behavior. No live gates. |
| IV. Documentation matches behavior | Pass | CLI source help/metadata, README, and `make docs-content` accompany command implementation. Fixture change is statically checked. |
| V. Small compatible repository-native changes | Pass | Existing packages and helpers; dedicated result and focused waiter justified by missing evidence/semantics in generic helpers. No new dependency/framework or broad cleanup. |
| Architecture ownership | Pass | Collection, retries, and observation stay internal; adapters own wire mapping/version guards; facades only map/delegate; commands validate, prompt, render, and exit. |
| Diagnostic/output rules | Pass | Failure trace and execution-path logger/PTY matrix in contracts and quickstart; no raw joined-error normal output or duplicate error renderer. |
| Cohesive implementation units | Pass | Interface changes include adapters/test doubles; single and collection behavior share one implementation; each work unit includes relevant fixtures/tests/formatting/checks. |

**Compatibility note**: Adding methods to public `process.API` requires external custom implementations of that interface to implement them. Existing library callers and existing command behavior remain unchanged. All repository implementations/stubs must be updated in the same compiling unit. New result types do not alter existing JSON schemas. Human suspension result summaries intentionally use stdout as selected-format output; nearby INFO-logger workflow summaries are not copied as result writers.

## Project Structure

### Documentation (this feature)

```text
specs/346-suspend-process-instances/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    ├── cli.md
    └── facade-service.md
```

`tasks.md` is generated by the subsequent `$speckit-tasks` phase, not by this planning command.

### Source Code (repository root)

```text
cmd/
├── suspend.go                                  # new verb and backoff registration
├── suspend_processinstance.go                  # new command, flags, validation, metadata
├── suspend_processinstance_batch.go            # preview/confirmation/result/exit lifecycle
├── cmd_views_processinstance_suspend.go        # new mode-aware views and safe summaries
├── command_contract_test.go                    # metadata/capability coverage
└── suspend_processinstance*_test.go            # execution, diagnostics, output, terminal tests
c8volt/process/
├── api.go, model.go, convert.go                # additive public contract and mappings
├── suspend.go                                  # thin facade methods
└── suspend_test.go                             # facade mapping and partial-error evidence
internal/domain/
└── processinstance_suspension.go                # new evidence/result enums and structs
internal/services/processinstance/
├── api.go                                      # additive no-I/O guard and single method
├── suspend.go                                  # collection ownership, keyed result slots
├── suspend_test.go
├── waiter/suspension.go                        # exact suspension observation semantics
├── waiter/suspension_test.go
├── v88/contract.go, v88/suspend.go               # explicit unsupported capability
├── v89/contract.go, v89/suspend.go               # explicit unsupported capability
├── v810/contract.go, v810/suspend.go             # generated endpoint and operation composition
├── v810/suspension_submission.go                # private retry/submission evidence helper
└── v810/suspend_test.go                         # response/retry/confirmation mapping
README.md
docs/cli/, docs/index.md                        # generated by docs-content
integration/scripts/run-c810-suspended.sh       # suspension setup only; static validation
```

**Structure Decision**: Extend existing process packages. New files above are planned; existing factory selection remains sufficient. Update concrete test doubles wherever compilation requires, including process/resource facade tests, process-instance dry-run tests, process-definition deletion tests, ops orphan-purge tests, and v810 client mocks. Do not edit generated Camunda clients. Reuse common helpers before adding anything shared; suspension-only logic remains in its owning area/adapter. The CLI lifecycle file is focused on control and output, never backend mechanics.

## Phase 0: Resolved Research

[research.md](research.md) records nine decisions, source evidence, rationale, and alternatives. All technical questions are resolved:

1. Existing v810 generated support is sufficient; HTTP 204 is the acceptance contract.
2. A dedicated local suspension guard covers public-library and preview calls on unsupported versions.
3. Dry-run previews immutable explicit keys with no state reads. No-wait does not require an advisory read permission.
4. Dedicated reports preserve independent submission, confirmation, last-state presence, date, response, and error facts.
5. Cumulative evidence survives temporary-network retries, backoff cancellation, and locally unsent pacing failures.
6. A focused waiter preserves no-observation versus observed UNKNOWN and uses typed absence rather than error substrings.
7. Service-owned collections fill skipped slots and retain context errors rather than trusting pool zero values.
8. Existing result/error helpers provide one partial-result envelope; human summaries and diagnostic messages have explicitly separate destinations.
9. Isolated execution, logger, and real-terminal checks supply the validation boundary; fixture updates do not authorize live runs.

## Phase 1: Design

[Data model](data-model.md) defines fields, workflow/submission/confirmation states, collection ordering, and invariants. [Facade/service contract](contracts/facade-service.md) defines additive signatures, ownership, retry/observation rules, and compatibility. [CLI contract](contracts/cli.md) fixes input grammar, flags, previews, prompt policy, output modes, error flow, documentation, and fixture scope. [Quickstart](quickstart.md) provides runnable isolated validation steps and required assertions.

Execution sequence: local CLI validation → service capability/input validation and no-I/O preview → dry-run result or confirmation → exact-key collection execution → authoritative submission response → optional bounded direct-key reads → one mode-aware result path → original error exit. The preview is not proof of eligibility; only the backend decides acceptance. Only accepted submission followed by SUSPENDED confirms the requested outcome.

The waiter stops unsuccessfully on a true engine-terminal observation, fails on non-absence read errors, and preserves evidence under timeout/interruption. ABSENT and UNKNOWN never confirm or count as true terminal suspension outcomes. Existing generic wait/cancel/delete behavior is untouched.

## Requirement Coverage and Validation Ownership

| Requirements | Design/validation owner |
| --- | --- |
| FR-001–FR-002 | CLI input helpers plus service normalization; flag/stdin/dedup/order and invalid-input command tests. |
| FR-003 | Adapter no-I/O guards; service and public-facade single/collection/preview tests, CLI configured-version tests. |
| FR-004–FR-006 | Exact-key v810 submission plus retained retry evidence; parent/child, rejection, race, malformed-response and uncertainty request-count tests. |
| FR-007–FR-009 | Focused waiter and no-wait branch; all observed states, identity, dates, timeout, interruption, permission denial, and no-resubmission tests. |
| FR-010–FR-012 | Preview/frozen scope/confirmation plus collection reports; dry-run, real-terminal, automation, fail-fast and partial-result tests. |
| FR-013–FR-016 | Mode-aware views, safe configured logger, existing exits; one-envelope EOF, exact streams, quiet, keys-only, empty-input and diagnostic regression matrix. |
| FR-017 | Existing process-instance mapping plus get/expect regression checks with isolated state responses. |
| FR-018–FR-020 | Source help/metadata, README, docs generator, static fixture rewrite and syntax check. |
| FR-021 | Scope exclusions and existing cancellation/forced-deletion regressions; no changes to their waiter/tree semantics. |

Every SC-001–SC-008 measurable outcome is exercised by the corresponding execution or documentation checks in the quickstart. Planning artifacts describe intended acceptance; they do not claim implementation tests have passed.

## Delivery and Task-Generation Guidance

Generate cohesive work units in this dependency order, including implementation, necessary fixtures, meaningful tests, formatting, and targeted checks in each:

1. **Internal submission evidence foundation**: domain evidence/result types, the v810 narrow generated-client signature/mock update, and a private suspension submission helper with exact-key mapping, 204 acceptance, rejection/uncertainty, retry and pacing evidence tests. This is a compilable internal foundation with no new public suspension operation yet. Keep adapter-specific mechanics in `v810/suspension_submission.go`; do not extract a generic mutation framework.
2. **Complete suspension service and facade workflow**: all service/public interface additions, all version adapters and affected test doubles, capability guards, preview, focused waiter, service-owned collections, thin facades, and corresponding single/collection/version/observation/partial-result tests. Expose the public operation only with its complete default waiting behavior. Keep the waiter and accepted-submission composition together; Story 1 and Story 2 share this safety invariant and must not cause repeated implementation.
3. **CLI and operator contract with fixture adoption**: verb/alias/flags/metadata, frozen preview and confirmation lifecycle, all output modes, one-envelope partial failures, safe aggregate diagnostics, real-terminal tests, help/README/generated docs, and the small Bash suspension-setup rewrite with syntax validation. Keep failure-path validation with the code that introduces it. Run remaining affected regressions and the justified full race check after integrating the complete feature; these are validation responsibilities, not standalone implementation tasks.

These are sequencing constraints, not a generated task checklist. Adjacent tasks must be reviewed for consolidation; record why tightly coupled work is separated. More than ten tasks triggers review. Context reading, routine branch checks, progress tracking, and commits are responsibilities within work units, not standalone tasks. Respect Ralph's story scope; group shared safety mechanics by cohesive outcome instead of repeating them for separate stories.

The completed implementation warrants one full race suite after targeted checks because concurrent collection behavior and expanded service/facade interfaces affect multiple consumers. Do not run it for this planning-only change, rerun merely to commit, or add live validation/cleanup gates. Feature commits must use Conventional Commits with subjects ending in ` #346`.

## Complexity Tracking

No constitutional exception is required. Dedicated suspension results are necessary because generic reporters lose acceptance/observation evidence. A focused waiter is necessary because existing generic waiting treats absence and observation loss differently; changing that global behavior risks cancellation/expect compatibility. Both additions remain local to existing packages. Main risks are lost partial results, false acceptance after ambiguous retries, false terminal confirmation, pool-skipped slots, logger bypass, duplicate JSON, and wider interface compile failures; each has explicit isolated coverage.
