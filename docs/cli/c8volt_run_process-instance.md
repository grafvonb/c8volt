---
title: "c8volt run process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt run process-instance

Start process instances and confirm creation

### Synopsis

Start process instances and wait until they are visible.

Use --bpmn-process-id for the latest version or --pd-key for an exact definition.
Supply initial variables with --set-vars and create multiple instances with
--count. Creation uses one configured tenant.

--set-bid assigns a native business ID at creation (only c8.10+). The same ID is
used for every --count attempt; backend uniqueness rules still apply.
Creation confirmation does not mean the process has completed.

```
c8volt run process-instance [flags]
```

### Examples

```
  c8volt run process-instance --bpmn-process-id order
  c8volt run process-instance --bpmn-process-id order --set-vars '{"customerId":"1234"}'
  c8volt run process-instance --bpmn-process-id order --set-bid ORDER-123
```

### Options

```
  -b, --bpmn-process-id strings   BPMN process IDs to start; latest version unless --pd-version is set
  -n, --count int                 number of instances to start for a single process definition (default 1)
      --fail-fast                 stop scheduling new instances after the first error
  -h, --help                      help for process-instance
      --no-wait                   return after creation is accepted
      --no-worker-limit           use all queued jobs as workers when --workers is unset
      --pd-key strings            exact definition keys to start; cannot combine with --bpmn-process-id
      --pd-version int32          definition version to start; requires --bpmn-process-id
      --set-bid string            business ID to assign at creation (only c8.10+)
      --set-vars string           JSON-encoded variables to pass to the started process instance(s)
  -w, --workers int               maximum concurrent workers; omit for an automatic limit
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

* [c8volt run]({{ "/cli/c8volt_run" | relative_url }})	 - Start process instances

