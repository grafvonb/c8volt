# CLI Contract: BPMN Selector Validation for Operational Commands

## Scope

This contract applies when a command directly receives `--bpmn-process-id`:

- `c8volt get pi`
- `c8volt run pi`
- `c8volt cancel pi`
- `c8volt delete pi`
- `c8volt get incident`
- `c8volt get pd`
- `c8volt delete pd`

Commands that do not directly receive `--bpmn-process-id` keep their existing behavior.

## Shared Selector Context

Validation uses the selector context implied by the command:

- BPMN process ID
- `--pd-version`, where supported
- `--pd-version-tag`, where supported
- effective tenant context from flags and configuration
- latest-definition mode only for commands that explicitly use latest semantics

## Successful Validation

When the BPMN selector has a visible process-definition match:

- `get pi`, `cancel pi`, and `delete pi` continue to existing process-instance search, dry-run, confirmation, and mutation behavior.
- `get incident` continues to existing incident search, paging, total, key-only, and process-instance-key-only behavior.
- `get pd` and `delete pd` continue to existing process-definition listing, selection, preview, confirmation, and deletion behavior.
- Empty process-instance or incident result sets remain valid when no runtime resources match.

## Failed Validation

When no visible process definition matches a selector, aligned commands fail with the shared diagnostic:

```text
no visible process definition matches the provided selector: [<bpmn-process-id>]
```

When multiple selectors are checked by a command, the plural shared diagnostic is used.

Validation failure occurs before:

- process-instance search paging
- incident search paging
- dry-run planning
- confirmation prompts
- cancellation, deletion, or create mutation submission

## Human Recovery Output

Prompt-eligible human output may offer to list visible or matching process definitions after the command has already determined that the selector is invalid.

The recovery listing must not change the command outcome; the original selector failure remains the command failure.

## Non-Interactive and Machine Modes

The command must fail clearly without recovery prompts when any of these apply:

- `--json`
- `--automation`
- `--keys-only`
- `--pi-keys-only`
- non-TTY stdin or stdout
- stdin key pipelines into downstream commands

Structured output behavior follows existing command error conventions.

## Pipeline Boundary

For a pipeline such as:

```sh
c8volt get pi -b <id> --keys-only | c8volt cancel pi -
```

the upstream selector command validates the BPMN process ID. The downstream keyed command does not infer or repeat BPMN validation because it receives keys, not a direct BPMN selector.

## Direct Process-Definition Commands

`get pd -b` and `delete pd -b` must have explicit missing-selector behavior covered by tests and documentation. The preferred outcome is the shared missing visible process-definition diagnostic for BPMN selector misses; if implementation preserves a distinct direct not-found contract, that contract must be documented and tested.
