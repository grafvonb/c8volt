---
title: "c8volt update job"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update job

Update a job by key

### Synopsis

Update a Camunda job by key.

The command supports retries, timeout updates, and worker outcome modes for Camunda 8.8 and 8.9. It builds a pre-mutation plan, supports --dry-run previews, and asks for confirmation before material interactive mutations. Retry updates are confirmed by reading the job by key by default; timeout updates and worker outcomes report accepted submission without deadline or outcome confirmation. JSON mutations require --dry-run, --auto-confirm, or --automation, and --json cannot be combined with --verbose. Camunda 8.7 returns an unsupported-version error before mutation.

```
c8volt update job [flags]
```

### Examples

```
  ./c8volt update job --key <job-key> --retries 3 --dry-run
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
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt update](c8volt_update)	 - Update existing resources

