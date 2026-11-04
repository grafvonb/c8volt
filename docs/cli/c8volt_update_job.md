---
title: "c8volt update job"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update job

Update a job or complete selected jobs

### Synopsis

Update job retries or timeout, or submit worker outcomes.

Use --set-retries for retries and --set-timeout for the job timeout.
Inherited --timeout controls HTTP requests only. Use --complete to complete a
job, --fail with --set-retries to fail it, or --throw-bpmn-error with
--set-error-code to report a BPMN error. Add --set-error-message for context.

Completion accepts explicit keys or search filters. --retries is an exact
completion filter, including zero; it never writes retries. The selection is
frozen before confirmation. Completing a listener acknowledges its work without
executing its business logic. Other actions require one key.

Use --dry-run to review the plan. Retry updates are verified; timeout updates
and worker outcomes return after acceptance. JSON retains partial failures.

```
c8volt update job [-] [flags]
```

### Examples

```
  c8volt update job --key <job-key> --set-retries 3 --dry-run
  c8volt update job --key <job-key> --set-timeout 5m
  c8volt update job --key <job-key> --complete --set-vars '{"approved":true}'
  c8volt update job --key <job-key> --fail --set-retries 0 --set-error-message "worker unavailable"
  c8volt update job --key <job-key> --throw-bpmn-error --set-error-code PAYMENT_DECLINED
```

### Options

```
  -n, --batch-size int32              jobs per discovery page for selector completion (default 1000)
      --complete                      complete the job through the worker outcome API
      --dry-run                       preview job updates without submitting mutation
      --element-id string             BPMN element ID selector for --complete
      --element-instance-key string   element instance key selector for --complete
      --fail                          report a technical job failure
      --fail-fast                     stop scheduling new completions after the first unsatisfied outcome
  -h, --help                          help for job
  -k, --key stringArray               job key to update; for --complete, repeat or use comma-separated keys and stdin '-'
      --kind string                   job kind selector for --complete; case-insensitive
  -l, --limit int32                   maximum unique selector completion targets; 0 means uncapped
      --listener-event-type string    listener event selector for --complete; case-insensitive
      --no-wait                       skip retry confirmation; bulk completion permits one reconciliation read without polling
      --no-worker-limit               use all queued jobs as workers when --workers is unset
      --pi-key string                 process instance key selector for --complete
      --retries int32                 exact retry count selector with keyless --complete (including 0)
      --set-bid string                assign the first business ID atomically with --complete (only c8.10+)
      --set-error-code string         BPMN error code to submit with --throw-bpmn-error
      --set-error-message string      operator message for worker outcome modes
      --set-retries int32             retry count to set or remaining retries for --fail (including 0)
      --set-retry-backoff string      duration before a failed job becomes retryable, for example 60s, 5m, or 1h
      --set-timeout string            timeout duration to submit for the job, for example 60s, 5m, or 1h
      --set-vars string               JSON object with variables for BPMN error or completion outcomes
  -s, --state string                  job state selector for --complete; case-insensitive
      --throw-bpmn-error              throw a BPMN error; requires --set-error-code
      --type string                   job type selector for --complete
      --worker string                 worker name selector for --complete
  -w, --workers int                   maximum concurrent completion workers; omit for an automatic limit
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

* [c8volt update]({{ "/cli/c8volt_update" | relative_url }})	 - Update existing resources

