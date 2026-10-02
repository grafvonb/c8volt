---
title: "c8volt update user-task"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update user-task

Update user-task variables by key

### Synopsis

Update effective user-task variables on Camunda 8.8, 8.9, or 8.10. Camunda 8.7 returns an unsupported-version error without mutation.

Provide repeated --key values or newline-separated keys from stdin with '-'. Supply exactly one payload source: --vars with a JSON object or --vars-file with its file path. The same variable map is applied to every unique key. Explicit keys use backend authorization without tenant filtering.

c8volt reads each task's complete effective variables, preserves existing local or inherited scopes, creates absent names at the task's element scope, freezes one deduplicated plan, and waits for the requested values at those scopes.

Use --dry-run to inspect changes without mutation, or --auto-confirm for unattended updates. Use --no-wait to return after accepted scope writes without confirmation.

```
c8volt update user-task [-] [flags]
```

### Examples

```
  ./c8volt update user-task --key <user-task-key> --vars '{"approved":true}' --dry-run
  ./c8volt update ut --key <user-task-key> --vars-file ./vars.json --dry-run
  ./c8volt update uts --key <user-task-key-a> --key <user-task-key-b> --vars '{"approved":true}' --auto-confirm
  printf '%s\n' "$USER_TASK_KEY_A" "$USER_TASK_KEY_B" | ./c8volt update user-tasks - --vars '{"approved":true}' --dry-run
  ./c8volt --automation --json update ut --key <user-task-key> --vars '{"approved":true}'
```

### Options

```
      --dry-run            preview variable updates without submitting mutation
      --fail-fast          stop scheduling new scope updates after the first error
  -h, --help               help for user-task
  -k, --key strings        user task key(s) to update; repeat or combine with stdin '-'
      --no-wait            return after scope update requests are accepted without variable confirmation
      --no-worker-limit    use all queued scope updates as workers when --workers is unset
      --vars string        JSON object with variables to set for each user task
      --vars-file string   path to JSON object file with variables to set for each user task
  -w, --workers int        maximum concurrent workers when updating multiple user-task scopes (default: min(count, 2*GOMAXPROCS, 32))
```

### Options inherited from parent commands

```
      --all-tenants        clear configured tenant filtering and search all tenants visible to the authenticated user; mutually exclusive with --tenant
  -y, --auto-confirm       auto-confirm prompts for non-interactive use
      --automation         enable non-interactive mode for commands that explicitly support it
      --config string      path to config file
      --debug              enable debug logging
  -j, --json               output as JSON (where applicable)
      --keys-only          output keys only (where applicable)
      --log-level string   log level (debug, info, warn, error) (default "info")
      --no-indicator       disable transient terminal activity indicators
      --profile string     config active profile name to use (e.g. dev, prod)
  -q, --quiet              suppress output except errors
      --tenant string      tenant ID for discovery/search, selection, create, deploy, and run flows; explicit empty values can clear configured discovery filters, and explicit keys/IDs remain backend-authorized
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt update]({{ "/cli/c8volt_update" | relative_url }})	 - Update existing resources

