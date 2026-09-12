# Implementation Plan: Standard Tenant Logging for Delete and Cancel

**Branch**: `codex/303-standard-tenant-logging` | **Date**: 2026-09-11 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/303-standard-tenant-logging/spec.md`

## Summary

Correct the two tenant-context emitters in `cmd/processinstance_mutation_progress.go` by replacing each raw stderr write with `printOpsDurableLine(cmd, line.Text, line.Warn)`. This restores existing standard INFO/WARN severity, configured formatting, and threshold filtering. Preserve every eligibility check, line producer, ordering decision, and rendered-state update. Add attached-logger regression coverage and command compatibility checks, and document the visible logging correction.

## Technical Context

**Language/Version**: Go 1.26; declared toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra v1.10.2, standard-library logging through `toolx/logging`, and existing command operational logging helper. No new dependencies.

**Storage**: No persistence or schema changes; command-context tenant evidence and rendered state already exist.

**Testing**: Go tests with testify, existing command stubs and real-terminal subprocess fixtures; `make test` runs `go test ./... -race -count=1`.

**Target Platform**: Existing supported CLI platforms; terminal regression fixtures run on their supported host platforms.

**Project Type**: CLI with public facade and internal service packages; this change stays in command rendering.

**Performance Goals**: One existing logging call per eligible line, no additional discovery, mutation requests, polling, or worker activity.

**Constraints**: Preserve message text, WARN classification, ordering, deduplication, mode precedence, prompt behavior, machine results, and all service decisions. Do not modify the shared logging helper or other raw stderr messages.

**Scale/Scope**: Two emitters shared by selector-based delete and cancel; adjacent tests and documentation only.

## Constitution Check

| Principle / gate | Pre-research assessment | Post-design assessment |
| --- | --- | --- |
| Operational proof | Pass: no completion, wait, or mutation semantics change | Pass: compatibility tests retain request counts, aborts, and outcomes |
| CLI-first, script-safe | Pass: stdout contracts and guards remain unchanged | Pass: separate stdout/stderr assertions, structured-result decoding, and real-terminal prompt regression retained |
| Tests and validation | Pass: defect is reproducible with an attached logger | Pass: targeted format/severity/filter tests, command regression, then required race-enabled full suite before implementation commit |
| Documentation matches behavior | Pass: visible logging correction requires documentation | Pass: update README tenant paragraph and delete/cancel source metadata; regenerate via `make docs-content` |
| Small compatible changes | Pass: reuse existing helper and existing test fixtures | Pass: no abstraction, dependency, service, facade, generated client, or lifecycle changes |

No gate failures or unresolved design questions. These are design assessments, not claims that implementation tests have run.

## Project Structure

### Documentation (this feature)

```text
specs/303-standard-tenant-logging/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/tenant-logging.md
└── tasks.md                     # Future speckit-tasks output
```

### Source Code (repository root)

```text
cmd/
├── processinstance_mutation_progress.go       # Two emission replacements
├── processinstance_mutation_progress_test.go  # Attached-logger matrix
├── ops_progress_render.go                    # Existing helper, unchanged
├── cmd_tenant_context.go                     # Existing text/Warn producers, unchanged
├── cmd_views_tenant_context.go               # Existing eligibility/rendering, unchanged
├── cancel_processinstance.go                 # Help metadata clarification
├── delete_processinstance.go                 # Help metadata clarification
├── cancel_processinstance_test.go            # Execution regression
├── delete_processinstance_test.go            # Execution regression
├── cancel_processinstance_selector_test.go   # Existing mode/empty/sparse coverage
├── delete_processinstance_selector_test.go   # Existing mode/empty/sparse coverage
└── cmd_confirmation_terminal_test.go         # Existing real-terminal coverage
README.md
docs/cli/                                   # Generated through make docs-content
toolx/logging/                              # Existing logging implementation, unchanged
```

**Structure Decision**: Keep the fix in the existing shared progress file; it introduces no mode or lifecycle declarations. Reuse command test support and terminal fixtures. No changes below `cmd/` are needed.

## Phase 0: Research Results

See [research.md](research.md). The shared helper preserves raw stderr output only when no logger is attached, routes WARN/INFO correctly with a logger, and never falls back after filtering. Existing dry-run preview renderers remain untouched. Standard logger configuration is distinct from JSON result selection.

## Phase 1: Design and Validation

1. Add focused failing regression cases for both emitters using a real attached logger, independent stdout/stderr buffers, and mixed informational/warning tenant context. Cover plain/JSON formats and INFO/WARN/ERROR thresholds, exact text/order, missing context, and repeat reporting.
2. Replace only the final emission inside each existing loop with the shared helper, passing both fields. Keep rendered marking before emission, including when the threshold suppresses every line. Preserve no-logger fallback tests.
3. Extend existing delete/cancel execution tests where needed to cover attached logging and result-stream separation. Retain ordinary, verbose, quiet, automation, auto-confirm, dry-run, JSON-result, keys-only, and supported combined-mode behavior. Retain empty/sparse discovery, direct keys, failures, aborts, and no-wait compatibility. Use real terminal stdin for prompt assertions, with configured/inherited stderr and stdout captured separately. Avoid duplicating existing comprehensive scenarios when they already prove the contract.
4. Clarify in README and the two command descriptions that eligible selector tenant diagnostics use standard INFO/WARN logging and configured format/level filtering. Document no change to tenant visibility policy; regenerate CLI documentation, never edit generated pages by hand. JSON diagnostic logging does not promise that interactive stderr consists solely of JSON: prompts remain plain text.
5. Run targeted tests, inspect exact output/JSON decoding and request counts, run `make docs-content`, gofmt touched Go files, then run `make test` before implementation commit. Inspect the final diff for unintended scope changes.

Contracts and conceptual state are documented in [contracts/tenant-logging.md](contracts/tenant-logging.md) and [data-model.md](data-model.md). [quickstart.md](quickstart.md) provides runnable validation commands.

## Compatibility and Risks

- Consumers of these diagnostic lines will now see standard prefixes or structured records and threshold suppression. This is the intended compatibility correction; result schemas and stdout remain stable.
- Existing tests without a logger cannot detect this defect; attached-logger cases are required.
- Changing the rendered marker timing would introduce replay after filtering. Preserve it exactly.
- Shared package flags require existing cleanup/reset patterns; do not parallelize tests that mutate them.
- Do not route confirmation questions, paging text, dry-run previews, or unrelated warnings through logging.

## Complexity Tracking

No constitution exceptions or added complexity.
