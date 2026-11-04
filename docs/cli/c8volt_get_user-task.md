---
title: "c8volt get user-task"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get user-task

Fetch or search native user tasks

### Synopsis

Inspect native user tasks by key or search by state, assignee, candidate group,
process instance, or variables. Search filters combine with AND.

Use --bid for one exact task business ID (only c8.10+). Variable filters match
local variables; --with-vars retrieves effective variables, including inherited
values. Use --limit to bound search results or --total to count matches.

Repeat --key or read newline-separated keys from stdin with '-'. Explicit keys
cannot be combined with search filters.

```
c8volt get user-task [-] [flags]
```

### Examples

```
  c8volt get user-task --key <user-task-key> --with-vars
  c8volt get user-task --state created --assignee alice --limit 10
  c8volt get user-task --candidate-group accounting --total
```

### Options

```
      --assignee string          exact assignee to filter in search mode
  -n, --batch-size int32         number of user tasks to request per page; does not cap total results (maximum 1000) (default 1000)
      --bid string               exact stored task business ID (only c8.10+); commas, spaces and wildcards are literal
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

* [c8volt get]({{ "/cli/c8volt_get" | relative_url }})	 - Inspect cluster, process, job, element, incident, tenant, and resource state

