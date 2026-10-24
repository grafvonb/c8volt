---
title: "c8volt update user-task"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update user-task

Update variables or complete user tasks by key

### Synopsis

Update effective user-task variables or complete explicitly selected user tasks on supported Camunda versions.

Use --complete to submit native completion for every unique key, with optional --vars or --vars-file data. Omitted variables and {} still submit completion. Requests are accepted without independent task-state verification or downstream waiting; no assignment is performed. --complete cannot be combined with --no-wait. A configured tenant does not filter explicit keys.

Completion accepts repeated or comma-separated --key values and newline-separated keys with positional '-'. Keys must contain exactly 16 decimal digits. --dry-run previews the frozen selection locally without requests or prompts. Ordinary execution confirms the entire selection once; --auto-confirm and --automation allow unattended execution. --workers bounds concurrency and --fail-fast stops new scheduling after a failure while retaining in-flight outcomes.

Without --complete (or with --complete=false), variable-update behavior is unchanged.

Provide repeated --key values or newline-separated keys from stdin with '-'. Supply exactly one payload source: --vars with a JSON object or --vars-file with its file path. The same variable map is applied to every unique key. Explicit keys use backend authorization without tenant filtering.

c8volt reads each task's complete effective variables, preserves existing local or inherited scopes, creates absent names at the task's element scope, freezes one deduplicated plan, and waits for the requested values at those scopes. Detailed previews label local writes as element:<key> and inherited writes as pi:<key> or element:<key> according to their owning scope.

Use --dry-run to inspect changes without mutation, or --auto-confirm for unattended updates. Use --no-wait to return after accepted scope writes without confirmation.

```
c8volt update user-task [-] [flags]
```

### Examples

```
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
      --complete           complete selected tasks with optional variables; report acceptance without verification
      --dry-run            preview variable updates or task completions without submitting mutations
      --fail-fast          stop scheduling new completions or scope updates after the first error
  -h, --help               help for user-task
  -k, --key strings        user task key(s) to update; repeat or combine with stdin '-' (default [])
      --no-wait            variable mode only: return after scope writes are accepted without confirmation; incompatible with --complete
      --no-worker-limit    use all queued completions or scope updates as workers when --workers is unset
      --vars string        JSON object with variables to set or submit with each completion
      --vars-file string   path to JSON object file with variables to set or submit with each completion
  -w, --workers int        maximum concurrent workers for task completions or variable scope updates (default: min(count, 2*GOMAXPROCS, 32))
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

