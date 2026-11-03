# Implementation Plan: Standardize Flag Intent

**Branch**: `369-standardize-flag-intent` | **Date**: 2026-11-03 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/369-standardize-flag-intent/spec.md`

## Summary

Deliver the agreed v5 breaking CLI grammar: `set-*` supplies written/initialized values, `with-*` requests extra detail, and bare properties remain selectors. Apply all 16 rename-table rows, split job retry filtering from mutation, split the BPMN-error action from its code, and restore inherited HTTP timeout resolution. Preserve backend payloads, public-library APIs, supported mode combinations, exact values, operational evidence, output schemas, and exit classifications.

Remove renamed legacy spellings immediately with no aliases or warning period. Rejected syntax receives canonical migration guidance through configured logging and, where supported, exactly one shared JSON error envelope. A small diagnostic-only argument preparation path closes the current pre-bootstrap parsing gap without contacting Camunda. See [research.md](research.md) for evidence and alternatives.

## Technical Context

**Language/Version**: Go 1.26.0, toolchain go1.26.2 as pinned in `go.mod`.

**Primary Dependencies**: Existing Cobra 1.10.2, pflag 1.0.10, Viper 1.21.0, slog-based `toolx/logging`, testify 1.11.1, and repository `testx` helpers. No dependency changes.

**Storage**: Existing configuration files/environment/profile resolution only; no new persistence or schema.

**Testing**: Go unit tests, test-owned HTTP fake servers, existing generated-client doubles, isolated subprocess/real-terminal fixtures, documentation generation and static inventory checks. No integration build tags or real services.

**Target Platform**: Existing Go CLI platforms; real-terminal checks follow current platform guards.

**Project Type**: CLI with public library facades; this feature changes the CLI boundary only.

**Performance Goals**: No additional discovery, mutation, authentication, or verification requests; diagnostic token processing linear in invocation length. Existing paging/concurrency limits remain unchanged.

**Constraints**: Clean v5 break; no accepted legacy aliases, no new mutation modes, no public-field/result-schema renames, no change to repair zero-retry behavior. Preserve configured logging, plain stderr prompts, and uncontaminated stdout results. No live-environment implementation validation.

**Scale/Scope**: 16 rename-table rows across update element/process/task/job, process run, two repair commands, process-definition enrichment, embedded listing, and incident suppression; shared command/bootstrap error support plus documentation/fixtures.

## Constitution Check

*Pre-research gate evaluated against `.specify/memory/constitution.md` v2.0.0. Post-design reevaluation below.*

| Principle / constraint | Pre-research | Design evidence / post-design |
| --- | --- | --- |
| Operational proof over intent | PASS | Existing mutation verification/acceptance/no-wait semantics are unchanged; no empty-scope acceptance claim or extra read is added. PASS. |
| CLI-first, script-safe interfaces | PASS with explicit compatibility note | User explicitly authorized a v5 break. Old syntax is rejected, output schemas/exit classes remain stable, and migration guidance names canonical replacements. PASS. |
| Proportionate validation | PASS | Planning uses document checks only. Implementation uses isolated targeted tests and extends to shared cmd/config/docsgen boundaries; broader race suite only for demonstrated shared runtime blast radius. PASS. |
| Documentation matches behavior | PASS | Source metadata, README, generated docs, completion, maintained scripts/examples and invocation fixtures move with each behavior slice. PASS. |
| Repository-native, small changes | PASS | Existing mode files, input helpers, error classes, rendering and service workflows retained. Diagnostic input recovery is bounded and justified by early parse errors. No dependency/framework introduced. PASS. |
| Layering and implementation test boundary | PASS | No command calls to generated/versioned clients, no facade/service workflow moves, no live validation. PASS. |

No unresolved gates or clarifications. The strictly valueless BPMN-error action is the technical resolution required to reject legacy codes `true` and `false` without reinterpretation.

## Project Structure

### Documentation (this feature)

```text
specs/369-standardize-flag-intent/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/cli-flags.md
```

`tasks.md` is intentionally deferred to `$speckit-tasks`.

### Source Code (repository root)

```text
cmd/
├── root.go, root_config.go, bootstrap_errors.go, cmd_bootstrap_failure.go
├── cmd_errors.go, command_contract.go, cmd_views_contract.go
├── flag_migration.go                 # proposed diagnostic map/validation
├── flag_migration_input.go           # proposed diagnostic-only token recovery
├── update_variables_payload.go
├── update_element.go, update_element_variables.go
├── update_processinstance*.go        # registration, variables, BID exclusions
├── update_usertask*.go               # registration, assignment, completion, variables
├── update_job.go, update_job_request.go, update_job_outcome.go, update_job_batch.go
├── run_processinstance.go
├── ops_repair*.go
├── get_processdefinition*.go, get_incident.go, embed_list.go
└── *_test.go                         # existing families plus focused migration tests
config/                               # unchanged schema; timeout binding regression coverage
c8volt/, internal/services/            # unchanged production contracts and workflows
testx/                                # reuse subprocess/terminal helpers
docsgen/, docs/cli/, docs/index.md     # regenerate documentation from source/README
README.md
scripts/, demos/vhs/, integration/     # invocation text audit only; no live execution
```

**Structure Decision**: Stay within existing package ownership. Shared input normalization/error support lives in focused `cmd/flag_migration*.go` files; no generic migration framework. Job action parsing remains in request/outcome files; batch lifecycle stays in its existing mode file. Final formatting remains in existing `cmd/cmd_views_*.go` renderers. New migration-only diagnostic support does not acquire service mechanics.

## Phase 0: Research Outcome

[research.md](research.md) resolves parser failure timing, leaf command identity, logger/JSON bootstrapping, exact-zero retries, Boolean-looking error codes, inherited timeout binding, source inventory, and isolated validation. No external API or library change is needed.

## Phase 1: Design

### Input and error flow

1. Register canonical flags only; a command-scoped diagnostic map knows removed spellings but never accepts or forwards them.
2. Preserve original argv and the selected leaf identity at the execution boundary. Use token boundaries/registered arity, not substring replacement, to identify removed names, attached action values, and diagnostic controls. Respect `--` and opaque flag values.
3. Parsing/Args failures enter local-only error-context preparation, recovering known output/configuration flags even after the bad token. Reuse existing configuration normalization, command-context logger/activity setup and fallback helpers; do not validate remote credentials, install services, authenticate, consume stdin, or read mutation files.
4. Semantic migration checks run after local context exists and before remote service bootstrap. Emit one classified invalid-input outcome through existing command rendering. Ensure returned errors and exit-after-render paths cannot render it twice. Normal unrelated failures preserve existing behavior.
5. Valid canonical input follows the existing command runner, facade, and service workflow. Separate query retries from submitted retries, and job timeout from inherited HTTP timeout, before constructing unchanged public inputs.

Do not turn the diagnostic token walker into an alternative command parser: recover only local reporting controls into temporary state, keep the original error, never execute recovered input. Parser tests must include flags after rejection, short/attached forms, flag-looking payload values, and terminator behavior.

### Operation mapping

- Update shared variable helpers and all users together, including source labels, presence tests and error messages; preserve operation-specific decoders and optional versus required payload behavior.
- Rename task assignment checks across dispatch, assignment input, force/JSON guardrails and completion exclusions. Keep `--unassign`, force/replacement policy and result operation names unchanged.
- Keep process BID selection and assignment exclusions intact while replacing variable flag references. `run pi` initializes BID through `--set-bid` but still maps to the existing creation request.
- Job retry modes read only `set-retries`; selector completion reads only `retries`. Batch/keyed restrictions remain as before. BPMN-error code is separate from action; reject attached values and surplus legacy positionals, including Boolean-looking codes.
- Remove job-local `timeout`; all mutation exclusions check `set-timeout`. Verify `http.timeout` binds to the inherited flag, not the mutation. A key plus HTTP timeout alone fails; either timeout may accompany an independently valid canonical mutation.
- Repair uses renamed values but preserves zero as skip restoration. No new skip flag or literal-zero semantics.
- Enrichment/suppression changes retain query validation, watch restrictions, returned data and output-mode support.

### Contracts and model

[contracts/cli-flags.md](contracts/cli-flags.md) is the executable-behavior contract for canonical syntax, rejection, timeout semantics and output routing. [data-model.md](data-model.md) records transient command intent and its mapping to unchanged requests. There is no new backend entity or persistence migration.

### Validation and documentation

Follow [quickstart.md](quickstart.md). Each implementation work unit includes related source changes, fixtures, tests, formatting, targeted checks, README/source examples and regenerated affected references. Shared migration error coverage must demonstrate that direct prints, default-logger routing, raw aggregate output and a second JSON envelope fail assertions.

Maintain request-count, partial-result, no-wait, dry-run, confirmation, empty-scope and sparse-page regressions. Configuration checks verify actual effective HTTP settings and captured job payloads independently. Static audit updates maintained live-suite invocation strings but never runs those suites or designates a target. Historical feature artifacts remain evidence rather than being mechanically rewritten.

## Work-Unit Guidance for Task Generation

Use cohesive independently verifiable outcomes, not separate registration/test/docs tasks:

1. Shared rejection/error-context support with its isolated logging/JSON/ordering coverage.
2. Job retry/timeout/BPMN-error grammar as one coordinated parser outcome, including affected batch/outcome tests and metadata.
3. Shared variable-source renames and affected element/process/task dispatch, exclusions and execution coverage together; do not temporarily break shared helper callers.
4. Creation/repair initialization and related payload/zero semantics; split creation from repair if needed for a practical iteration because their workflows are independent.
5. Enrichment/suppression surface updates with query/watch validation and metadata coverage.

These are proposed boundaries, not tasks. Every unit also updates affected maintained usage and docs. A final consistency review verifies complete coverage rather than deferring all docs or tests to a trailing task. Keep shared error support distinct because it has its own independently verifiable reporting contract; keep job input changes together because they share action dispatch and presence handling. Do not create speculative parallelization tasks. More than ten generated tasks requires consolidation review, not automatic rejection.

## Complexity Tracking

No constitution violations. The only additional command support is diagnostic preparation for pre-bootstrap input errors; ordinary flag removal cannot satisfy configured logging and shared JSON when Cobra stops before the root pre-run. The helper is limited to error reporting, avoiding a second execution parser or legacy alias framework.
