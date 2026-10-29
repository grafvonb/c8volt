# CLI Contract: User-Task Assignment and Unassignment

**Feature**: [spec.md](../spec.md) | **Models**: [data-model.md](../data-model.md)

This is the intended new contract, not evidence of shipped implementation. Existing completion and variable-update payloads remain unchanged.

## Command and mode selection

Canonical path: `update user-task`; aliases: `user-tasks`, `ut`, `uts`.

```text
c8volt update user-task [-] --key <key>[,<key>...] --assign <assignee> [--allow-override]
c8volt update user-task [-] --key <key>[,<key>...] --unassign
```

`--key`/`-k` may repeat or be omitted when stdin supplies keys through positional `-`. Only `-` is an allowed positional argument. At least one valid selected key is mandatory. Keys from flags precede stdin lines; duplicates retain their first position. Each key is exactly 16 ASCII digits after the existing key whitespace normalization. Explicit empty flag occurrences/components are errors; blank stdin lines are ignored, but empty overall selection is an error.

| Flag | Selection/validation rule |
| --- | --- |
| `--assign <assignee>` | Presence selects assignment; empty/whitespace fails; preserve nonblank opaque identity |
| `--unassign` | True selects unassignment; false is inactive |
| `--complete` | True selects existing completion; false is inactive |
| `--allow-override` | Presence allowed only with assignment, including explicit false; value controls policy |
| `--vars`, `--vars-file` | Any presence is incompatible with assign/unassign, even empty |
| `--no-wait` | Enabled value incompatible with assign/unassign; false does not enable it |
| `--workers`/`-w` | Explicit value must be positive; unset uses existing automatic worker selection |
| `--no-worker-limit` | Existing policy when explicit worker count is absent |
| `--fail-fast` | Stop new scheduling after observed failure; retain submitted outcomes |
| `--dry-run` | Local intent preview with no backend requests or confirmation |

Multiple active mutation modes are invalid. With no active selector, existing variable mode remains unchanged. `--allow-override` alone is invalid; it does not select assignment. Auto-confirm, automation, generic force settings and tenant configuration never imply override.

Existing JSON execution confirmation guard applies: use dry-run, auto-confirm or automation. JSON plus verbose remains rejected. Output precedence stays JSON, then keys-only, then human; quiet does not suppress explicitly selected machine results. Existing timeout, authentication, tenant and no-error-code conventions remain applicable.

## Approval and request contract

Locally freeze action, unique keys, assignee and override before one eligible confirmation. Do not read task state or perform discovery. Approval identifies assignment/unassignment and the selected scope; assignment also identifies the opaque assignee and whether reassignment is allowed. Identity text must be safely escaped for plain output rather than inserting control characters as interactive text; the backend identity value itself stays unchanged.

Examples of intended confirmation wording:

```text
You are about to assign user-task 2251799813685249 to "alice" (reassignment: not allowed). Do you want to proceed?
You are about to assign 2 user tasks to "alice" (reassignment: allowed). Do you want to proceed?
You are about to unassign user-task 2251799813685249. Do you want to proceed?
```

Ordinary single-task execution shows the confirmation followed by the result, without a duplicate plan or tenant-scope log. Plan and tenant-scope details require `--verbose`, including tenant context for dry-run. For bulk approval/previews, identify every selected key using the existing detail conventions and list the unique selection; the compact prompt may use the count. Prompts use plain `cmd.ErrOrStderr()` and retain existing default-no, yes parsing, EOF, decline, nonterminal and stdout-redirection behavior. No prompt for dry-run, invalid input, auto-confirm or automation. Do not add completion advisory warnings.

Assignment sends explicit false/true override in the same native operation as the assignee. Unassignment sends only the normal native unassignment operation. Await the documented response, including blocking task listeners. No read-then-unassign-then-assign, additional polling, read-back or downstream waiting. Missing/current/same-assignee state is decided by the backend, never by suppressed errors.

## Shared envelope

Use existing `ResultEnvelope` without adding/changing shared fields:

- Required `outcome` and canonical `command: "update user-task"`.
- Existing optional `class`, `tenantContext`, `payload`, `detail`.
- Detail retains required `message` and optional `class`/`suggestion`.
- Local preview: `outcome: "succeeded"`.
- Successful execution: `outcome: "accepted"`, meaning a successful native response was observed.
- Validation/operation error: existing classified `invalid`/`failed` envelope; attach operation payload only when meaningful execution evidence exists.
- Emit exactly once and require EOF after decoding. Never call another error renderer after a partial-result envelope.

For assignment reports, `assignee` and `allowOverride` are both present, even when override is false. For unassignment, both are omitted, not null or empty placeholders. Required report arrays are initialized. All count fields are present, including zero.

Example successful assignment payload within the unchanged envelope (optional tenant context omitted here):

```json
{
  "outcome": "accepted",
  "command": "update user-task",
  "payload": {
    "operation": "assign",
    "phase": "execution",
    "requestedKeys": ["2251799813685249"],
    "requestedCount": 1,
    "assignee": "alice",
    "allowOverride": false,
    "items": [{"key": "2251799813685249", "status": "accepted", "submission": "accepted", "statusCode": 204}],
    "counts": {"preview": 0, "accepted": 1, "failed": 0, "timedOut": 0, "canceled": 0, "notAttempted": 0}
  }
}
```

Example local unassignment preview:

```json
{
  "outcome": "succeeded",
  "command": "update user-task",
  "payload": {
    "operation": "unassign",
    "phase": "preview",
    "requestedKeys": ["2251799813685249"],
    "requestedCount": 1,
    "items": [{"key": "2251799813685249", "status": "preview", "submission": "not_submitted"}],
    "counts": {"preview": 1, "accepted": 0, "failed": 0, "timedOut": 0, "canceled": 0, "notAttempted": 0}
  }
}
```

Per-item enums and omission rules are defined in [data-model.md](../data-model.md). Counts sum to requestedCount for complete reports. Retain raw per-item machine error evidence when present, but never use it directly in human formatting. A failed overall invocation can retain accepted items; a timeout/cancellation does not establish rejection. Last statusCode alone cannot override earlier unknown submission.

## Human and keys-only output

Human stdout is empty. Use the configured command-context INFO logger for plans, dry-run summaries, successful summaries and verbose outcome details, with existing format/level/destination/activity behavior. Quiet and WARN suppress INFO. Message text has no hand-added timestamp or severity prefix.

Intended single-result messages:

```text
updated user-task 2251799813685249: assignment confirmed
updated user-task 2251799813685249: unassignment confirmed
```

Bulk summaries identify action and requested count and omit zero human counters. Previews identify action/keys, assignment identity and permission, ending with `no requests submitted`; they never promise task existence, permission or success. Verbose rows describe safe per-resource outcomes, not raw HTTP chains.

Keys-only emits each unique preview key during dry-run and only successful-response keys during execution, in selection order. It never prints failed, uncertain or unattempted keys as successful; no qualifying key means zero bytes, not a blank line. Partial failure preserves its non-success exit even with successful keys on stdout.

Normal/verbose-only failure uses one bounded aggregate ERROR, grouping repeated safe reasons with counts. Keep at most three distinct groups plus a count for other failures. Preserve uncertain-outcome wording. Quiet retains the error. Do not emit a second error diagnostic for a JSON envelope that already carries the failure. DEBUG uses existing redacted diagnostics, independent of verbose, and never result stdout.

## Errors and exits

Use existing local invalid-input/precondition handling, `ferrors.FromDomain`, envelope classification and `exitAfterRenderedResult`; do not invent new numeric exits. Preserve mixed-error precedence and configured no-error-code behavior. Invalid input has no fabricated operation report. Decline/EOF preserves existing abort behavior.

After partial results are rendered, retain the original operation error for exit, never render again. A rendering failure gets one compact logger error; if an operation also failed, its class/exit retains precedence. This does not authorize raw error text at any human output boundary.

## Acceptance coverage

Exercise actual command execution with independent stdout/stderr/logger captures: every input form and alias; assignment true/false override; empty and false flags; local preview with forbidden backend; all/partial failures; cancellation; quiet+machine; automation and auto-confirm; exact JSON+EOF; keys-only zero-byte cases; INFO/WARN suppression; grouped errors; both DEBUG entry points; plain and structured formats; inherited/configured destinations; and real-terminal approve/decline/EOF/redirect cases. Preserve completion/variable/read regression coverage. All dependencies must be isolated test-owned fakes.
