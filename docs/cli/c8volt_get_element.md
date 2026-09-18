---
title: "c8volt get element"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get element

List or fetch runtime element instances

### Synopsis

List or fetch Camunda runtime element instances.

Use --key for a known element instance. Otherwise search by process instance, BPMN element ID, state, type, process definition, or BPMN process ID.

--batch-size controls each discovery request; --limit caps returned elements across all pages. Use --total to count matching elements, or --with-listeners to include runtime listener jobs.

Listener rows label job creation time as s: (not worker execution start) and job end time as e:. They show d: only for an ACTIVATED job with a deadline, and omit unavailable timestamps. Listener dur: measures time from job creation to its recorded end, or elapsed time for a pending job; it includes worker waiting time and is omitted when timestamps are unavailable or invalid. A completed listener can appear as:

  job-1 TASK_LISTENER lsnr:CREATING COMPLETED tp:updateTaskData r:0 s:2026-09-16T13:07:16.359 e:2026-09-16T13:07:16.842 dur:483ms

Requires Camunda 8.8 or newer.

```
c8volt get element [flags]
```

### Examples

```
  ./c8volt get element --key <element-instance-key>
  ./c8volt get element --key <element-instance-key> --with-listeners
  ./c8volt get element --pi-key <process-instance-key> --limit 10
  ./c8volt get element --pi-key <process-instance-key> --with-listeners
  ./c8volt get element --pi-key <process-instance-key> --total
  ./c8volt --json get element --pi-key <process-instance-key> --limit 5
  ./c8volt --json get element --key <element-instance-key> --with-listeners
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
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt get]({{ "/cli/c8volt_get" | relative_url }})	 - Inspect cluster, process, job, element, incident, tenant, and resource state

