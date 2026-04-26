---
title: "c8volt cancel process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt cancel process-instance

Cancel process instances by key or filters

### Synopsis

Cancel process instances by key or search filters.

By default c8volt validates the affected root and descendant instances, asks for confirmation, and waits until cancellation is observed. Use --force when a selected child must be escalated to its root instance.

Use --dry-run to preview the resolved scope without submitting cancellation, prompting for confirmation, or waiting for completion.

Use --auto-confirm for unattended destructive runs. Add --no-wait when accepted cancellation is enough for the current step, then verify later with `get pi` or `expect pi`.

```
c8volt cancel process-instance [flags]
```

### Examples

```
  ./c8volt cancel pi --key <process-instance-key>
  ./c8volt cancel pi --key <process-instance-key> --dry-run
  ./c8volt cancel pi --key <process-instance-key> --force
  ./c8volt cancel pi --state active --batch-size 250
  ./c8volt cancel pi --state active --batch-size 250 --limit 25
  ./c8volt cancel pi --state active --batch-size 250 --limit 25 --dry-run
  ./c8volt cancel pi --state active --start-date-before 2026-03-31
  ./c8volt cancel pi --state active --start-date-newer-days 30
  ./c8volt cancel pi --bpmn-process-id C88_SimpleUserTask_Process --state active --batch-size 200 --auto-confirm
  ./c8volt cancel pi --state active --batch-size 200 --auto-confirm --no-wait
  ./c8volt expect pi --key <process-instance-key> --state canceled
  ./c8volt get pi --key <process-instance-key> --keys-only | ./c8volt cancel pi --auto-confirm --no-wait -
```

### Options

```
  -n, --batch-size int32            number of process instances to process per page (max limit 1000 enforced by server) (default 1000)
  -b, --bpmn-process-id string      BPMN process ID to filter process instances
      --dry-run                     preview which process instances would be canceled without submitting cancellation
      --end-date-after string       only include process instances with end date >= YYYY-MM-DD
      --end-date-before string      only include process instances with end date <= YYYY-MM-DD
      --end-date-newer-days int     only include process instances with end date N days old or newer (0 means today) (default -1)
      --end-date-older-days int     only include process instances with end date N days old or older (default -1)
      --fail-fast                   stop scheduling new instances after the first error
      --force                       force cancellation of the root process instance if a process instance is a child, including all its child instances
  -h, --help                        help for process-instance
  -k, --key strings                 process instance key(s) to cancel
  -l, --limit int32                 maximum number of matching process instances to process across all pages
      --no-state-check              skip checking the current state of the process instance before cancelling it
      --no-wait                     skip waiting for the cancellation to be fully processed
      --no-worker-limit             disable limiting the number of workers to GOMAXPROCS when --workers > 1
      --pd-version int32            process definition version
      --pd-version-tag string       process definition version tag
      --start-date-after string     only include process instances with start date >= YYYY-MM-DD
      --start-date-before string    only include process instances with start date <= YYYY-MM-DD
      --start-date-newer-days int   only include process instances N days old or newer (0 means today) (default -1)
      --start-date-older-days int   only include process instances N days old or older (default -1)
  -s, --state string                state to filter process instances: all, active, completed, canceled, terminated (default "all")
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

* [c8volt cancel](c8volt_cancel)	 - Cancel running process instances

