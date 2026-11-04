---
title: "c8volt get process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get process-instance

List or fetch process instances

### Synopsis

Inspect process instances by key or search by definition, state, incidents,
variables, or time range. Search filters combine with AND.

Use --bid for one exact business ID (only c8.10+); commas and wildcards are literal.
Add --with-incidents for incident details, --with-vars for root variables, or
--with-elements for runtime elements. Add --with-listeners to --with-elements
for listener jobs.

Use --limit to bound search results or --total to count matches. Explicit keys
cannot be combined with search filters.

```
c8volt get process-instance [flags]
```

### Examples

```
  c8volt get process-instance --key <process-instance-key> --with-incidents
  c8volt get process-instance --bpmn-process-id order --state active --limit 10
  c8volt get process-instance --bid ORDER-123 --json
```

### Options

```
  -n, --batch-size int32                process instances per discovery page (server maximum 1000) (default 1000)
      --bid string                      exact business ID to filter by (only c8.10+); commas, spaces and wildcards are literal
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
  -l, --limit int32                     maximum matching instances to return; omit for all matches
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
  -s, --state string                    state to filter process instances: all, active, suspended (only c8.10+), completed, canceled, terminated (default "all")
      --total                           return only the numeric total of matching process instances; capped backend totals are counted by paging
      --var stringArray                 require variable equality or advanced clause(s); repeat or separate clauses with commas
      --var-exists stringArray          require variable name(s) to exist; repeat or separate names with commas
      --var-like stringArray            require variable value pattern clause(s); repeat or separate clauses with commas
      --var-value-limit int             maximum characters to show for variable values when --with-vars is set; 0 disables truncation
      --with-elements                   include runtime element instances for keyed or list/search process-instance output
      --with-incidents                  include direct incident keys, states, and messages for keyed or list/search process-instance output
      --with-listeners                  include runtime listener jobs; requires --with-elements
      --with-vars                       include process-instance-scope variables for keyed or list/search process-instance output
  -w, --workers int                     maximum concurrent workers; omit for an automatic limit
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

