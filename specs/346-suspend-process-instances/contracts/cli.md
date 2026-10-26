# CLI Contract: Suspend Process Instances

**Feature**: [../spec.md](../spec.md) | **Result model**: [../data-model.md](../data-model.md)

## Command and inputs

```text
c8volt suspend process-instance --key <key> [--key <key> ...]
c8volt suspend pi --key <key>,<key> --no-wait
c8volt suspend pi --auto-confirm -
c8volt suspend pi --key <key> --dry-run --json
```

These are command syntax examples, not instructions to run against a live target during implementation.

| Input/control | Contract |
| --- | --- |
| `process-instance`, alias `pi` | New child of the new `suspend` verb. |
| `--key`, `-k` | Repeated/comma-separated explicit keys, merged before stdin; first-occurrence order after trimming/deduplication. |
| Positional `-` | Read newline-separated keys from piped/redirected stdin using existing helpers. Empty/whitespace-only input is invalid; terminal stdin with `-` is invalid. |
| `--dry-run` | Local capability/input validation and exact-key preview; no reads, mutations, or confirmation prompts. Does not assert ACTIVE eligibility. |
| `--no-wait` | Return after successful submission, without confirmation reads. Does not weaken validation or convert preview/error into acceptance. |
| `--workers`, `-w` | Existing positive explicit worker validation and bounded default worker selection. |
| `--no-worker-limit` | Existing behavior when workers is unset; do not introduce a second worker policy. |
| `--fail-fast` | Stop scheduling after failure; retain in-flight/accepted results and explicitly identify unstarted keys. |
| Confirmation and automation | Existing inherited `--auto-confirm` and `--automation` behavior. |
| Backoff flags | Register `addBackoffFlagsAndBindings(suspendCmd)`; preserve existing flag visibility and config precedence. CLI default backoff timeout is two minutes; positive config/flag overrides and earlier caller deadlines apply to confirmation. |
| `--timeout` | Existing per-request HTTP timeout; distinct from the confirmation bound. |
| Output/logging | Existing JSON, keys-only, quiet, verbose, debug, configured logging, and request-rate controls. |

No search selectors, `--force`, root escalation, `--no-state-check`, `--batch-size`, `--limit`, native batch, or resume command are added. Use `collectExplicitPIAdminInputOptions()` so discovery tenant filters do not rewrite the explicit selection; do not invent tenant evidence when no read occurred.

Validate all local arguments before mutation. Match recent mutation-command restrictions: JSON requires dry-run, auto-confirm, or automation; JSON plus verbose is invalid. These are validation errors, not opportunities to silently change modes. Otherwise effective result precedence remains JSON > keys-only > human, with quiet orthogonal. Preserve existing debug/quiet precedence rather than overriding it for this command.

## Ownership and dispatch

- `cmd/suspend.go`: parent command, shared backoff flag registration, help wiring.
- `cmd/suspend_processinstance.go`: Cobra construction, flags, local input/mode validation, metadata, and dispatch.
- `cmd/suspend_processinstance_batch.go`: collection execution lifecycle: preview, frozen scope, confirmation, service invocation, one-result dispatch, and exit. It owns no worker, retry, polling, or backend planning loops.
- `cmd/cmd_views_processinstance_suspend.go`: final human/JSON/keys-only views and bounded human failure formatting.

Use one collection path for one or many keys. Obtain a service dry-run preview, copy its ordered keys, return the preview if requested, otherwise confirm if eligible and execute the frozen keys with dry-run disabled. No preview output is emitted to machine stdout before an actual run. All entry paths preserve capability validation on 8.8/8.9, including dry-run.

Advertise state-changing mutation, full shared contract, full automation, and explicit human/JSON/keys-only output modes through command metadata helpers. Update capability/command contract tests and generated docs from source.

## Confirmation and prompts

Use the configured `cmd.ErrOrStderr()` with the existing confirmation helper and terminal-stdin policy. Redirected stdout does not suppress otherwise eligible confirmation. The question source is:

```text
You are about to suspend N process instance(s). Do you want to proceed?
```

Preserve the shared formatter's final question line, `[y/N]: ` suffix, default no, accepted answers, EOF handling, and local-precondition abort. Dry-run, invalid/empty input, unsupported capability, and implicit approval modes must not prompt. Approval freezes only target keys, not a claimed state. Decline submits nothing. Test real terminal stdin with stdout and stderr captured separately, including configured and inherited destinations.

## Result envelope

Use the existing envelope fields: `outcome`, optional `class`, `command`, optional `tenantContext`, `payload`, and optional `detail`. Payload is `SuspensionResults` with ordered `items`, for one key as well as many; no global schema changes. Preview uses the same typed items with status `preview`, submission `not_submitted`, and confirmation `not_requested`.

| Condition | Envelope selection |
| --- | --- |
| Valid dry-run, including no-wait | Explicit `renderSucceededResult`; never generic flag-derived accepted rendering. |
| All selected targets accepted, no-wait | `renderAcceptedResult`. |
| All selected targets confirmed in default mode | `renderSucceededResult`. |
| Any execution failure | `resultEnvelopeForError` with complete partial payload attached, then `renderResultEnvelope`; failure wins over no-wait. |
| Local validation, empty stdin, unsupported version, or abort before execution | Existing command-error envelope and classification, exactly once; no invented execution items. |
| Empty direct-library result rendered by an internal view test | Succeeded with `items: []`, never accepted solely due to no-wait. CLI still rejects missing input. |

Example confirmed result shape (illustrative key):

```json
{
  "outcome": "succeeded",
  "command": "suspend process-instance",
  "payload": {
    "items": [{
      "key": "2251799813685249",
      "status": "confirmed",
      "submission": "accepted",
      "confirmation": "confirmed",
      "statusCode": 204,
      "hasObservation": true,
      "lastObservedState": "SUSPENDED"
    }]
  }
}
```

Preserve the shared helper's actual canonical command-path convention, optional fields, and indentation rather than hardcoding this example. `suspendedDate` is included only when existing mapped evidence supplies it. Partial failures may include accepted/confirmed and failed items together; neither the error nor JSON rendering may discard those items.

## Keys-only and human output

Keys-only preview prints every selected preview key, in order. Keys-only actual execution prints only keys with `submission=accepted`, including accepted-but-unconfirmed targets, and retains the failure exit when confirmation fails. It never prints rejected, unknown-acceptance, or unsubmitted targets. No qualifying keys means exactly zero stdout bytes. Help must explicitly say that execution keys indicate acceptance, not confirmed suspension; use JSON or the exit/result contract for confirmation evidence.

Human mutation summaries and verbose outcome details use `renderHumanLine` and the configured command-context INFO logger. Human stdout is empty. Logger level, format, destination, and activity handling apply; do not embed timestamps or severity in message text. JSON envelopes and keys-only results remain on stdout; prompts remain plain configured stderr.

The preview message is:

```text
dry run: suspend process-instance; selected: N; eligibility not checked; no changes applied
```

Default waiting and no-wait messages respectively use:

```text
suspension: suspended C/N process instance(s)
suspension: submitted A/N process instance(s); confirmation not requested
```

C counts confirmed suspension; A counts accepted submissions. Default waiting appends `; accepted but unconfirmed U` when A exceeds C. Both modes append nonzero `; failed F` and `; not attempted S` counts. Accepted-but-unconfirmed work may also be failed; these facts are not disjoint outcome buckets. Omit zero detail counters. An empty execution view says `suspension: no process instances selected; no changes applied`, even with no-wait; missing CLI selection remains an error.

Quiet or a configured WARN/ERROR level suppresses INFO summaries and verbose rows, while explicitly selected JSON/keys output remains. Verbose rows allowlist submission, confirmation, and last-state facts; translate enum underscores to words without changing machine fields. Never concatenate raw errors or backend status strings. Keep keys out of default summaries.

Optional workflow/progress messages also use the configured logger. DEBUG HTTP diagnostics stay in the established transport with existing redaction; verbose alone does not enable them. Follow nearby mutation renderers and the mandatory human mutation summary rule in `AGENTS.md`.

## Failure path and exit

Trace the path as service cause plus typed partial result → facade conversion preserving both → mode-aware result renderer → configured compact aggregate diagnostic when appropriate → existing exit resolver.

- Before a result is rendered, use established bootstrap/command error helpers with a safe human message where needed.
- After partial results are rendered, non-JSON failure emits exactly one aggregate ERROR record through the command logger, then `exitAfterRenderedResult` with the original operation error. JSON already carries the error and gets no duplicate human failure record or second envelope.
- Group repeated causes using safe labels such as rejected, acceptance unknown, request not sent, confirmation failed, terminal observed, or timed out; include counts and a bounded reason. Do not log raw error strings or joined causes at normal/verbose levels.
- Writer failures get a bounded configured diagnostic and no second renderer. Preserve the operation error's exit when one already exists; otherwise use the rendering failure's established classification.
- Quiet must retain the compact error. `--no-err-codes` continues through existing `ResolveExitCode` behavior.

Existing exits remain: generic/local precondition 1, invalid arguments 2, not found 3, timeout 4, unavailable 5, conflict 6; unsupported and mixed errors use existing classification/precedence rather than new mappings. Read denial after acceptance retains the accepted item and permission error classification. Terminal-before-confirmation returns conflict-class failure with its observed state. Actual cancellation/deletion paths are unchanged.

## Documentation and fixture integration

Help, source metadata, README, and generated CLI docs must cover ACTIVE-only eligibility, Camunda 8.10, `SUSPEND_PROCESS_INSTANCE` on `PROCESS_DEFINITION` when authorization is enabled, read permission for default confirmation, exact-key scope, preview/no-wait/keys semantics, retained runtime state, independent relatives, in-flight workers, external effects, rejected job completion/failure during suspension, possible repeat job execution after resumption, and backpressure delays/rejection. Do not describe an instantaneous emergency stop.

Update only suspension setup in `integration/scripts/run-c810-suspended.sh` to use `c8 suspend pi --key "$key" --auto-confirm --backoff-timeout 60s --json`, capture its result, and check a successful confirmed payload. Preserve the subsequent `expect pi` scenario, configured target behavior, and existing scoped cleanup. Remove curl/auth setup only if it becomes unused in that script; do not replace other operations or broaden live scope. Validate the script statically with `bash -n`, never source or execute it during implementation.
