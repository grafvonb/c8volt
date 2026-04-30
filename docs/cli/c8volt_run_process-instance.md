---
title: "c8volt run process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt run process-instance

Start process instances and confirm activation

### Synopsis

Start process instances and confirm activation.

Run by BPMN process ID for the latest version, or by process definition key for an exact definition.

By default c8volt waits for active instances. Add --no-wait to verify later with `get pi`, `expect pi`, or `walk pi`.

```
c8volt run process-instance [flags]
```

### Examples

```
  ./c8volt run pi -b C88_SimpleUserTask_Process
  ./c8volt run pi -b C88_SimpleUserTask_Process --vars '{"customerId":"1234"}'
  ./c8volt run pi -b C88_SimpleUserTask_Process -n 100 --workers 8
  ./c8volt --json run pi -b C88_SimpleUserTask_Process --no-wait
  ./c8volt expect pi --key <process-instance-key> --state active
```

### Options

```
  -b, --bpmn-process-id strings   BPMN process ID(s) to run process instance for (mutually exclusive with --pd-key). Runs latest version unless --pd-version is specified
  -n, --count int                 number of instances to start for a single process definition (default 1)
      --fail-fast                 stop scheduling new instances after the first error
  -h, --help                      help for process-instance
      --no-wait                   return after creation is accepted
      --no-worker-limit           disable limiting the number of workers to GOMAXPROCS when --workers > 1
      --pd-key strings            specific process definition key(s) to run process instance for (mutually exclusive with --bpmn-process-id)
      --pd-version int32          specific version of the process definition to use when running by BPMN process ID (supported only with --bpmn-process-id)
      --vars string               JSON-encoded variables to pass to the started process instance(s)
  -w, --workers int               maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))
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
      --tenant string      tenant ID for tenant-aware command flows (overrides env, profile, and base config)
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt run](c8volt_run)	 - Start process instances

