---
title: "c8volt run"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt run

Start state-changing work such as process instances

### Synopsis

Start state-changing work such as process instances.

Use this command family when you want c8volt to create new work in Camunda. Choose
`run process-instance` to start one or more instances. Child commands document
whether they wait for confirmed creation by default, when `--no-wait` can return
accepted work earlier, and how to pair the result with follow-up inspection commands.

```
c8volt run [flags]
```

### Examples

```
  ./c8volt run process-instance --help
  ./c8volt run process-instance --bpmn-process-id order-process
  ./c8volt --automation --json run process-instance --bpmn-process-id order-process --no-wait
```

### Options

```
      --backoff-max-retries int    max retry attempts (0 = unlimited)
      --backoff-timeout duration   overall timeout for the retry loop (default 2m0s)
  -h, --help                       help for run
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
* [c8volt run process-instance](c8volt_run_process-instance)	 - Start process instance(s) and confirm they are active

