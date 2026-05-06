# CLI Contract: Process Definition Selector Validation

## Scope

The contract applies when process-instance commands receive `--bpmn-process-id`:

- `c8volt get pi`
- `c8volt cancel pi`
- `c8volt delete pi`
- `c8volt run pi`

Commands without `--bpmn-process-id` keep their existing behavior.

## Shared Selector Context

Validation must use the same process-definition selector context the command implies:

- BPMN process ID or IDs
- `--pd-version`, when provided
- `--pd-version-tag`, when provided by commands that support it
- effective tenant context from flags/configuration

## Successful Validation

When every requested BPMN process ID has a visible process-definition match:

- `get pi` continues to process-instance search and may still return `found: 0` when no instances match.
- `cancel pi` and `delete pi` continue to their existing paging, dry-run, confirmation, and mutation flow.
- `run pi` continues to create process instances.

## Failed Validation: Human Interactive Output

When one selector is missing or invisible, human output should include:

```text
No visible process definition matches the provided selector:
  bpmnProcessId: <id>

It may not exist, the version/tag/tenant may not match, or your credentials may not have access.
List visible process definitions? [Y/n]:
```

When multiple selectors are missing or invisible, human output should include each missing BPMN process ID:

```text
No visible process definitions match the provided selector(s):
  bpmnProcessId: <id-a>
  bpmnProcessId: <id-b>

They may not exist, the version/tag/tenant may not match, or your credentials may not have access.
List visible process definitions? [Y/n]:
```

If the user accepts the prompt, visible process definitions must render using the existing process-definition list format, including the final `found: <n>` summary.

## Failed Validation: Non-Interactive and Machine Modes

The command must fail clearly without prompting when any of these apply:

- `--json`
- `--automation`
- `--keys-only`
- stdin/stdout usage where prompting would block

Structured output behavior must follow existing command error conventions.

## Multi-ID All-Or-Nothing Behavior

For `run pi` with multiple BPMN process IDs:

- Validate all BPMN process IDs before creating any process instances.
- If any BPMN process ID is missing or invisible, fail the whole command.
- Do not create process instances for valid IDs when another ID in the same request fails validation.

## Exit and Error Expectations

- Missing or invisible process-definition selectors are command failures.
- The error must identify the selector validation problem, not only report zero process-instance matches.
- Existing no-error-code behavior must continue to honor repository conventions.
