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

Use variable-search flags to narrow native searches by local variables on each task. --var-exists requires every listed local variable name to exist. --var accepts name=value equality shorthand plus advanced name.$operator=value clauses for $eq, $neq, $exists, $in, $notIn, and $like; $notin is accepted as $notIn. Values use the existing process-instance filter encoding: quote strings and use JSON string arrays for membership operators. --var-like uses native wildcard patterns: * matches zero or more characters, ? matches one character, and escaped wildcards remain literal. Commas inside quoted values and JSON arrays stay inside the variable clause. Variable clauses and ordinary search predicates are combined with AND. Parent-scope values do not satisfy local filters; negative and existence matching retain backend semantics.

Interactive searches offer another page separately from command results when more matches remain. Use --auto-confirm or --automation for unattended paging. --quiet suppresses human results but preserves explicitly requested JSON, keys-only, and numeric total output.

Human rows show task key, tenant, element ID, and state, followed by related pi:, ei:, and pd: keys. Assignee is always last: assignee:<user> when assigned, otherwise assignee:<unassigned>. Task name, BPMN process ID, and process-definition version are available in JSON. Other empty optional fields are omitted.

Filtering does not retrieve variables. Add --with-vars independently to retrieve the effective variables selected by the backend for each returned task on Camunda 8.8, 8.9, or 8.10. Human output nests variables beneath their task. --var-value-limit sets a nonnegative Unicode-character limit after structured values are compacted; zero, the default, keeps full received values. Truncation labels distinguish backend-incomplete values from display shortening. JSON always preserves received values and backend truncation metadata. Effective keys-only and --total output skip variable retrieval.

Use --json for one collection envelope or --keys-only for one task key per line. Keys cannot be combined with search filters, --limit, or --total; --total also conflicts with --limit, --json, and --keys-only. Search and keyed reads require Camunda 8.8, 8.9, or 8.10; Camunda 8.7 is unsupported. Variable mutation, task mutations, forms, audit history, date filters, custom sorting, and watch mode are not provided by this command.

```
c8volt get user-task [-] [flags]
```

### Examples

```
  ./c8volt get ut --var 'status="approved"'
  ./c8volt get ut --var-exists payload
  ./c8volt get ut --var-like 'email=*@example.com'
  ./c8volt get ut --assignee alice --var 'status="approved"' --limit 20
  ./c8volt get ut --var 'status="approved"' --total
  ./c8volt get ut --var 'status="approved"' --with-vars
  ./c8volt get ut --key <user-task-key> --with-vars
  ./c8volt get ut --assignee alice --limit 10 --with-vars
  ./c8volt get ut --pi-key <process-instance-key> --with-vars --var-value-limit 120
  ./c8volt --json get ut --key <user-task-key> --with-vars
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
      --var stringArray          require local variable equality or advanced clause(s); repeat or separate clauses with commas
      --var-exists stringArray   require local variable name(s) to exist; repeat or separate names with commas
      --var-like stringArray     require local variable value pattern clause(s); repeat or separate clauses with commas
      --var-value-limit int      maximum characters to show for variable values when --with-vars is set; 0 disables truncation
      --with-vars                include effective variables for selected user tasks
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

