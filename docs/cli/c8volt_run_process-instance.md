---
title: "c8volt run process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt run process-instance

Start process instances and confirm creation

### Synopsis

Start process instances and confirm creation.

Run by BPMN process ID for the latest version, or by process definition key for an exact definition.

When running by BPMN process ID, c8volt validates all requested process definitions before creating anything. Mixed visible and missing BPMN IDs fail as one request, so no partial process instances are started; automation-oriented modes never prompt for recovery output.

By default c8volt waits until created instances are observable. Created instances are confirmed after Camunda observes ACTIVE, COMPLETED, CANCELED, or TERMINATED.

Use --keys-only to pipe created process instance keys into strict lifecycle checks with expect pi.

```
c8volt run process-instance [flags]
```

### Examples

```
  ./c8volt run pi -b <bpmn-process-id>
  ./c8volt run pi -b <bpmn-process-id> --vars '{"customerId":"1234"}'
  ./c8volt run pi -b <bpmn-process-id> -n 3 --workers 2
  ./c8volt --json run pi -b <bpmn-process-id> --vars '{"customerId":"1234"}'
  ./c8volt run pi -b <bpmn-process-id> --keys-only | ./c8volt expect pi --state completed -
  ./c8volt run pi -b <long-running-bpmn-process-id> --keys-only | ./c8volt expect pi --state active -
```

### Options

```
  -b, --bpmn-process-id strings   BPMN process ID(s) to run process instance for (mutually exclusive with --pd-key). Runs latest version unless --pd-version is specified
  -n, --count int                 number of instances to start for a single process definition (default 1)
      --fail-fast                 stop scheduling new instances after the first error
  -h, --help                      help for process-instance
      --no-wait                   return after creation is accepted
      --no-worker-limit           use all queued jobs as workers when --workers is unset
      --pd-key strings            specific process definition key(s) to run process instance for (mutually exclusive with --bpmn-process-id)
      --pd-version int32          specific version of the process definition to use when running by BPMN process ID (supported only with --bpmn-process-id)
      --vars string               JSON-encoded variables to pass to the started process instance(s)
  -w, --workers int               maximum concurrent workers when --count > 1 (default: min(count, 2*GOMAXPROCS, 32))
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

* [c8volt run](c8volt_run)	 - Start process instances

