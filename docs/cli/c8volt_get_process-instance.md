---
title: "c8volt get process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get process-instance

List or fetch process instances

```
c8volt get process-instance [flags]
```

### Examples

```
  ./c8volt get pi --state active
  ./c8volt get pi --bpmn-process-id C88_SimpleUserTask_Process --state active
  ./c8volt get pi --bpmn-process-id C88_SimpleUserTask_Process --count 250
  ./c8volt get pi --state active --auto-confirm
  ./c8volt get pi --start-date-after 2026-01-01 --start-date-before 2026-01-31
		  ./c8volt get pi --start-date-older-days 7 --start-date-newer-days 30
  ./c8volt get pi --end-date-before 2026-03-31 --state completed
		  ./c8volt get pi --end-date-newer-days 14 --state completed
  ./c8volt get pi --key 2251799813711967 --key 2251799813711977
```

### Options

```
  -b, --bpmn-process-id string      BPMN process ID to filter process instances
      --children-only               show only child process instances, meaning instances that have a parent key set
  -n, --count int32                 number of process instances to fetch per page (max limit 1000 enforced by server) (default 1000)
      --end-date-after string       only include process instances with end date >= YYYY-MM-DD
      --end-date-before string      only include process instances with end date <= YYYY-MM-DD
      --end-date-newer-days int     only include process instances with end date N days old or newer (0 means today) (default -1)
      --end-date-older-days int     only include process instances with end date N days old or older (default -1)
      --fail-fast                   stop scheduling new instances after the first error
  -h, --help                        help for process-instance
      --incidents-only              show only process instances that have incidents
  -k, --key strings                 process instance key(s) to fetch
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
  -s, --state string                state to filter process instances: all, active, completed, canceled (default "all")
      --with-age                    include process instance age in one-line output and JSON meta
  -w, --workers int                 maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))
```

### Options inherited from parent commands

```
  -y, --auto-confirm               auto-confirm prompts for non-interactive use
      --backoff-max-retries int    max retry attempts (0 = unlimited)
      --backoff-timeout duration   overall timeout for the retry loop (default 2m0s)
      --config string              path to config file
      --debug                      enable debug logging, overwrites and is shorthand for --log-level=debug
  -j, --json                       output as JSON (where applicable)
      --keys-only                  output as keys only (where applicable), can be used for piping to other commands
      --log-format string          log format (json, plain, text) (default "plain")
      --log-level string           log level (debug, info, warn, error) (default "info")
      --log-with-source            include source file and line number in logs
      --no-err-codes               suppress error codes in error outputs
      --profile string             config active profile name to use (e.g. dev, prod)
  -q, --quiet                      suppress all output, except errors, overrides --log-level
      --tenant string              tenant ID for tenant-aware command flows (overrides env, profile, and base config)
  -v, --verbose                    adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt get](c8volt_get)	 - Get resources

