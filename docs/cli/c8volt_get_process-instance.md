---
title: "c8volt get process-instance"
slug: "c8volt_get_process-instance"
description: "CLI reference for c8volt get process-instance"
---

## c8volt get process-instance

Get process instances

```
c8volt get process-instance [flags]
```

### Options

```
  -b, --bpmn-process-id string   BPMN process ID to filter process instances
      --children-only            show only child process instances, meaning instances that have a parent key set
  -n, --count int32              number of process instances to fetch (max limit 1000 enforced by server) (default 1000)
  -h, --help                     help for process-instance
      --incidents-only           show only process instances that have incidents
  -k, --key string               process instance key to fetch
      --no-incidents-only        show only process instances that have no incidents
      --orphan-children-only     show only child instances where parent key is set but the parent process instance does not exist (anymore)
      --parent-key string        parent process instance key to filter process instances
      --pd-key string            process definition key (mutually exclusive with bpmn-process-id, pd-version, and pd-version-tag)
      --pd-version int32         process definition version
      --pd-version-tag string    process definition version tag
      --roots-only               show only root process instances, meaning instances with empty parent key
  -s, --state string             state to filter process instances: all, active, completed, canceled (default "all")
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

* [c8volt get](c8volt_get.md)	 - Get resources

