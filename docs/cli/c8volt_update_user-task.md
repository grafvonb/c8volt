---
title: "c8volt update user-task"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update user-task

Update variables, complete, assign or unassign user tasks by key

### Synopsis

Update variables, complete, assign, or unassign user tasks selected by key.

Use --set-vars or --set-vars-file for variables. Updates preserve the owning scope
of existing variables; new names are written to the task's element scope.
Use --complete with optional variables to complete a task; the backend response
confirms completion. Already completed tasks are skipped with a conflict.

Use --set-assignee to assign a task or --unassign to remove its assignee.
Replacing another assignee requires approval or --force; --auto-confirm alone
does not authorize replacement. Assignment modes cannot include variables.

Use --dry-run to preview. Completion and assignment await the backend response
and cannot be combined with --no-wait.

```
c8volt update user-task [-] [flags]
```

### Examples

```
  c8volt update user-task --key <user-task-key> --set-vars-file vars.json --dry-run
  c8volt update user-task --key <user-task-key> --complete --set-vars '{"approved":true}'
  c8volt update user-task --key <user-task-key> --set-assignee alice --dry-run
```

### Options

```
      --complete               complete selected tasks with optional variables; await backend confirmation
      --dry-run                preview variable updates, completions or assignment changes without submitting mutations
      --fail-fast              stop scheduling new task mutations or scope updates after the first error
      --force                  replace existing assignees without confirmation; requires --set-assignee
  -h, --help                   help for user-task
  -k, --key strings            user task key(s) to update; repeat or combine with stdin '-' (default [])
      --no-wait                skip variable-update verification; incompatible with --complete, --set-assignee or --unassign
      --no-worker-limit        use all queued task mutations or scope updates as workers when --workers is unset
      --set-assignee string    assign selected tasks to this backend identity; ask before replacement, or use --force
      --set-vars string        JSON object with variables to set or submit with each completion
      --set-vars-file string   path to JSON object file with variables to set or submit with each completion
      --unassign               remove the assignee from selected tasks
  -w, --workers int            maximum concurrent workers; omit for an automatic limit
```

### Options inherited from parent commands

```
      --all-tenants          search all accessible tenants; cannot combine with --tenant
  -y, --auto-confirm         auto-confirm prompts for non-interactive use
      --automation           enable non-interactive mode for commands that explicitly support it
      --config string        path to config file
      --debug                enable debug logging
  -j, --json                 output as JSON (where applicable)
      --keys-only            output keys only (where applicable)
      --log-level string     log level (debug, info, warn, error) (default "info")
      --no-indicator         disable transient terminal activity indicators
      --no-version-check     skip the automatic advisory gateway version check
      --profile string       config active profile name to use (e.g. dev, prod)
  -q, --quiet                suppress output except errors
      --request-rate float   maximum shared request starts per second (fractions allowed; 0 disables pacing)
      --tenant string        tenant for discovery and creation; explicit keys use backend authorization
      --timeout duration     HTTP request timeout (default 30s)
  -v, --verbose              show additional output
```

### SEE ALSO

* [c8volt update]({{ "/cli/c8volt_update" | relative_url }})	 - Update existing resources

