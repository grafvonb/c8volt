# CLI Contract: Update User-Task Variables

## Grammar and input

```text
c8volt update user-task [aliases: ut, uts, user-tasks] [-]
  --key/-k <key[,key...]>                 repeatable; merge with stdin '-'
  (--vars <JSON object> | --vars-file <path>)
  [--dry-run] [--no-wait]
  [--workers/-w <positive integer>] [--no-worker-limit] [--fail-fast]
```

Inherit existing JSON, keys-only, quiet, verbose, automation, auto-confirm, configuration, tenant, debug and backoff controls. Do not add task-search or scope-selection flags. A dash is the only positional input. Match PI stdin detection, validation timing, default worker policy, and flag precedence; do not import the broader get-UT implicit search behavior. Missing keys fail locally; empty payload `{}` with valid keys is a no-op after successful planning.

Inline/file payloads are equivalent. Reject absent/both sources, unreadable files, invalid JSON, arrays, scalars and root null. Property null remains a value. Reuse the PI parser and error classifications through a small shared extraction if needed.

JSON plus verbose is invalid, matching PI. JSON mutation requires dry-run, auto-confirm or automation. These restrictions are validated before remote discovery, including requests that might later prove unchanged.

## Scope semantics

Existing names update the exact effective scope returned by `get ut --with-vars`. Missing names are added at the task's element-instance scope. Identical inherited values remain unchanged. Shared scope/name changes are deduplicated. Preview and confirmation retain task-level counts and describe inherited targets without discovering other affected tasks.

## Human output

Use the PI rhythm and variable tokens, replacing only the resource wording:

```text
dry run: update user-task variables: N user task(s), C change(s), A addition(s), U unchanged, T untouched; no changes applied
plan: update user-task variables: N user task(s), C change(s), A addition(s), U unchanged, T untouched
```

Use existing tenant context before applicable human plans. Variable details reuse `~ name: before -> after`, `+ name: value`, and unchanged/untouched conventions, adding scope identity only where needed to explain inherited effects. Keep single-task and verbose details aligned with PI. For multiple tasks, compact inherited-scope evidence must remain visible even when detailed values are hidden; do not add endpoint/request/cursor chatter.

Confirmation uses the established question with resource substitution:

```text
You are about to update N requested variable value(s) on M user task(s). Do you want to proceed?
```

Counts refer to task-variable changes and changed tasks, consistent with the preview; shared physical writes do not change the displayed counts. The prompt goes to `cmd.ErrOrStderr()` with existing default/no/EOF handling. Preserve terminal-stdin eligibility when stdout is redirected. Dry-run/no-op/auto-confirm/automation never prompt.

Retain PI confirmed/submitted/mutation-failed/confirmation-failed wording and summary structure for changed tasks. Include partial/unfinished status truthfully; verbose details may list scopes. Do not report accepted work as fully confirmed or unchanged.

## JSON and keys

Register state-changing mutation, full shared-contract support and full automation support. Use stable structs and the existing envelope schema; do not alter shared schemas.

- Preview/no-op payload follows the PI preview shape: `operation`, `requestedKeys`, `requestedCount`, `updateCount`, `variableAddCount`, `variableChangeCount`, `variableUnchangedCount`, `variableUntouchedCount`, `userTasks,omitempty`, `mutationSubmitted`. Per-task plans carry `userTaskKey` instead of PI identity and variable scope evidence. Retain PI null/empty-array conventions for category slices. No-op has `mutationSubmitted=false` and creates no execution reports.
- Execution payload is `items,omitempty`, with per-task result fields and scope outcomes described in [data model](../data-model.md). Per-task order matches unique input order.
- Ordinary confirmed execution, dry-run, and successful no-op use outcome `succeeded`; no-wait with accepted writes uses `accepted`. No-op remains succeeded even with no-wait.
- Failures use the normalized existing error outcome/class/detail. When partial results exist, place them in that same envelope's payload; render once, then use the existing post-render exit path. Do not emit success and subsequently a second error envelope.
- JSON stdout contains exactly one envelope and EOF. No prompts, tenant prose, progress, or diagnostics leak into it.
- Keys-only preview lists changed task keys once; execution lists successfully confirmed/submitted changed task keys once. Failed/skipped/unchanged tasks produce no result key. Overall failure is still reflected by the existing exit/error path. Empty/no-op produces zero bytes.
- Quiet suppresses human output only; explicitly selected JSON/keys survive. Preserve existing JSON-over-keys precedence. Dry-run never claims submission.

Rendering never issues backend requests. Writer failure must not trigger a second result rendering attempt.

## Validation matrix

Cover inline/file, canonical/alias, repeated/comma/stdin keys; all-supported-version scopes; malformed payloads/keys/workers; local, inherited, new and unchanged values; shared targets; sparse pages; truncated values; dry-run/normal/no-wait; human/JSON/keys/quiet and quiet+machine combinations; auto-confirm/automation; abort/EOF; configured/inherited stderr with real terminal stdin; unauthorized/missing tasks; failures before and after acceptance; fail-fast/cancellation and PI regression behavior.
