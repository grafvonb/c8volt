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

--latest selects the newest definition per exact tenant ID and BPMN process ID, breaking version ties by the lowest exact-text key. Camunda 8.7 selects within its 1000 visible-definition compatibility window; Camunda 8.8 or newer uses native latest filtering.

--stat includes exact-version statistics and requires Camunda 8.8 or newer.

--watch repeats the lookup until interrupted, timed out, or retries are exhausted. It starts immediately; --watch-interval controls subsequent checks. Without a selector it observes all visible definitions. Successful checks reset the consecutive retry budget. --watch cannot be combined with --json, --keys-only, --xml, --quiet, or --automation.

```
c8volt get process-definition [flags]
```

### Examples

```
  ./c8volt get process-definition --latest
  ./c8volt get process-definition --bpmn-process-id <bpmn-process-id> --latest
  ./c8volt get process-definition --bpmn-process-id <bpmn-process-id> --latest --watch
  ./c8volt get process-definition --watch --watch-interval 2s
  ./c8volt get process-definition --key <process-definition-key> --json
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
      --stat                      include process definition statistics; 8.8 or newer includes incident counts, 8.7 unsupported
      --watch                     repeat the process-definition lookup until interrupted, timed out, or retries are exhausted
      --watch-interval duration   interval between process-definition watch refreshes after the immediate first refresh (default 1s)
      --xml                       output the selected process definition as raw XML (requires --key and no other filters)
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
  -v, --verbose            show additional output and API request diagnostics on stderr
```

### SEE ALSO

* [c8volt get]({{ "/cli/c8volt_get" | relative_url }})	 - Inspect cluster, process, job, element, incident, tenant, and resource state

