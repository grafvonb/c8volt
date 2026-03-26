# Research: Add Process Definition XML Command

## Decision 1: Reuse the existing versioned XML service methods through the public process facade

- **Decision**: Expose process-definition XML retrieval by adding a matching method to the public `c8volt/process` facade and delegating to the existing `internal/services/processdefinition` XML methods already implemented for the supported versions.
- **Rationale**: Both versioned processdefinition services already support XML retrieval, so the lowest-risk approach is to extend the existing facade wiring instead of adding a parallel command-only service path.
- **Alternatives considered**:
  - Call internal services directly from the CLI command: rejected because it bypasses the repository's current facade layering.
  - Add a separate top-level export command: rejected because the issue requests a `get process-definition` extension, not a new command family.

## Decision 2: Make `--xml` a single-item mode that requires `--key`

- **Decision**: Treat `--xml` as an opt-in single-definition retrieval mode that requires `--key` and rejects list-style filters or render modes that do not fit a raw XML payload.
- **Rationale**: XML retrieval is defined for one process definition at a time, and forcing explicit single-key selection prevents ambiguous list behavior and accidental misuse in scripts.
- **Alternatives considered**:
  - Infer the target from list filters like `--bpmn-process-id` or `--latest`: rejected because matching could return multiple definitions and create surprising behavior.
  - Silently pick the first matching definition: rejected because it hides selection logic from operators and breaks trustworthy CLI semantics.

## Decision 3: Write raw XML directly to standard output and bypass the generic render helpers

- **Decision**: When `--xml` is used successfully, print the XML payload directly to stdout instead of routing it through the existing one-line, JSON, or keys-only render helpers.
- **Rationale**: Redirect safety is a primary user requirement, and direct stdout output keeps the resulting file usable without extra formatting, counts, or summary lines.
- **Alternatives considered**:
  - Extend the generic render-mode helpers with an XML mode: rejected because it adds broader rendering complexity for a narrowly scoped feature.
  - Wrap the XML in JSON or metadata by default: rejected because it would make `> example.bpmn` workflows harder to use.

## Decision 4: Reject incompatible output flags instead of inventing mixed semantics

- **Decision**: Define `--xml` as incompatible with generic render modifiers such as `--json`, `--keys-only`, and any other list-oriented path that would conflict with raw XML output.
- **Rationale**: Explicit validation is clearer and safer than trying to guess which output mode should win when the user asks for both structured summaries and raw XML.
- **Alternatives considered**:
  - Let `--xml` silently override the other output flags: rejected because hidden precedence rules are hard to discover in scripts.
  - Let `--json` wrap the XML string: rejected because it does not satisfy the shell-redirection workflow called out in the issue.

## Decision 5: Validate at command level and keep service coverage targeted

- **Decision**: Add focused `cmd/` tests for XML success, failure, and redirect-safe output behavior, and only extend lower-level coverage where public facade wiring or uncovered XML edge cases need explicit regression protection.
- **Rationale**: The main behavioral change is in the public CLI surface and output contract, so command-level tests provide the highest-value regression signal while existing versioned service tests already cover much of the underlying XML retrieval logic.
- **Alternatives considered**:
  - Rely only on service tests: rejected because they do not prove the CLI flag, validation, and stdout behavior work correctly.
  - Rely only on manual command checks: rejected because repository policy requires automated validation and `make test`.

## Decision 6: Update help text first and regenerate CLI docs from Cobra metadata

- **Decision**: Capture the new XML behavior in Cobra help text and regenerate `docs/cli/` with `make docs`, with `README.md` changes limited to process-definition retrieval examples if needed.
- **Rationale**: This repository treats `docs/cli/` as generated output from command metadata, so documentation must follow the command help rather than diverging through hand-edited reference pages.
- **Alternatives considered**:
  - Edit `docs/cli/c8volt_get_process-definition.md` directly: rejected because the repository already has a doc generation path.
  - Skip docs because the feature is small: rejected because the change is user-visible and introduces a new operator-facing flag.
