---
title: "c8volt get process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get process-instance

List or fetch process instances

### Synopsis

Get process instances by key or by search criteria.

Use direct lookup when you know a process-instance key, or combine search filters to inspect matching process instances by process definition, tenant, state, incidents, variables, jobs, user tasks, and time ranges.

Search results support interactive paging, scriptable JSON aggregation, and count-only workflows. Direct key lookup stays strict: missing keys return not-found.

Tenant contract: --tenant scopes search/list discovery and selector validation where supported. Explicit --key and stdin keys are backend-authorized admin input; c8volt displays returned tenant metadata without rejecting solely because it differs from the selected tenant.

When --bpmn-process-id is set, c8volt validates that the process definition is visible before searching process instances. A missing selector fails with a local diagnostic instead of looking like a valid empty result; --json, --automation, --keys-only, and non-TTY runs never prompt for recovery output.

Use --with-incidents to include direct incident details under matching process-instance rows in keyed or list/search output.

Use --with-vars to include process-instance-scope variables under matching process-instance rows in keyed or list/search output.

Use --with-elements to include runtime element instances under matching process-instance rows. Nested human element rows include dur:<duration> when start/end timestamps or active state support a runtime duration.

Use variable-search flags to narrow list/search results natively on Camunda 8.8 and 8.9; Camunda 8.7 returns an unsupported-version error for those flags. --var-exists requires every listed variable name to exist. --var accepts name=value equality shorthand plus advanced name.$operator=value clauses for $eq, $neq, $exists, $in, $notIn, and $like; $notin is accepted as $notIn. --var-like uses native wildcard patterns: * matches zero or more characters, ? matches one character, and escaped wildcards remain literal. Commas inside quoted values and JSON arrays stay inside the variable clause. Variable scopeKey means the scope where the variable is directly defined.

Use --has-user-tasks to fetch process instances by their owning user-task keys.

Run `c8volt get pi --help` for the complete flag reference.

```
c8volt get process-instance [flags]
```

### Examples

```
  ./c8volt get pi --bpmn-process-id <bpmn-process-id> --state active --limit 5
  ./c8volt get pi --key <process-instance-key>
  ./c8volt get pi --state active --total
  ./c8volt get pi --has-user-tasks <user-task-key>
  ./c8volt get pi --incidents-only --with-incidents --limit 5
  ./c8volt get pi --direct-incidents-only --incident-error-type io_mapping_error --incident-error-message intentional --limit 5
  ./c8volt get pi --var-exists payload,email --limit 5
  ./c8volt get pi --var 'status="approved"' --limit 5
  ./c8volt get pi --var 'status.$in=["approved","pending"]' --limit 5
  ./c8volt get pi --var-like 'email=*@example.com,customerId=CUST-????' --limit 5
  ./c8volt get pi --state active --with-vars --var-value-limit 120 --limit 5
  ./c8volt get pi --key <process-instance-key> --with-incidents
  ./c8volt get pi --key <process-instance-key> --with-vars
  ./c8volt get pi --key <process-instance-key> --with-vars --var-value-limit 120
  ./c8volt get pi --key <process-instance-key> --with-elements
  ./c8volt get pi --start-date-after 2026-05-01 --start-date-before 2026-05-31 --limit 5
  ./c8volt get pi --key <process-instance-key> --key <another-process-instance-key>
```

### Options

```
  -n, --batch-size int32                number of process instances to fetch per page (max limit 1000 enforced by server) (default 1000)
  -b, --bpmn-process-id string          BPMN process ID to filter process instances
      --children-only                   show only child process instances
      --direct-incidents-only           show only process instances with direct incident details
      --end-date-after string           only include process instances with end date >= YYYY-MM-DD
      --end-date-before string          only include process instances with end date <= YYYY-MM-DD
      --end-date-newer-days int         only include process instances with end date N days old or newer (0 means today) (default -1)
      --end-date-older-days int         only include process instances with end date N days old or older (default -1)
      --fail-fast                       stop scheduling new instances after the first error
      --has-user-tasks strings          user task key(s) whose owning process instances should be fetched
  -h, --help                            help for process-instance
      --incident-error-message string   case-insensitive incident error message substring filter for keyed --with-incidents or list/search --direct-incidents-only
      --incident-error-type string      case-insensitive incident error type filter for keyed --with-incidents or list/search --direct-incidents-only
      --incident-message-limit int      maximum characters to show for incident messages when --with-incidents is set; 0 disables truncation
      --incident-state string           incident state scope for keyed --with-incidents: active, pending, resolved, migrated, unknown, all (default "active")
      --incidents-only                  show only process instances that have incidents
  -k, --key strings                     process instance key(s) to fetch
  -l, --limit int32                     maximum number of matching process instances to return or process across all pages
      --no-incidents-only               show only process instances that have no incidents
      --no-worker-limit                 use all queued jobs as workers when --workers is unset
      --orphan-children-only            show only child instances with missing parents
      --parent-key string               parent process instance key to filter process instances
      --pd-key string                   process definition key (mutually exclusive with bpmn-process-id, pd-version, and pd-version-tag)
      --pd-version int32                process definition version
      --pd-version-tag string           process definition version tag
      --roots-only                      show only root process instances
      --start-date-after string         only include process instances with start date >= YYYY-MM-DD
      --start-date-before string        only include process instances with start date <= YYYY-MM-DD
      --start-date-newer-days int       only include process instances N days old or newer (0 means today) (default -1)
      --start-date-older-days int       only include process instances N days old or older (default -1)
  -s, --state string                    state to filter process instances: all, active, completed, canceled, terminated (default "all")
      --total                           return only the numeric total of matching process instances; capped backend totals are counted by paging
      --var stringArray                 require variable equality or advanced clause(s); repeat or separate clauses with commas
      --var-exists stringArray          require variable name(s) to exist; repeat or separate names with commas
      --var-like stringArray            require variable value pattern clause(s); repeat or separate clauses with commas
      --var-value-limit int             maximum characters to show for variable values when --with-vars is set; 0 disables truncation
      --with-elements                   include runtime element instances for keyed or list/search process-instance output
      --with-incidents                  include direct incident keys, states, and messages for keyed or list/search process-instance output
      --with-vars                       include process-instance-scope variables for keyed or list/search process-instance output
  -w, --workers int                     maximum concurrent workers when --batch-size > 1 (default: min(batch-size, 2*GOMAXPROCS, 32))
```

### Options inherited from parent commands

```
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
      --tenant string      tenant ID for discovery/search, selection, create, deploy, and run flows; explicit keys/IDs remain backend-authorized
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt get](c8volt_get)	 - Inspect cluster, process, job, element, incident, tenant, and resource state

