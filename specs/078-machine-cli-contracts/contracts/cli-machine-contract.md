# Contract: Machine-Readable CLI Discovery And Result Model

## Discovery Command Contract

The machine-readable discovery surface is one dedicated top-level command.

| Concern | Required behavior |
|--------|-------------------|
| Entry point | Expose discovery through a dedicated top-level command such as `c8volt capabilities --json` |
| Coverage | Include top-level and nested command paths |
| Command metadata | Include path, aliases, summary, mutation class, output modes, flags, and contract support status |
| Truthfulness | Do not mark commands as fully supported unless they actually return the shared result envelope |

### Minimum discovery example

```json
{
  "command": "capabilities",
  "version": "v1",
  "commands": [
    {
      "path": "get process-instance",
      "aliases": ["get pi"],
      "summary": "List or fetch process instances",
      "mutation": "read_only",
      "contractSupport": "full",
      "outputModes": [
        {"name": "json", "supported": true, "machinePreferred": true},
        {"name": "keys-only", "supported": true, "machinePreferred": false}
      ],
      "flags": [
        {"name": "key", "type": "stringSlice", "required": false, "repeated": true},
        {"name": "json", "type": "bool", "required": false, "repeated": false}
      ]
    }
  ]
}
```

The exact field ordering may differ, but the contract must preserve the same semantics.

## Result Envelope Contract

Supported commands in machine-readable mode use one shared top-level result envelope.

| Field | Required behavior |
|------|-------------------|
| `outcome` | Must be one of `succeeded`, `accepted`, `invalid`, `failed` |
| `command` | Must identify the canonical command path |
| `payload` | Must preserve the command-family-specific JSON payload |
| `class` | May expose the detailed repository-native class when helpful |
| `detail` | Must carry actionable detail for `invalid` and relevant `failed` outcomes |

### Succeeded example

```json
{
  "outcome": "succeeded",
  "command": "get process-instance",
  "payload": {
    "total": 1,
    "items": [
      {"key": "2251799813711967", "state": "active", "tenantId": "tenant-a"}
    ]
  }
}
```

### Accepted example

```json
{
  "outcome": "accepted",
  "command": "run process-instance",
  "payload": {
    "items": [
      {"key": "2251799813711967", "status": "requested"}
    ]
  }
}
```

### Invalid example

```json
{
  "outcome": "invalid",
  "class": "invalid_input",
  "command": "run process-instance",
  "detail": {
    "message": "provide either --pd-key or --bpmn-process-id",
    "suggestion": "set one process definition selector"
  }
}
```

### Failed example

```json
{
  "outcome": "failed",
  "class": "unavailable",
  "command": "get cluster topology",
  "detail": {
    "message": "service unavailable: get cluster topology: gateway timeout"
  }
}
```

## Outcome Contract

| Outcome | Required meaning |
|--------|-------------------|
| `succeeded` | Confirmed successful completion |
| `accepted` | State-changing work requested successfully, but not yet confirmed complete |
| `invalid` | Caller-correctable input or validation problem |
| `failed` | Non-validation execution failure |

The outcome vocabulary is intentionally smaller than the underlying `ferrors` class model.

## Exit-Code Alignment Contract

| Rule | Required behavior |
|------|-------------------|
| Process-level signal | Existing exit codes remain authoritative |
| JSON detail | The envelope provides structured detail but must not contradict the exit code |
| Invalid input | Must still preserve the current invalid-args process semantics |
| Non-validation failures | Must still preserve the current repository exit-code semantics for not found, timeout, unavailable, conflict, and generic error cases |

The feature must not introduce a JSON-only success model that disagrees with the current CLI process behavior.

## Contract Support Contract

Every command listed in discovery must report one of these support states:

| Status | Required meaning |
|--------|-------------------|
| `full` | The command fully supports the shared result envelope |
| `limited` | The command exposes some machine-readable behavior, but not the full shared contract |
| `unsupported` | The command is visible in discovery but should not be treated as part of the shared machine contract yet |

Unsupported or limited commands stay visible in discovery so automation can make informed choices.

## Representative Family Contract

The initial contract rollout must cover at least one representative command from each of these families:

- `get`
- `run`
- `expect`
- `walk`
- `deploy`
- `delete`
- `cancel`

The discovery command may list more commands, but the initial acceptance scope only requires representative full-contract coverage for those families.

## Documentation Contract

The recommended automation contract must be described in:

- command help text for the discovery surface
- `README.md`
- generated CLI docs under `docs/cli/`

Generated docs must be refreshed from Cobra metadata rather than edited by hand.
