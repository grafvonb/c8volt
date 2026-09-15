---
title: "c8volt get user-task"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get user-task

Fetch or search native user tasks

### Synopsis

Get native Camunda user tasks by key or search criteria.

Provide repeated or comma-separated --key values, or newline-separated keys on stdin. A trailing '-' explicitly selects stdin; nonterminal stdin is also detected without it. Each unique key is fetched once in first-input order.

Every requested key must resolve or the command fails without a partial result. Keyed reads require Camunda 8.8 or newer and use backend authorization without discovery-tenant filtering, so --tenant does not hide an authorized task and the task's actual tenant is returned.

Without keys, search by process, element, state, assignment, candidate, and effective tenant scope. Predicates are combined with AND. Supported states are ASSIGNING, CANCELED, CANCELING, COMPLETED, COMPLETING, CREATED, CREATING, FAILED, and UPDATING; state matching is case-insensitive, and all applies no state predicate. --batch-size controls each discovery request, --limit caps returned tasks across all pages, and --total emits the exact matching count.

Interactive searches offer another page separately from command results when more matches remain. Use --auto-confirm or --automation for unattended paging. --quiet suppresses human results but preserves explicitly requested JSON, keys-only, and numeric total output.

Human rows show task key, tenant, element ID, and state, followed by related pi:, ei:, and pd: keys. Assignee is always last: assignee:<user> when assigned, otherwise assignee:<unassigned>. Task name, BPMN process ID, and process-definition version are available in JSON. Other empty optional fields are omitted.

Use --json for one collection envelope or --keys-only for one task key per line. Keys cannot be combined with search filters, --limit, or --total; --total also conflicts with --limit, --json, and --keys-only. Search and keyed reads require Camunda 8.8, 8.9, or 8.10; Camunda 8.7 is unsupported. Task mutations, variables, forms, audit history, date filters, custom sorting, and watch mode are not provided by this command.

```
c8volt get user-task [-] [flags]
```

### Examples

```
  ./c8volt get user-task --key <user-task-key>
  ./c8volt get ut -k <user-task-key>,<another-user-task-key>
  ./c8volt get user-task --state created --assignee alice --limit 25
  ./c8volt get user-task --candidate-group accounting --total
  ./c8volt --automation --keys-only get user-task --batch-size 100
  printf '%s\n' "$USER_TASK_KEY" | ./c8volt get user-tasks
  printf '%s\n' "$USER_TASK_KEY" | ./c8volt get uts -
  ./c8volt --json get user-task --key <user-task-key>
  ./c8volt --keys-only get user-task --key <user-task-key>
```

### Options

```
      --assignee string          exact assignee to filter in search mode
  -n, --batch-size int32         number of user tasks to request per page; does not cap total results (maximum 1000) (default 1000)
  -b, --bpmn-process-id string   BPMN process ID to filter in search mode
      --candidate-group string   exact candidate-group membership to filter in search mode
      --candidate-user string    exact candidate-user membership to filter in search mode
      --element-id string        BPMN task element ID to filter in search mode
      --fail-fast                stop scheduling new user task reads after the first error
  -h, --help                     help for user-task
  -k, --key strings              user task key(s) to fetch; repeat, comma-separate, or combine with stdin
  -l, --limit int32              maximum number of matching user tasks to return across all pages; omit for unlimited
      --no-worker-limit          use all queued user task reads as workers when --workers is unset
      --pd-key string            process definition key to filter in search mode
      --pi-key string            process instance key to filter in search mode
  -s, --state string             user task state to filter in search mode; case-insensitive; all disables the predicate (default "all")
      --total                    return only the exact numeric total of matching user tasks
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

