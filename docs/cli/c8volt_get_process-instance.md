---
title: "c8volt get process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get process-instance

List or fetch process instances

### Synopsis

List process instances by search filters or fetch them by key.
Use this command to inspect workflow instances by key, process-definition selectors, state, or date filters.

Use --total when you only need the numeric count of matching process instances. Direct --key lookups stay strict: if the requested process instance is missing, c8volt returns the normal not-found error.

When search results span multiple pages, human-oriented output prompts before continuing unless --auto-confirm or --json is set. JSON mode consumes remaining pages and returns one aggregated result.

```
c8volt get process-instance [flags]
```

### Examples

```
  ./c8volt get pi --state active
  ./c8volt get pi --state active --total
  ./c8volt get pi --bpmn-process-id C88_SimpleUserTask_Process --state active
  ./c8volt get pi --bpmn-process-id C88_SimpleUserTask_Process --batch-size 250
  ./c8volt get pi --state active --batch-size 250 --limit 25
  ./c8volt get pi --state active --auto-confirm
  ./c8volt --json get pi --state active --batch-size 250
  ./c8volt get pi --key 2251799813711967 --json
  ./c8volt get pi --start-date-after 2026-01-01 --start-date-before 2026-01-31
  ./c8volt get pi --start-date-older-days 7 --start-date-newer-days 30
  ./c8volt get pi --end-date-before 2026-03-31 --state completed
  ./c8volt get pi --end-date-newer-days 14 --state completed
  ./c8volt get pi --key 2251799813711967 --key 2251799813711977
```

### Options

```
  -n, --batch-size int32            number of process instances to fetch per page (max limit 1000 enforced by server) (default 1000)
  -b, --bpmn-process-id string      BPMN process ID to filter process instances
      --children-only               show only child process instances, meaning instances that have a parent key set
      --end-date-after string       only include process instances with end date >= YYYY-MM-DD
      --end-date-before string      only include process instances with end date <= YYYY-MM-DD
      --end-date-newer-days int     only include process instances with end date N days old or newer (0 means today) (default -1)
      --end-date-older-days int     only include process instances with end date N days old or older (default -1)
      --fail-fast                   stop scheduling new instances after the first error
  -h, --help                        help for process-instance
      --incidents-only              show only process instances that have incidents
  -k, --key strings                 process instance key(s) to fetch
  -l, --limit int32                 maximum number of matching process instances to return or process across all pages
      --no-incidents-only           show only process instances that have no incidents
      --no-worker-limit             disable limiting the number of workers to GOMAXPROCS when --workers > 1
      --orphan-children-only        show only child instances where parent key is set but the parent process instance does not exist (anymore)
      --parent-key string           parent process instance key to filter process instances
      --pd-key string               process definition key (mutually exclusive with bpmn-process-id, pd-version, and pd-version-tag)
      --pd-version int32            process definition version
      --pd-version-tag string       process definition version tag
      --roots-only                  show only root process instances, meaning instances with empty parent key
      --start-date-after string     only include process instances with start date >= YYYY-MM-DD
      --start-date-before string    only include process instances with start date <= YYYY-MM-DD
      --start-date-newer-days int   only include process instances N days old or newer (0 means today) (default -1)
      --start-date-older-days int   only include process instances N days old or older (default -1)
  -s, --state string                state to filter process instances: all, active, completed, canceled, terminated (default "all")
      --total                       return only the numeric total of matching process instances; capped backend totals stay lower bounds
      --with-age                    include process instance age in one-line output and JSON meta
  -w, --workers int                 maximum concurrent workers when --batch-size > 1 (default: min(batch-size, GOMAXPROCS))
```

### Options inherited from parent commands

```
  -y, --auto-confirm       auto-confirm prompts for non-interactive use
      --automation         enable non-interactive mode for commands that explicitly support it
      --config string      path to config file
      --debug              enable debug logging, overwrites and is shorthand for --log-level=debug
  -j, --json               output as JSON (where applicable)
      --keys-only          output as keys only (where applicable), can be used for piping to other commands
      --log-level string   log level (debug, info, warn, error) (default "info")
      --no-indicator       disable transient terminal activity indicators
      --profile string     config active profile name to use (e.g. dev, prod)
  -q, --quiet              suppress all output, except errors, overrides --log-level
      --tenant string      tenant ID for tenant-aware command flows (overrides env, profile, and base config)
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt get](c8volt_get)	 - Inspect cluster, process, and resource state

