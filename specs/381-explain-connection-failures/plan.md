# Implementation Plan: Explain Known Connection and Authentication Failures

**Branch**: `381-explain-connection-failures` | **Date**: 2026-11-18 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/381-explain-connection-failures/spec.md`; [issue #381](https://github.com/grafvonb/c8volt/issues/381).

## Summary

Improve `boundedCommandFailureReason` so callers such as process-definition search explain recognized connection and authentication causes without DEBUG. Extract reusable bounded recognition and wording from the existing connection-test renderer, preserve request-rate precedence, and retain the existing class fallback for unrecognized evidence. Keep connection-test-specific cancellation, topology, setup, and fallback advice local to that renderer.

Start from `10bd90ae`, already the branch HEAD. Do not bring in #379. Production scope is the shared formatter and its connection-rendering helper, plus concise command help metadata required for the behavior change. Existing service, facade, transport, error conversion, logger, envelope, execution, and caller contracts stay intact. Extend focused isolated tests and relevant documentation.

## Technical Context

**Language/Version**: Go 1.26.0; pinned toolchain `go1.26.10`.

**Primary Dependencies**: Existing Cobra 1.10.2 command tree, standard `errors` and `log/slog`, `c8volt/ferrors`, `cmd` connection-fact adapter, `internal/services/httpc.FailureDetails`, and existing logging helpers. No dependency changes.

**Storage**: No new storage, persistent data, migrations, or configuration.

**Testing**: Go testing and testify 1.12.1; existing command subprocess runners, test-owned HTTP fakes, and a child-process-only fake transport for deterministic connection refusal. Separate stdout, stderr, and configured logger capture.

**Target Platform**: Existing c8volt CLI platforms; no platform-specific production behavior added.

**Project Type**: CLI application with public Go facades; this change owns final human rendering in `cmd` only.

**Performance Goals**: Zero additional requests, retries, or response reads; bounded inspection of existing error evidence. No performance benchmark is needed for a wording-only formatter change.

**Constraints**: Typed evidence only; no error-string parsing, raw human transport output, new diagnostic framework, new flags, facade normalization changes, or repository-wide caller migration. Preserve class/exit/JSON evidence, partial results, request-rate precedence, cancellation, and command-specific messages.

**Scale/Scope**: Existing callers of `boundedCommandFailureReason`; two rendering production files, focused test changes, root help metadata, and directly related docs. No service adapters or generated clients change.

## Constitution Check

*GATE: assessed before research and re-evaluated after design.*

| Principle / repository gate | Initial assessment | Post-design evidence |
| --- | --- | --- |
| I. Operational proof | PASS: no operational success claims or verification changes | Formatting adds no probes, submissions, retries, or workflow transitions; compatibility scenarios assert request counts |
| II. CLI-first, script-safe | PASS: human wording correction only | JSON-first handling, original error objects, classes, exit controls, output selection, and partial-result rendering are unchanged |
| III. Proportionate validation | PASS: planning uses document checks only | Implementation starts with targeted isolated command tests; full suite is conditional on a concrete wider regression, not an automatic gate |
| IV. Documentation matches behavior | PASS: directly related docs required | Update README, diagnostics guidance, one generic root help sentence, then regenerate with `make docs-content` |
| V. Small, compatible changes | PASS: reuse current formatter and helper | One private recognition function in the owning connection view; no new framework, package, dependency, or service mechanics |
| Logger and confidentiality | PASS: preserve configured logger boundaries | Execution tests assert ERROR records, destination, both streams, duplicates, normal/verbose redaction, and existing DEBUG rules |
| Implementation testing boundary | PASS: isolated fixtures only | No Camunda installation, cluster, smoke/integration target, or live resource is a validation prerequisite |
| Issue identity / baseline | PASS: branch 381 at `10bd90ae` | Feature directory uses 381; no #379 dependency, branch rename, or unrelated refactor |

No gate violations or unresolved design questions remain.

## Project Structure

### Documentation (this feature)

```text
specs/381-explain-connection-failures/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/human-errors.md
```

`tasks.md` is a later `$speckit-tasks` output and is not generated during planning.

### Source Code (repository root)

```text
cmd/
├── cmd_views_contract.go                       # shared bounded formatter: minimal delegation
├── cmd_views_config_test_connection.go         # existing connection wording + private reuse point
├── cmd_views_contract_test.go                  # proposed focused cause/precedence tests
├── get_processdefinition_error_envelope_test.go # extend affected command execution fixtures
├── config_test_connection_diagnostics_test.go  # retain exact config-specific guidance
├── delete_processdefinition_unfinished_test.go # repeated failures, counts, partial JSON
├── request_rate_mutation_test.go              # existing compatibility fixtures
└── root.go                                    # concise generic help metadata only
README.md
docs/configuration.md
docs/index.md                                  # regenerated from README
docs/cli/                                      # generated; never hand-edited
```

Reference-only dependencies remain `cmd/cmd_errors.go`, `c8volt/ferrors/errors.go`, `internal/services/httpc/failure.go`, and existing `testx`/logging helpers. No production edits to these are planned.

**Structure Decision**: Keep pure rendering in its existing `cmd` view files. No execution mode, service layer behavior, or shared support package is introduced. A private helper remains in the existing connection view so recognition and wording have one owner.

## Design and Implementation Boundaries

### Shared delegation and connection reuse

1. Preserve the existing class-derived fallback (`Classify` with underscores rendered as spaces).
2. Check request-rate-wait evidence first and immediately return the exact existing class-plus-unsent/uncertainty suffix. This also prevents recursion when the connection-test wrapper delegates request-rate handling back to the shared formatter.
3. Ask a private connection-reason function for `(boundedReason, recognized)`. Keep that function in `cmd_views_config_test_connection.go`, using `commandConnectionFailureDetails` and the existing switches. No transport imports or cause matching move into `cmd_views_contract.go`.
4. Return the recognized reason, otherwise the original class fallback. Cancellation is declined by the generic helper, preserving the shared formatter's baseline cancellation fallback rather than returning `connection test canceled` or inventing remediation.
5. Let `configConnectionFailureMessage` retain its prefix, request-rate guard, cancellation wording, and existing class-only fallbacks. Both callers reuse the same cause/status wording. Use a small explicit caller context (for example, a topology-endpoint boolean) solely for the gateway 404 label; preserve `topology endpoint not found` for connection tests and use neutral endpoint advice for ordinary commands. Avoid duplicate status switches or a generalized options framework.

Preserve current cause precedence returned by `FailureDetails`, including malformed-response precedence and independently discovered annotations in joined errors. Do not reclassify evidence or expand the recognized cause set.

### Evidence and output boundaries

`httpc` typed failure → facade conversion through `ferrors` → existing command's bounded human reason → existing configured logger → original exit resolution. JSON takes its existing branch before human overrides and retains the original normalized error. Partial-result paths continue their single-envelope-and-exit behavior.

Some bare transport types are discarded by current `ferrors` normalization. Test those types directly at the formatter boundary and use retained authentication/status annotations in command execution tests. If a cause is unavailable after existing normalization, preserve fallback. Do not change conversion or search raw strings to recover it.

Do not modify `handleCommandErrorWithMode`, `resultEnvelopeForError`, `ferrors`, services, retry code, or command execution just to broaden this feature. Preserve caller prefixes and aggregate counts; only the bounded shared cause suffix changes when supported evidence exists. Existing command-specific overrides remain authoritative.

### Focused validation design

- Add a formatter table for raw, wrapped, joined, and retained normalized evidence: refusal, reset/unexpected EOF, timeout, DNS, certificate, malformed response, HTTP/status and authentication cases, cancellation, misleading text, and class-only fallback. Prove request-rate precedence with combined evidence.
- Extend the process-definition execution fixture for the original OAuth authentication refusal. In its subprocess only, install a fake default transport returning a typed refusal for the test-owned token request and rejecting any unexpected request. Use explicit temporary config, zero retries, and `--no-version-check` for this narrowly scoped fixture; assert one failed auth attempt and zero gateway requests. Exercise actual command → facade → renderer → logger → exit handling; do not construct just a final message for this regression.
- Reuse the existing normal/verbose/quiet/JSON, plain/structured, configured/inherited destination, and flag/config DEBUG matrices. Include quiet+JSON and exit suppression. Assert complete ERROR records and exact counts rather than a substring. Check both streams and any separate logger sink, including when stderr must be empty.
- Extend the existing multi-definition deletion failure fixture instead of building a new batch harness. Keep selected/accepted counts, failed/partial evidence, original exit class, request counts, and one JSON envelope with EOF. Seed repeated distinctive raw evidence and require absence at ordinary human levels. Do not change machine detail to satisfy human redaction checks.
- Preserve exact connection-test-specific output. Retain existing request-rate/cancellation tests and successful-result checks. Assert DEBUG evidence using existing redaction expectations; account for logger format differences and quiet precedence.
- Isolate inherited `C8VOLT_*`, helper-process variables, and Go flags using `isolateRequestRateEnvironment` patterns before supplying explicit fixture settings. No profile, credential, proxy, or environment may route tests to a real service; fake transports must reject unexpected targets.

Required assertions must fail if the human ERROR is replaced by a direct stderr print, duplicated, sent to stdout, or replaced with a raw joined error. Review changed direct prints and error interpolation at every touched output boundary.

### Documentation and validation scope

Update README near the existing verbose/DEBUG guidance and `docs/configuration.md` diagnostics section to explain that recognized causes and next checks appear without DEBUG. Add one concise generic sentence to root command `Long` help; do not edit individual command handlers merely for documentation. Regenerate via `make docs-content`, which constructs command metadata without executing commands and synchronizes README into `docs/index.md`. Review generated changes; never hand-edit `docs/cli`.

Planning validation is limited to artifact structure, traceability, local links, feature selection, whitespace, and diff review. Implementation commands and expected assertions are in [quickstart.md](quickstart.md). No runtime check is claimed by this plan. Focused tests cover the actual formatter blast radius; broader package/race validation requires new failure evidence or a wider executable diff. Real-engine behavior remains unverified and is not a completion gate.

Planning checks passed on 2026-11-18: required artifact/section presence, resolved placeholders, local Markdown links, fenced-block balance, whitespace, feature selection, active agent plan pointer, and existing test-name references. The proposed formatter test is explicitly identified as new. Constitution gates passed before and after design. No Go tests or documentation generation ran for this planning-only change.

### Task-generation guidance

Keep helper extraction, shared delegation, affected exact expectations, necessary execution/logger safety tests, formatting, docs, and targeted validation in cohesive work units that compile. Do not create separate tasks for context reading, branch checks, progress tracking, or hypothetical parallel execution. Do not ship a single-cause implementation and later rebuild the same formatter for the remaining supported causes. Treat compatibility assertions as required coverage in the relevant implementation unit, while honoring Ralph's single-story scope; user stories must not force repeated production rewrites. Review adjacent proposed tasks for consolidation and explain any deliberate separation before finalizing `tasks.md`.

## Complexity Tracking

No constitution exceptions. The private recognition function replaces duplicated cause wording; caller context only preserves the existing topology-specific advice. No new abstractions or runtime mechanics are justified or required.
