---
title: "c8volt update job"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update job

Update a job by key

### Synopsis

Update a Camunda job by key on Camunda 8.8 or newer.

Supports retry and timeout updates and worker outcomes. Explicit --key uses backend authorization without tenant filtering.

c8volt plans the update and asks for confirmation before material interactive mutations. Retry updates are verified by reading the job; timeout updates and worker outcomes return after acceptance without waiting for confirmation.

Use --dry-run to inspect the plan without mutation. With --json, mutations require --dry-run, --auto-confirm, or --automation; --json cannot be combined with --verbose.

```
c8volt update job [flags]
```

### Examples

```
  ./c8volt update job --key <job-key> --retries 3 --dry-run
  ./c8volt --tenant tenant-a update job --key <job-key> --retries 3 --dry-run
  ./c8volt update job --key <job-key> --retries 3 --auto-confirm
  ./c8volt update job --key <job-key> --timeout 5m --auto-confirm
  ./c8volt update job --key <job-key> --fail --retries 0 --message "worker unavailable" --dry-run
  ./c8volt update job --key <job-key> --throw-bpmn-error PAYMENT_DECLINED --message "card declined" --dry-run
  ./c8volt update job --key <job-key> --complete --vars '{"approved":true}' --dry-run
  ./c8volt --json update job --key <job-key> --retries 3 --dry-run
```

### Options

```
      --complete                  complete the job through the worker outcome API
      --dry-run                   preview job updates without submitting mutation
      --fail                      report a technical job failure
  -h, --help                      help for job
  -k, --key string                job key to update
      --message string            operator message for worker outcome modes
      --no-wait                   return after the update request is accepted without retry confirmation
      --retries int32             retry count to set, or remaining retries for --fail
      --retry-backoff string      duration before a failed job becomes retryable, for example 60s, 5m, or 1h
      --throw-bpmn-error string   BPMN error code to throw for the job
      --timeout string            timeout duration to submit for the job, for example 60s, 5m, or 1h
      --vars string               JSON object with variables for BPMN error or completion outcomes
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
  -v, --verbose            show additional output and API request diagnostics on stderr
```

### SEE ALSO

* [c8volt update]({{ "/cli/c8volt_update" | relative_url }})	 - Update existing resources

