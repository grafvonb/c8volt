---
title: "c8volt get user-task"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get user-task

Fetch native user tasks by key

### Synopsis

Get native Camunda user tasks by key.

Provide repeated or comma-separated --key values, or newline-separated keys on stdin. A trailing '-' explicitly selects stdin; nonterminal stdin is also detected without it. Each unique key is fetched once in first-input order.

Every requested key must resolve or the command fails without a partial result. Keyed reads require Camunda 8.8 or newer and use backend authorization without discovery-tenant filtering, so --tenant does not hide an authorized task and the task's actual tenant is returned.

Use --json for one collection envelope or --keys-only for one task key per line. Search, filtering, limits, and totals are not yet available; their reserved flags cannot be combined with keys.

```
c8volt get user-task [-] [flags]
```

### Examples

```
  ./c8volt get user-task --key <user-task-key>
  ./c8volt get ut -k <user-task-key>,<another-user-task-key>
  printf '%s\n' "$USER_TASK_KEY" | ./c8volt get user-tasks
  printf '%s\n' "$USER_TASK_KEY" | ./c8volt get uts -
  ./c8volt --json get user-task --key <user-task-key>
  ./c8volt --keys-only get user-task --key <user-task-key>
```

### Options

```
      --assignee string          reserved for exact assignee search; not yet available
  -n, --batch-size int32         reserved search page size (maximum 1000); not yet available (default 1000)
  -b, --bpmn-process-id string   reserved for search by BPMN process ID; not yet available
      --candidate-group string   reserved for candidate-group search; not yet available
      --candidate-user string    reserved for candidate-user search; not yet available
      --element-id string        reserved for search by BPMN task element ID; not yet available
      --fail-fast                stop scheduling new user task reads after the first error
  -h, --help                     help for user-task
  -k, --key strings              user task key(s) to fetch; repeat, comma-separate, or combine with stdin
  -l, --limit int32              reserved search result limit; not yet available
      --no-worker-limit          use all queued user task reads as workers when --workers is unset
      --pd-key string            reserved for search by process definition key; not yet available
      --pi-key string            reserved for search by process instance key; not yet available
  -s, --state string             reserved for case-insensitive state search; not yet available (default "all")
      --total                    reserved exact search count; not yet available
  -w, --workers int              maximum concurrent workers when fetching multiple user tasks
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

* [c8volt get]({{ "/cli/c8volt_get" | relative_url }})	 - Inspect cluster, process, job, element, incident, tenant, and resource state
