---
title: "c8volt delete process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt delete process-instance

Delete process instance(s) by key or search filters, optionally cancelling first

```
c8volt delete process-instance [flags]
```

### Examples

```
  ./c8volt delete pi --key 2251799813711967 --force
  ./c8volt delete pi --state completed --count 250
  ./c8volt delete pi --state completed --end-date-after 2026-01-01 --end-date-before 2026-01-31 --auto-confirm
  ./c8volt delete pi --state completed --end-date-older-days 7 --end-date-newer-days 60 --auto-confirm
  ./c8volt delete pi --bpmn-process-id order-process --start-date-after 2026-01-01 --start-date-before 2026-01-31 --auto-confirm
  ./c8volt delete pi --bpmn-process-id order-process --state completed --count 200 --auto-confirm
  ./c8volt delete pi --state active --start-date-newer-days 30 --auto-confirm
  ./c8volt get pi --state completed --keys-only | ./c8volt delete pi - --auto-confirm
```

### Options

```
  -b, --bpmn-process-id string      BPMN process ID to filter process instances
  -n, --count int32                 number of process instances to process per page (max limit 1000 enforced by server) (default 1000)
      --end-date-after string       only include process instances with end date >= YYYY-MM-DD
      --end-date-before string      only include process instances with end date <= YYYY-MM-DD
      --end-date-newer-days int     only include process instances with end date N days old or newer (0 means today) (default -1)
      --end-date-older-days int     only include process instances with end date N days old or older (default -1)
      --fail-fast                   stop scheduling new instances after the first error
      --force                       force cancellation of the process instance(s), prior to deletion
  -h, --help                        help for process-instance
  -k, --key strings                 process instance key(s) to delete
      --no-state-check              skip checking the current state of the process instance before deleting it
      --no-wait                     skip waiting for the deletion to be fully processed
      --no-worker-limit             disable limiting the number of workers to GOMAXPROCS when --workers > 1
      --pd-version int32            process definition version
      --pd-version-tag string       process definition version tag
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

* [c8volt delete](c8volt_delete)	 - Delete resources

