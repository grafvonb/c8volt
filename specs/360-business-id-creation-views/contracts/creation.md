# Contract: Business-ID Creation

**Scope**: [specification](../spec.md), FR-001–FR-007 and FR-014–FR-017.

## CLI

Add scalar string flag `--bid <value>` to `run process-instance` and its existing `pi` alias. Use `Flags().Changed("bid")` to distinguish omission from `--bid ''`; do not use a string-slice flag. Apply it to every existing constructed creation input for `--bpmn-process-id` and `--pd-key`, including supported collection selection. Existing selector/version/tenant/count/worker flag rules remain unchanged.

Illustrative user commands, not implementation-validation commands:

```text
c8volt run pi --bpmn-process-id order --bid 'order,AbC'
c8volt run pi --pd-key 2251799813685249 --bid 'order-123' --no-wait
c8volt run pi --bpmn-process-id order --bid 'order-123' --count 3 --workers 1
```

These require an explicitly configured supported 8.10 environment when used by an operator; do not change the default version. The last example may partially fail when backend uniqueness enforcement rejects repeated active roots. Do not promise all three will be accepted or invent alternate IDs.

Nil/omitted input submits no field. Supplied input must be nonempty, valid Unicode, and at most 256 UTF-16 code units. Whitespace is preserved and may itself be a valid nonempty ID. Both selectors and every requested count use the exact accepted value.

## Public library interface

Extend `c8volt/process.ProcessInstanceData` with `BusinessId *string` (`json:"businessId,omitempty"`). Retain existing signatures:

```go
CreateProcessInstance(ctx context.Context, data ProcessInstanceData, opts ...options.FacadeOption) (ProcessInstance, error)
CreateProcessInstances(ctx context.Context, datas []ProcessInstanceData, opts ...options.FacadeOption) ([]ProcessInstance, error)
CreateNProcessInstances(ctx context.Context, data ProcessInstanceData, n int, wantedWorkers int, opts ...options.FacadeOption) ([]ProcessInstance, error)
```

`options` denotes the existing facade-options import alias. No new method or facade option is needed. Add `BusinessId *string` to returned `ProcessInstance`. Facades mechanically map input/output and return converted errors through `ferrors.FromDomain`, preserving nonempty partial slices on error. They do not schedule, wait, or perform version-specific transport logic.

Nil input works as before across 8.8/8.9/8.10. Non-nil business-ID input on 8.8/8.9 returns unsupported before that operation mutates; the check belongs in each older adapter and applies to public calls as well as CLI calls. v810 validation returns the existing invalid-input class. Backend conflicts retain conflict classification; timeouts/cancellation retain their existing classification.

## Native mapping

Both generated `ProcessInstanceCreationInstructionById` and `ProcessInstanceCreationInstructionByKey` receive `BusinessId` only when supplied. Preserve selector discriminators, version, tenant, and variable fields. Use generated request serialization in test-owned HTTP fixtures to verify exact body content and absence; converter-only tests are insufficient.

Native creation responses map through `ProcessInstanceCreation`. Ordinary waiting replaces its business ID with the successful observation's ID, including nil. No-wait retains the native response ID. Neither path falls back to the submitted input. No new read or retry is introduced for this field.

## Results, failure, and streams

- Success keeps the existing `ProcessInstances` payload and shared success/accepted envelope semantics. `--no-wait` indicates acceptance only.
- Partial failure includes only successful creation results under the existing payload shape, including their business IDs. The error envelope retains the original class/detail and failure outcome. `total` reflects returned items, not requested count.
- Render exactly one envelope with the partial payload, then use `exitAfterRenderedResult` with the original normalized error and existing `NoErrCodes` behavior. Never call a second error renderer afterward. A writer error must not trigger a second attempted JSON envelope; preserve operation-error exit precedence when one exists.
- Keys-only emits only successful keys, one per line, zero bytes if none, and still returns the original failure exit status.
- Human creation rows, summaries, and verbose outcomes use configured INFO logging via established helpers. Show a compact `bid:<value>` field beside the process name when present, quoting empty values and values containing characters other than letters, digits, or `-_.@/:`. Absent IDs do not add an empty column. Structured logging must not hide the ID solely in attributes unsupported by the plain handler.
- Non-JSON failures use one configured ERROR with counts supported by returned evidence and bounded grouped reasons. Do not label every unreturned request a backend rejection: fail-fast, cancellation, and unattempted work differ. Do not print raw joined errors or transport text. Existing DEBUG logging alone carries redacted HTTP details.
- Ordinary query results remain stdout payloads. Prompts remain plain configured stderr; no new confirmation is introduced.

## Compatibility note

Adding optional fields is additive for keyed Go struct literals and JSON consumers. Consumers using unkeyed public struct literals may need source updates. Absent values preserve prior serialized output. Human creation output moves from direct stdout to configured INFO logging to satisfy the repository's mutation-output contract; document this visible stream correction. Existing keys, totals, envelope structure, flags, and failure classes remain stable.
