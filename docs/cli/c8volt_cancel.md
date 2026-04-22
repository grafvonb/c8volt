---
title: "c8volt cancel"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt cancel

Cancel running work with explicit confirmation semantics

### Synopsis

Cancel running work with explicit confirmation semantics.

Use this command family when you need c8volt to stop active work in Camunda. Child
commands explain what gets validated before cancellation, when prompts appear, how
`--auto-confirm` enables unattended destructive flows, and when `--no-wait`
returns accepted cancellation before final completion is observed.

```
c8volt cancel [flags]
```

### Examples

```
  ./c8volt cancel process-instance --help
  ./c8volt cancel process-instance --key 2251799813711967
  ./c8volt cancel process-instance --state active --count 200 --auto-confirm --no-wait
```

### Options

```
      --backoff-max-retries int    max retry attempts (0 = unlimited)
      --backoff-timeout duration   overall timeout for the retry loop (default 2m0s)
  -h, --help                       help for cancel
```

### Options inherited from parent commands

```
  -y, --auto-confirm        auto-confirm prompts for non-interactive use
      --automation          enable the canonical non-interactive contract for commands that explicitly support it
      --config string       path to config file
      --debug               enable debug logging, overwrites and is shorthand for --log-level=debug
  -j, --json                output as JSON (where applicable)
      --keys-only           output as keys only (where applicable), can be used for piping to other commands
      --log-format string   log format (json, plain, text) (default "plain")
      --log-level string    log level (debug, info, warn, error) (default "info")
      --log-with-source     include source file and line number in logs
      --no-err-codes        suppress error codes in error outputs
      --no-indicator        disable transient terminal activity indicators
      --profile string      config active profile name to use (e.g. dev, prod)
  -q, --quiet               suppress all output, except errors, overrides --log-level
      --tenant string       tenant ID for tenant-aware command flows (overrides env, profile, and base config)
  -v, --verbose             adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt](c8volt)	 - Operate Camunda 8 with guided help and script-safe output modes
* [c8volt cancel process-instance](c8volt_cancel_process-instance)	 - Cancel process instance(s) by key or search filters and wait for completion

