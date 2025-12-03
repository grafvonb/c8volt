---
title: "c8volt run process-instance"
slug: "c8volt_run_process-instance"
description: "CLI reference for c8volt run process-instance"
---

## c8volt run process-instance

Run process instance(s) by process definition

```
c8volt run process-instance [flags]
```

### Options

```
  -b, --bpmn-process-id strings   BPMN process ID(s) to run process instance for (mutually exclusive with --pd-id). Runs latest version unless --pd-version is specified
  -n, --count int                 Number of instances to start for a single process definition (default 1)
      --fail-fast                 Stop scheduling new instances after the first error
  -h, --help                      help for process-instance
      --pd-id strings             Specific process definition ID(s) to run process instance for (mutually exclusive with --bpmn-process-id)
      --pd-version int32          Specific version of the process definition to use when running by BPMN process ID (supported only with --bpmn-process-id)
      --vars string               JSON-encoded variables to pass to the started process instance(s)
  -w, --workers int               Maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))
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
      --no-wait                             skip waiting for the creation to be fully processed (no status checks)
  -q, --quiet                               suppress all output, except errors, overrides --log-level
      --tenant string                       default tenant ID
```

### SEE ALSO

* [c8volt run](c8volt_run.md)	 - Run resources

