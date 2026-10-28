---
title: "c8volt update"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update

Update existing resources

### Synopsis

Update element-local, process-instance or user-task variables, complete, assign or unassign user tasks, or update job retries, timeouts, and worker outcomes.

Element updates freeze exact local destinations before approval and confirm requested values unless --no-wait is selected. Dry-run and completed no-op plans never submit mutations.

User-task --complete accepts explicit keys and optional variables and previews locally with --dry-run. One lookup per key detects already completed tasks (conflict) and warns about missing assignees before confirmation. The backend's successful response confirms completion; c8volt adds no task-state polling or downstream waiting. For user tasks, --no-wait skips variable-update verification and is incompatible with --complete, which has no additional waiter to skip.

User-task --assign <assignee> and --unassign change explicit task assignments without reads or polling. Assignment protects existing assignees unless --allow-override is enabled. Dry-run previews intent locally; ordinary execution approves once and awaits the native response, including blocking listeners. Partial and uncertain outcomes remain visible. These modes reject variable payloads, each other, --complete and enabled --no-wait.

Requires Camunda 8.8 or newer. Use a resource subcommand to plan and confirm updates.

```
c8volt update [flags]
```

### Examples

```
  ./c8volt update element --key <element-key> --vars '{"approved":true}' --dry-run
  ./c8volt update process-instance --key <process-instance-key> --vars '{"customerTier":"gold"}' --dry-run
  ./c8volt update process-instance --key <process-instance-key> --vars-file ./vars.json --dry-run
  ./c8volt update user-task --key <user-task-key> --vars '{"approved":true}' --dry-run
  ./c8volt update user-task --key <user-task-key> --vars-file ./vars.json --auto-confirm
  ./c8volt update user-task --key 2251799813685249 --assign alice --dry-run
  ./c8volt update user-task --key 2251799813685249 --assign alice --allow-override --auto-confirm
  ./c8volt update user-task --key 2251799813685249 --unassign --auto-confirm
  ./c8volt update ut --key 2251799813685249 --complete --auto-confirm
  ./c8volt update job --key <job-key> --retries 3 --dry-run
  ./c8volt update job --key <job-key> --timeout 5m --auto-confirm
  printf '%s\n' "$PROCESS_INSTANCE_KEY_A" "$PROCESS_INSTANCE_KEY_B" | ./c8volt update process-instance - --vars '{"customerTier":"gold"}' --dry-run
  printf '%s\n' "$USER_TASK_KEY_A" "$USER_TASK_KEY_B" | ./c8volt update user-task - --vars '{"approved":true}' --dry-run
  ./c8volt --automation --json update process-instance --key <process-instance-key> --vars '{"customerTier":"gold"}' --dry-run
```

### Options

```
  -h, --help   help for update
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

* [c8volt]({{ "/cli/c8volt/" | relative_url }})	 - Operate Camunda 8 workflows from the command line
* [c8volt update element]({{ "/cli/c8volt_update_element" | relative_url }})	 - Update variables at explicit element scopes
* [c8volt update job]({{ "/cli/c8volt_update_job" | relative_url }})	 - Update a job or complete selected jobs
* [c8volt update process-instance]({{ "/cli/c8volt_update_process-instance" | relative_url }})	 - Update process-instance variables by key
* [c8volt update user-task]({{ "/cli/c8volt_update_user-task" | relative_url }})	 - Update variables, complete, assign or unassign user tasks by key

