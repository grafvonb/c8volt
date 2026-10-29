---
title: "c8volt update user-task"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update user-task

Update variables, complete, assign or unassign user tasks by key

### Synopsis

Update variables, complete, assign or unassign explicitly selected user tasks on supported Camunda versions.

Use --assign <assignee> to assign every selected task to the same backend identity, or --unassign to remove its assignee. Assignment, unassignment and completion are mutually exclusive. Assignment and unassignment reject --vars, --vars-file and enabled --no-wait. Without an active mode selector, the variable-update workflow applies.

Assignment accepts repeated/comma-separated --key values and newline-separated stdin with positional '-'. Keys must be valid 16-digit values; duplicates retain first-input order. Without --force, c8volt reads each unique task once before approval. If another user owns a task, c8volt warns and asks whether to replace the assignee. Answering Yes authorizes replacement at execution time. Tasks already assigned to the requested identity finish successfully without a prompt or assignment request; the message says no change is needed. Batches count them as unchanged and exclude them from confirmation. Tasks observed unassigned keep backend replacement protection. Failed lookups stop before approval or mutation; there is no reread after approval.

Use --force with --assign to permit replacement without confirmation or preliminary reads. --force requires --assign. --auto-confirm, --automation and nonterminal input do not grant replacement permission: already-assigned tasks are skipped with a conflict and a hint to use --force. With --fail-fast, an unapproved ownership conflict prevents all assignment requests. Interactive approval covers the entire selected batch once, including any replacements shown in the warnings.

--dry-run previews keys, action, identity and replacement policy locally with zero requests or prompts, including with --force. Unassignment performs no preliminary reads. --workers bounds lookups and native requests; --fail-fast stops new scheduling after failures while preserving in-flight results. Both operations await the native response, including blocking task listeners, with no polling or read-back. Partial failures retain ordered outcomes; timeouts and cancellation may leave acceptance unknown.

Human summaries use the configured INFO logger; reassignment warnings use WARN and confirmation questions remain plain text. Plan and tenant-scope details require --verbose. Quiet and configured levels suppress informational messages as usual. JSON emits one envelope with partial evidence and original failure exits; keys-only prints preview keys or successfully submitted keys only. Entirely unchanged selections produce a succeeded JSON envelope with unchanged/not_submitted items and no keys-only output. Quiet preserves machine output. JSON assignment requires --force, --auto-confirm, --automation or --dry-run and cannot combine with --verbose. Other user-task modes retain their existing JSON confirmation rules. HTTP diagnostics require DEBUG independently of verbose.

Use --complete to submit native completion for each eligible unique key, with optional --vars or --vars-file data. Omitted variables and {} do not suppress completion. Completion preserves JSON numbers exactly, including large integers and precise decimals. The backend's successful response confirms completion, including blocking task listeners. c8volt performs no additional task-state polling or downstream waiting; no assignment is performed. --no-wait skips variable-update verification and cannot bypass the completion request or its listener processing, so it is incompatible with --complete. A configured tenant does not filter explicit keys.

Before confirmation, c8volt reads each unique task once. Already completed tasks are skipped with a conflict result; any supplied variables are not applied. Human summaries omit zero counters; conflict errors explain that the task is already completed and no completion request was sent. Variables are mentioned only when supplied. Tasks with no assignee produce a warning and remain eligible. Lookup failures produce a warning and leave eligibility to the backend. There is no recheck after approval. Ordinary human approval shows selection scope, plan, warnings and confirmation in that order. A single-task prompt names the key; a bulk prompt names the eligible count. A single success says "updated user-task <key>: completion confirmed"; batches use "update user-task" with nonzero counts. Completion summaries and verbose details honor the configured log level; quiet suppresses them. Auto-confirm and automation skip the approval plan; quiet and machine output suppress its informational messages.

Completion accepts repeated or comma-separated --key values and newline-separated keys with positional '-'. Keys must contain exactly 16 decimal digits. --dry-run previews the frozen selection locally without requests or prompts. Ordinary execution confirms the eligible selection once, with no prompt when none can be submitted; --auto-confirm and --automation allow unattended execution. --workers bounds concurrency and --fail-fast stops new scheduling after a failure while retaining in-flight outcomes. With --fail-fast, finding an already completed task during preparation prevents all completion submissions. Without --fail-fast, completed tasks are skipped and eligible tasks continue.

Without an active --assign, --unassign or --complete selector, variable-update behavior is unchanged; false boolean selectors are inactive.

Provide repeated --key values or newline-separated keys from stdin with '-'. Supply exactly one payload source: --vars with a JSON object or --vars-file with its file path. The same variable map is applied to every unique key. Explicit keys use backend authorization without tenant filtering.

c8volt reads each task's complete effective variables, preserves existing local or inherited scopes, creates absent names at the task's element scope, freezes one deduplicated plan, and waits for the requested values at those scopes. Detailed previews label local writes as element:<key> and inherited writes as pi:<key> or element:<key> according to their owning scope.

Use --dry-run to inspect changes without mutation, or --auto-confirm for unattended updates. Use --no-wait to return after accepted scope writes without confirmation.

```
c8volt update user-task [-] [flags]
```

### Examples

```
  ./c8volt update user-task --key 2251799813685249 --assign alice --dry-run
  ./c8volt update user-task --key 2251799813685249,2251799813685250 --assign alice --force
  ./c8volt update user-task --key 2251799813685249 --unassign --auto-confirm
  printf '%s\n' "$USER_TASK_KEY_A" "$USER_TASK_KEY_B" | ./c8volt update user-task - --assign alice --workers 1 --automation --json
  ./c8volt update ut --key 2251799813685249 --complete
  ./c8volt update ut --key 2251799813685249,2251799813685250 --complete --vars '{"approved":true}' --dry-run
  ./c8volt update ut --key 2251799813685249 --complete --vars-file ./vars.json --auto-confirm
  printf '%s\n' "$USER_TASK_KEY_A" "$USER_TASK_KEY_B" | ./c8volt update ut - --complete --workers 1 --auto-confirm
  ./c8volt update user-task --key <user-task-key> --vars '{"approved":true}' --dry-run
  ./c8volt update ut --key <user-task-key> --vars-file ./vars.json --dry-run
  ./c8volt update uts --key <user-task-key-a> --key <user-task-key-b> --vars '{"approved":true}' --auto-confirm
  printf '%s\n' "$USER_TASK_KEY_A" "$USER_TASK_KEY_B" | ./c8volt update user-tasks - --vars '{"approved":true}' --dry-run
  ./c8volt --automation --json update ut --key <user-task-key> --vars '{"approved":true}'
```

### Options

```
      --assign string      assign selected tasks to this backend identity; ask before replacement, or use --force
      --complete           complete selected tasks with optional variables; report backend-confirmed completion without additional polling
      --dry-run            preview variable updates, completions or assignment changes without submitting mutations
      --fail-fast          stop scheduling new task mutations or scope updates after the first error
      --force              replace existing assignees without confirmation; requires --assign
  -h, --help               help for user-task
  -k, --key strings        user task key(s) to update; repeat or combine with stdin '-' (default [])
      --no-wait            skip variable-update verification after accepted scope writes; native task mutations have no additional polling, so incompatible with --complete, --assign or --unassign
      --no-worker-limit    use all queued task mutations or scope updates as workers when --workers is unset
      --unassign           remove the assignee from selected tasks
      --vars string        JSON object with variables to set or submit with each completion
      --vars-file string   path to JSON object file with variables to set or submit with each completion
  -w, --workers int        maximum concurrent workers for task mutations or variable scope updates (default: min(count, 2*GOMAXPROCS, 32))
```

### Options inherited from parent commands

```
      --all-tenants          clear configured tenant filtering and search all tenants visible to the authenticated user; mutually exclusive with --tenant
  -y, --auto-confirm         auto-confirm prompts for non-interactive use
      --automation           enable non-interactive mode for commands that explicitly support it
      --config string        path to config file
      --debug                enable debug logging
  -j, --json                 output as JSON (where applicable)
      --keys-only            output keys only (where applicable)
      --log-level string     log level (debug, info, warn, error) (default "info")
      --no-indicator         disable transient terminal activity indicators
      --profile string       config active profile name to use (e.g. dev, prod)
  -q, --quiet                suppress output except errors
      --request-rate float   maximum shared request starts per second (fractions allowed; 0 disables pacing)
      --tenant string        tenant ID for discovery/search, selection, create, deploy, and run flows; explicit empty values can clear configured discovery filters, and explicit keys/IDs remain backend-authorized
      --timeout duration     HTTP request timeout (default 30s)
  -v, --verbose              show additional output
```

### SEE ALSO

* [c8volt update]({{ "/cli/c8volt_update" | relative_url }})	 - Update existing resources

