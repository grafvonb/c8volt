---
title: "c8volt run process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt run process-instance

Start process instances and confirm creation

### Synopsis

Start process instances and confirm creation.

Use a BPMN process ID for the latest version or a process-definition key for an exact definition. All requested BPMN IDs must be visible before any instance is started.

Creation uses the configured tenant, or the default tenant when none is configured. --all-tenants is not supported because creation requires one destination tenant.

By default c8volt waits until created instances are observable as ACTIVE, COMPLETED, CANCELED, or TERMINATED.

```
c8volt run process-instance [flags]
```

### Examples

```
  ./c8volt run process-instance --bpmn-process-id <bpmn-process-id>
  ./c8volt --tenant tenant-a run process-instance --bpmn-process-id <bpmn-process-id>
  ./c8volt run process-instance --bpmn-process-id <bpmn-process-id> --vars '{"customerId":"1234"}'
  ./c8volt run process-instance --bpmn-process-id <bpmn-process-id> --count 3 --workers 2
  ./c8volt --json run process-instance --bpmn-process-id <bpmn-process-id> --vars '{"customerId":"1234"}'
  ./c8volt run process-instance --bpmn-process-id <bpmn-process-id> --keys-only | ./c8volt expect process-instance --state completed -
  ./c8volt run process-instance --bpmn-process-id <long-running-bpmn-process-id> --keys-only | ./c8volt expect process-instance --state active -
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
      --all-tenants        clear configured tenant filtering and search all tenants visible to the authenticated user; mutually exclusive with --tenant
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
      --tenant string      tenant ID for discovery/search, selection, create, deploy, and run flows; explicit empty values can clear configured discovery filters, and explicit keys/IDs remain backend-authorized
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output and API request diagnostics on stderr
```

### SEE ALSO

* [c8volt run]({{ "/cli/c8volt_run" | relative_url }})	 - Start process instances

