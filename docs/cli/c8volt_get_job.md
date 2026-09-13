---
title: "c8volt get job"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get job

Inspect or search jobs

### Synopsis

Inspect or search Camunda jobs.

Use --key for a known job. Otherwise search by state, type, process instance, element instance, BPMN element, worker, retries, job kind, or listener event type.

--batch-size controls each discovery request; --limit caps jobs across all pages; --total counts matching jobs.

Requires Camunda 8.8 or newer.

```
c8volt get job [flags]
```

### Examples

```
  ./c8volt get job --key <job-key>
  ./c8volt get job --state failed --batch-size 10 --limit 50
  ./c8volt get job --state failed --total
  ./c8volt --json get job --key <job-key>
```

### Options

```
  -n, --batch-size int32              number of jobs to request per page; does not cap total results (max limit 1000 enforced by server) (default 1000)
      --element-id string             BPMN element ID to filter in search mode
      --element-instance-key string   element instance key to filter in search mode
      --error-message-limit int       maximum characters to show for error messages; 0 keeps full messages
  -h, --help                          help for job
  -k, --key string                    job key for exact lookup; omit to list or search jobs
      --kind string                   Camunda job kind to filter in search mode; case-insensitive
  -l, --limit int32                   maximum number of matching jobs to return across all pages; omit to continue through all matches
      --listener-event-type string    listener event type to filter in search mode; case-insensitive
      --pi-key string                 process instance key to filter in search mode
      --retries int32                 exact retry count to filter in search mode
  -s, --state string                  Camunda job state to filter in search mode; case-insensitive
      --total                         return only the numeric total of matching jobs
      --type string                   job type to filter in search mode
      --worker string                 worker name to filter in search mode
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

