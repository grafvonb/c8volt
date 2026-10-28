---
title: "c8volt get process-definition"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get process-definition

List or fetch deployed process definitions

### Synopsis

List or fetch deployed process definitions.

Select by key, BPMN process ID, version, or version tag. Use --xml only with --key. A --bpmn-process-id selector must match a visible definition.

--tenant limits list and selector discovery. Explicit --key and XML lookups use backend authorization without tenant filtering.

--latest selects the newest definition per exact tenant ID and BPMN process ID using native latest filtering, breaking version ties by the lowest exact-text key.

Camunda 8.10 supplies optional lifecycle state in human rows and JSON (state): ACTIVE allows new instances, DRAINING blocks new starts while unfinished instances remain, and DELETED means runtime removal, not removal of historical data. Missing state is omitted, including on 8.8/8.9; unfamiliar values are preserved.

Lifecycle and --stat counts are independent: suspended instances can keep a definition DRAINING even with zero active instances. Backend deletion does not cancel or resume instances; delete pd --force explicitly cancels unfinished instances first. These reads do not change deletion behavior.

--stat includes exact-version statistics and requires Camunda 8.8 or newer. On Camunda 8.10, rows append su: for suspended instances, with a dash for an observed zero; active counts remain separate. JSON statistics include a suspended count and suspendedCountSupported availability flag; a supported zero omits the count. Camunda 8.8/8.9 omit the suspended token and fields because the count is unavailable. With --stat --watch, suspended count and availability changes update the row and clear stale values. Empty searches suppress the human summary under --quiet while preserving requested JSON and keys-only results.

--watch repeats the lookup indefinitely by default until interrupted or retries are exhausted. It refreshes immediately, then at --watch-interval. --watch-timeout limits the session (0 = unlimited), independently of --timeout for HTTP requests and retry backoff. Request timeouts are retried while the session is active. Without a selector it observes all visible definitions.

A trailing * marks changed definitions and + marks new ones; changed displayed values are also bold on supported terminals when TERM is set and not dumb, and NO_COLOR is unset. Redirected output uses plain markers. Lifecycle and statistics changes count once per row. The first refresh is unmarked; later refreshes compare with the last successfully displayed result. Failures preserve that baseline, omitted values disappear, and unchanged refreshes clear markers. Returned DELETED rows remain visible until absent from a successful refresh.

Refresh summaries and verbose timing use the configured logger at INFO; slow-refresh warnings use WARN. Suppressing INFO does not suppress rows. --watch cannot be combined with --json, --keys-only, --xml, --quiet, or --automation.

```
c8volt get process-definition [flags]
```

### Examples

```
  ./c8volt get process-definition --latest
  ./c8volt get process-definition --bpmn-process-id <bpmn-process-id> --latest
  ./c8volt get process-definition --bpmn-process-id <bpmn-process-id> --latest --watch
  ./c8volt get process-definition --watch --watch-interval 2s
  ./c8volt get process-definition --watch --watch-timeout 10m
  ./c8volt get process-definition --key <process-definition-key> --json
  ./c8volt get process-definition --key <process-definition-key> --stat
  ./c8volt get process-definition --stat --json
  ./c8volt get process-definition --key <process-definition-key> --xml
```

### Options

```
  -n, --batch-size int32          number of process definitions to request per discovery page; does not cap total results (max limit 1000 enforced by server) (default 1000)
  -b, --bpmn-process-id string    BPMN process ID to filter process instances
  -h, --help                      help for process-definition
  -k, --key string                process definition key to fetch
      --latest                    only include the latest matching process-definition version per exact tenant/BPMN process group
      --pd-version int32          process definition version
      --pd-version-tag string     process definition version tag
      --stat                      include process definition statistics, incident counts, and suspended counts when supported
      --watch                     repeat the process-definition lookup until interrupted, --watch-timeout is reached, or retries are exhausted
      --watch-interval duration   interval between process-definition watch refreshes after the immediate first refresh (default 1s)
      --watch-timeout duration    maximum watch session duration (0 = unlimited; requires --watch)
      --xml                       output the selected process definition as raw XML (requires --key and no other filters)
```

### Options inherited from parent commands

```
      --all-tenants          clear configured tenant filtering and search all tenants visible to the authenticated user; mutually exclusive with --tenant
  -y, --auto-confirm         auto-confirm prompts for non-interactive use
      --automation           enable non-interactive mode for commands that explicitly support it
      --config string        path to config file
      --debug                enable debug logging
  -j, --json                 output as JSON (where applicable)
      --keys-only            output keys only (where applicable)
      --log-level string     log level (debug, info, warn, error) (default "info")
      --no-indicator         disable transient terminal activity indicators
      --profile string       config active profile name to use (e.g. dev, prod)
  -q, --quiet                suppress output except errors
      --request-rate float   maximum shared request starts per second (fractions allowed; 0 disables pacing)
      --tenant string        tenant ID for discovery/search, selection, create, deploy, and run flows; explicit empty values can clear configured discovery filters, and explicit keys/IDs remain backend-authorized
      --timeout duration     HTTP request timeout (default 30s)
  -v, --verbose              show additional output
```

### SEE ALSO

* [c8volt get]({{ "/cli/c8volt_get" | relative_url }})	 - Inspect cluster, process, job, element, incident, tenant, and resource state

