# Implementation Plan: Process-Definition Lifecycle State

**Branch**: `348-definition-lifecycle-state` | **Date**: 2026-10-27 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/348-definition-lifecycle-state/spec.md`

## Summary

Expose the definition lifecycle already supplied by Camunda 8.10 through existing definition reads, public results, human/JSON output, statistics views, and watch changes. Preserve older reads and unavailable-state omission. Add a string field to the existing models, map it through the shared converters, and extend the existing definition display/watch identity. Do not add requests, interfaces, state machines, generated-client changes, or deletion behavior.

The touched watch code has three existing logger bypasses. Correct these locally while preserving message meaning, ordering, and write-error/baseline behavior; this is required by FR-015 and repository logging rules, not a general logging refactor.

## Technical Context

**Language/Version**: Go 1.26.0; declared toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra v1.10.2, Viper v1.21.0, generated Camunda clients, standard JSON/slog support, existing flat rendering/logging helpers. No new dependencies.

**Storage**: No persistence changes. Definition values and watch baselines are in-memory observations.

**Testing**: Go tests, testify v1.11.1, existing generated-client mocks, test-owned HTTP fakes and counters, command/subprocess harnesses, real PTY tests with fake service dependencies. Integration files may be compiled but not run during implementation.

**Target Platform**: Existing supported CLI/library platforms, including macOS and Linux terminal behavior. Preserve existing platform skips for PTY support.

**Project Type**: Go CLI and public Go facade library.

**Performance Goals**: Zero additional backend requests, prompts, or mutations for lifecycle display; preserve existing pagination, statistics work, watch cadence, retries, and row ordering.

**Constraints**: Lifecycle availability from 8.10; current 8.8/8.9 reads omit state. Preserve unknown values and omission. No state inference, extra fetch, default ACTIVE, synthetic DELETING, deletion-policy change, new no-wait coverage, or live implementation validation.

**Scale/Scope**: One optional result attribute; the existing lookup/listing/statistics/watch surfaces; one existing draining integration observation; command help and generated docs. No new execution mode or polling mechanism.

## Constitution Check

*Gate evaluated before research and re-evaluated after Phase 1 design.*

| Principle | Before research | After design |
| --- | --- | --- |
| I. Operational proof | Pass: display supplied evidence, never infer completion from counts. | Pass: missing state and DELETED/absence remain distinct; no mutation or deletion-completion changes. |
| II. CLI-first/script-safe | Pass: existing command/flag/output contracts are retained with one additive optional field. | Pass: omission, one-envelope/EOF, empty keys, quiet, XML, and watch incompatibilities are explicit in contracts. |
| III. Proportional validation | Pass: planning uses document checks; implementation uses isolated fixtures. | Pass: targeted read/render/watch checks and compile-only integration validation; broader checks conditional on actual shared impact/failures. |
| IV. Documentation matches behavior | Pass: help, metadata, README, and regenerated CLI docs are required. | Pass: lifecycle meanings, 8.10 support, logger configuration, and backend-versus-force deletion are covered. |
| V. Small/repository-native changes | Pass: existing generated result and conversion/display ownership are sufficient. | Pass: one string field, existing flat columns/watch baseline, and a local summary helper; no new abstraction or dependency. |

Repository gates also pass by design: command → facade → service layering is preserved; version-specific mapping stays in v810; generated clients are untouched; result output remains stdout; affected diagnostics use the configured logger; summary write errors still prevent baseline commit. Current logger bypasses are explicitly resolved rather than accepted as precedent. No constitution exception or unresolved clarification remains.

## Project Structure

### Documentation (this feature)

```text
specs/348-definition-lifecycle-state/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    ├── cli.md
    └── library.md
```

`tasks.md` belongs to the subsequent `speckit-tasks` phase and is not created here.

### Source Code (repository root)

```text
internal/domain/processdefinition.go                 # Optional definition State
internal/services/processdefinition/
├── v810/convert.go                                  # Generated state → domain
├── v810/service_test.go                             # Lookup/search/stat propagation
├── v88/service_test.go, v89/service_test.go          # Older-read omission
└── search_test.go                                   # Paged/latest/watch preservation
c8volt/process/
├── model.go, convert.go                             # Optional public State and mapping
└── convert_test.go, client_test.go, model_test.go   # Conversion/delegation/serialization
cmd/
├── cmd_views_processdefinition.go                   # Shared state cell/change styling/summary
├── get_processdefinition_watch.go                   # Existing warning/timing logger calls
├── get_processdefinition.go                         # Help and output metadata
├── cmd_views_processdefinition_test.go              # Layout and comparison fixtures
├── get_processdefinition_test.go                    # Existing read/mode regressions
├── get_processdefinition_suspended_test.go           # Fake-backed state/count/request cases
├── get_processdefinition_watch_test.go              # Refresh/error/baseline behavior
├── get_processdefinition_watch_terminal_test.go     # Real PTY + fake backend styling
└── get_processdefinition_watch_timeout_test.go      # Configured logger execution patterns
integration/cli/real_state_destructive_test.go       # Existing draining CLI observation only
README.md, integration/README.md                     # Operator/fixture descriptions as needed
docs/cli/*, docs/index.md                            # Regenerate through make docs-content
```

**Structure Decision**: Extend existing owners. `internal/services/processdefinition/search.go`, v88/v89 converters, the shared flat renderer, and the watch runner should need no new production mechanics. New focused lifecycle test files are acceptable when clearer than expanding an existing fixture; reuse helpers and avoid a parallel test framework. Update only the managed active-plan pointer in `AGENTS.md` for downstream work.

## Phase 0 — Resolved Research

See [research.md](research.md). The two focused read-only investigations covered (1) data mapping, version compatibility, statistics, and requests, and (2) ordinary/watch rendering, logging, docs, and integration observation. They resolved all representation and ownership questions before design.

Key decisions: optional plain string; direct shared mappings; stable lifecycle column before statistics; state as part of existing watch display identity; configured logging with propagated summary write errors; extension of existing isolated tests and the existing draining observer.

## Phase 1 — Design and Contracts

- [Data model](data-model.md): field semantics, omission, stable display columns, and baseline transition table.
- [CLI contract](contracts/cli.md): existing invocations, rows, JSON, empty modes, watch, diagnostic routing, and integration acceptance.
- [Library contract](contracts/library.md): additive result field across existing methods; no new methods/options or request behavior.
- [Quickstart](quickstart.md): runnable isolated checks after implementation, prerequisites and expected outcomes, and compile-only integration validation.

### Implementation cohesion for task generation

1. Keep the field additions, v810 mapping, facade propagation, and required conversion/mock assertions together. Include serialized search bodies, v88/v89 omission, page/latest/watch propagation, and statistics/request preservation.
2. Build ordinary and statistics display once in the shared definition projection. Keep the fixed-column layout and watch column-index adaptation together so intermediate work still renders correctly. Include human/JSON/keys/empty/quiet command checks and relevant help/docs updates.
3. Complete watch state comparison/highlighting and the narrow configured-logger corrections together with baseline/error/PTY tests. Use the existing runner; do not split out context reading, progress tracking, or formatting as standalone tasks.
4. Attach the existing draining observer update and its compile check to the relevant read-output work. Do not create a new live suite, live execution gate, or additional no-wait scenarios.

These are cohesion constraints, not a generated task list. Every eventual task includes its necessary tests, formatting, documentation, and targeted validation. Review adjacent tasks for consolidation without separating coupled column or logger/error changes.

### Validation and risk controls

- Prove omission by inspecting serialized keys, not only Go zero values; fresh success after a supplied state must clear it.
- Keep search response `Body` synchronized with `JSON200` fixtures to exercise the actual v810 decoder.
- Test mixed state availability with statistics so variable row lengths cannot shift columns; verify all-unavailable legacy rows remain unchanged.
- Exercise state-only, statistics-only, simultaneous, omitted, unknown, failed, unchanged, skipped-transition, returned-DELETED, and absent-key watch sequences. Preserve one update per changed row.
- Configure real command-context loggers in watch fixtures. Cover plain/structured formats, separate configured/inherited destinations, INFO filtering at WARN, WARN warnings, verbose timing, no duplicate diagnostics, and failed enabled-summary writes retaining baseline. Preserve isolated failure/redaction/DEBUG checks required by repository guidance for touched output paths.
- Extend existing command execution and PTY coverage; renderer-only assertions do not establish stream/configuration compliance. Do not create a broad matrix of unrelated mutation flags.
- Compare request counts against each mode's current fake expectations, including statistics/paging. Lifecycle introduces zero requests or mutations.
- Regenerate docs from command source during implementation. During this planning phase, run only document/link/whitespace checks; no runtime tests or doc generation are needed.
- Compile the edited integration package without running its TestMain or scenarios. Leave actual engine lifecycle timing and availability explicitly unverified.
- Use the full race-enabled suite only if implementation expands shared helpers/mechanics or targeted results leave wider risk unresolved; record the reason. No integration/live tags or environment-backed scripts may enter implementation test commands.

## Complexity Tracking

No constitution violations require justification. The only small helper addition is the error-returning configured watch-summary writer, required to honor logger settings without losing the existing failed-render baseline guarantee. It belongs in the existing definition view file and does not introduce a new logging subsystem.
