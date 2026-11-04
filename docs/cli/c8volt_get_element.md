---
title: "c8volt get element"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get element

List or fetch runtime element instances

### Synopsis

Inspect runtime element instances by key or process context.

Use --pi-key to inspect a process instance's elements. Add --with-vars for
variables owned directly by each element, or --with-listeners for listener jobs.
Parent-scope variables are excluded. Listener duration includes worker waiting time.

```
c8volt get element [flags]
```

### Examples

```
  c8volt get element --pi-key <process-instance-key> --limit 10
  c8volt get element --key <element-instance-key> --with-vars
  c8volt get element --pi-key <process-instance-key> --with-listeners
```

### Options

```
  -n, --batch-size int32         number of elements to request per page; does not cap total results (max limit 1000 enforced by server) (default 1000)
  -b, --bpmn-process-id string   BPMN process ID to filter in search mode
      --element-id string        BPMN element ID to filter in search mode
  -h, --help                     help for element
  -k, --key string               element instance key for exact lookup; omit to list or search runtime elements
  -l, --limit int32              maximum number of matching elements to return across all pages; omit to continue through all matches
      --pd-key string            process definition key to filter in search mode
      --pi-key string            process instance key to filter in search mode
  -s, --state string             runtime element state to filter in search mode; case-insensitive
      --total                    return only the numeric total of matching elements
      --type string              runtime element type to filter in search mode; case-insensitive
      --with-listeners           include runtime listener jobs
      --with-vars                include complete element-local variables; skipped for keys-only and total output
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

* [c8volt get]({{ "/cli/c8volt_get" | relative_url }})	 - Inspect cluster, process, job, element, incident, tenant, and resource state

