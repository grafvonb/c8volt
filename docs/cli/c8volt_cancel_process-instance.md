---
title: "c8volt cancel process-instance"
slug: "c8volt_cancel_process-instance"
description: "CLI reference for c8volt cancel process-instance"
---

## c8volt cancel process-instance

Cancel process instance(s) by key(s) and wait for the cancellation to complete

```
c8volt cancel process-instance [flags]
```

### Options

```
  -b, --bpmn-process-id string   BPMN process ID to filter process instances
      --fail-fast                Stop scheduling new instances after the first error
      --force                    force cancellation of the root process instance if a process instance is a child, including all its child instances
  -h, --help                     help for process-instance
  -k, --key strings              process instance key(s) to cancel
      --no-state-check           skip checking the current state of the process instance before cancelling it
      --no-wait                  skip waiting for the cancellation to be fully processed (no status checks)
      --pd-version int32         process definition version
      --pd-version-tag string    process definition version tag
  -s, --state string             state to filter process instances: all, active, completed, canceled (default "all")
  -w, --workers int              Maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))
```

### Options inherited from parent commands

```
      --api-camunda-base-url string         Camunda API base URL
      --api-operate-base-url string         Operate API base URL
      --api-tasklist-base-url string        Tasklist API base URL
      --auth-cookie-base-url string         auth cookie base URL
      --auth-cookie-password string         auth cookie password
      --auth-cookie-username string         auth cookie username
      --auth-mode string                    authentication mode (oauth2, cookie, none) (default "none")
      --auth-oauth2-client-id string        auth client ID
      --auth-oauth2-client-secret string    auth client secret
      --auth-oauth2-scopes stringToString   auth scopes as key=value (repeatable or comma-separated) (default [])
      --auth-oauth2-token-url string        auth token URL
  -y, --auto-confirm                        auto-confirm prompts for non-interactive use
      --backoff-max-retries int             Max retry attempts (0 = unlimited)
      --backoff-timeout duration            Overall timeout for the retry loop (default 2m0s)
      --config string                       path to config file
      --debug                               enable debug logging, overwrites and is shorthand for --log-level=debug
      --http-timeout string                 HTTP timeout (Go duration, e.g. 30s)
  -j, --json                                output as JSON (where applicable)
      --keys-only                           output as keys only (where applicable), can be used for piping to other commands, like cancel or delete
      --log-format string                   log format (json, plain, text) (default "plain")
      --log-level string                    log level (debug, info, warn, error) (default "info")
      --log-with-source                     include source file and line number in logs
      --no-err-codes                        suppress error codes in error outputs
  -q, --quiet                               suppress all output, except errors, overrides --log-level
      --tenant string                       default tenant ID
```

### SEE ALSO

* [c8volt cancel](c8volt_cancel.md)	 - Cancel resources

