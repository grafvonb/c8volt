---
title: "c8volt update job"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update job

Update a job or complete selected jobs

### Synopsis

Update a Camunda job by key on Camunda 8.8 or newer.

Supports retry and timeout updates and worker outcomes. Explicit --key uses backend authorization without tenant filtering.

For bulk --complete, repeat --key, supply comma-separated keys, or use one positional '-' to read newline-separated keys. Flags precede stdin; blank lines are ignored, invalid or empty key tokens fail, and all input is validated before mutation. Duplicate keys are completed once. Empty stdin succeeds without mutation. Explicit keys retain backend authorization across tenants and show actual tenant context. One scalar key preserves the existing single-job result; every collection keeps the bulk result even after deduplication.

For selector --complete, omit --key and supply one or more AND predicates: --state, --type, --pi-key, --element-instance-key, --element-id, --worker, --retries, --kind, or --listener-event-type. Explicit --retries 0 is an exact selector in this mode; keyed --complete still rejects --retries. Other update modes require one scalar key and reject collections and selectors.

Selector discovery uses the configured tenant; --tenant overrides it, and empty --tenant or --all-tenants searches all accessible tenants. --batch-size controls each page; --limit caps unique selected jobs (0 means uncapped). Controls alone do not select jobs. Explicit keys cannot combine with predicates or search controls.

The complete selection is frozen before one confirmation. Incomplete discovery submits nothing. Empty results succeed without confirmation or mutation. --workers bounds concurrency; --fail-fast stops new work after an unsatisfied result. New sequential listener jobs require another invocation.

Completing a listener job acknowledges its work without executing its business logic. Accepted completion skips independent confirmation and does not verify process termination. Bulk requests are not automatically replayed; ambiguous outcomes use bounded observation, with at most one read under --no-wait. No retries, incidents or processes are repaired implicitly.

Bulk dry-run previews show count, filters or explicit selection and tenant scope; --verbose lists candidate context. JSON retains all preview candidates or ordered execution outcomes, including partial failures. Quiet suppresses prose while preserving requested JSON. A later selector invocation discovers a fresh selection.

Bulk previews, summaries and verbose rows use the configured logger at INFO level. Bulk failures produce one compact logged error after the result summary, grouping repeated reasons. Quiet retains that error. Use --verbose for per-job outcomes and --debug or configured debug logging for HTTP diagnostics. JSON preserves the original failure evidence and partial results in one envelope.

c8volt plans the update and asks for confirmation before material interactive mutations. Retry updates are verified by reading the job; timeout updates and worker outcomes return after acceptance without waiting for confirmation.

Use --dry-run to inspect the plan without mutation. With --json, mutations require --dry-run, --auto-confirm, or --automation; --json cannot be combined with --verbose.

```
c8volt update job [-] [flags]
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
  ./c8volt update job --kind TASK_LISTENER --listener-event-type CANCELING --state CREATED --type removeTaskData --complete --dry-run --verbose
  ./c8volt update job --type removeTaskData --retries 0 --limit 1000 --batch-size 100 --complete --auto-confirm --json
  ./c8volt update job --key 2251799870796844,2251799870796848 --key 2251799870796849 --complete --dry-run
  ./c8volt update job - --complete --auto-confirm --json < reviewed-job-keys.txt
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
      --message string                operator message for worker outcome modes
      --no-wait                       skip retry confirmation; bulk completion permits one reconciliation read without polling
      --no-worker-limit               use all queued jobs as workers when --workers is unset
      --pi-key string                 process instance key selector for --complete
      --retries int32                 retry count to set, remaining retries for --fail, or exact selector with keyless --complete (including 0)
      --retry-backoff string          duration before a failed job becomes retryable, for example 60s, 5m, or 1h
  -s, --state string                  job state selector for --complete; case-insensitive
      --throw-bpmn-error string       BPMN error code to throw for the job
      --timeout string                timeout duration to submit for the job, for example 60s, 5m, or 1h
      --type string                   job type selector for --complete
      --vars string                   JSON object with variables for BPMN error or completion outcomes
      --worker string                 worker name selector for --complete
  -w, --workers int                   maximum concurrent completion workers (default: min(count, 2*GOMAXPROCS, 32))
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
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt update]({{ "/cli/c8volt_update" | relative_url }})	 - Update existing resources

