---
title: "c8volt get job"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get job

Inspect or search jobs

### Synopsis

Inspect a job by key or search by state, type, process instance, worker, or retries.

Use --retries 0 to find jobs with no retries remaining. Combine filters with AND.
Use --limit to bound results or --total to count matches.

```
c8volt get job [flags]
```

### Examples

```
  c8volt get job --key <job-key>
  c8volt get job --state failed --retries 0 --limit 10
  c8volt get job --pi-key <process-instance-key> --json
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

