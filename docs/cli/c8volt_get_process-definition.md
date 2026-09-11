---
title: "c8volt get process-definition"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get process-definition

List or fetch deployed process definitions

### Synopsis

List or fetch deployed process definitions.

Inspect deployed BPMN models by key, BPMN process ID, version selectors, or
latest deployed version. Use `--xml` only with `--key`.

Tenant contract: `--tenant` scopes list/latest and BPMN selector discovery where
supported. Explicit `--key` and XML key lookups are backend-authorized admin input;
c8volt displays returned tenant metadata without rejecting solely because it differs
from the selected tenant.

Collection order is stable across list, `--latest`, `--stat`, JSON, keys-only,
and watch output: tenant ID ascending, BPMN process ID ascending, version
descending, then process-definition key ascending. Tenant IDs and BPMN process
IDs are compared exactly and case-sensitively; `<default>` is ordinary text
for ordering; keys are opaque text and are not parsed numerically.

`--latest` selects one newest definition per exact tenant ID and BPMN process
ID pair, using the lowest exact-text process-definition key when versions tie.
Camunda `8.7` applies this latest selection within its existing 1000 visible
definition compatibility window; Camunda `8.8` or newer uses native latest
filtering and the same final collection order.

Watch mode repaints one terminal view, starting immediately and then waiting
`1s` between refreshes unless `--watch-interval` is set. Each refresh body
matches normal list output without watch-only snapshot labels. Without a selector,
`--watch` observes all visible process definitions. JSON, keys-only, XML,
quiet, and automation combinations are rejected before lookup work. Existing
timeout and backoff retry settings bound the watch run; successful refreshes reset
the consecutive retry budget.

When `--bpmn-process-id` is set, c8volt validates that at least one visible
process definition matches the selector before rendering output. A missing selector
fails with the shared local diagnostic instead of rendering an ambiguous empty list.

With `--json`, validation and runtime failures during command execution use one
shared error envelope. Without `--json`, the diagnostic is written to stderr.
`--no-err-codes` changes only the process exit status; the reported failure and
immediate termination are unchanged. Bootstrap failures and argument or flag parsing
errors before command execution retain their established diagnostics.

`--stat` requires Camunda `8.8` or newer and prints exact-version
counts. Camunda `8.7` does not support native statistics.

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
  -n, --batch-size int32          number of process definitions to request per discovery page; does not cap total returned rows (max limit 1000 enforced by server) (default 1000)
  -b, --bpmn-process-id string    BPMN process ID to filter process instances
  -h, --help                      help for process-definition
  -k, --key string                process definition key to fetch
      --latest                    only include the latest matching process-definition version per exact tenant/BPMN process group
      --pd-version int32          process definition version
      --pd-version-tag string     process definition version tag
      --stat                      include process definition statistics; 8.8 or newer includes incident counts, 8.7 unsupported
      --watch                     repeat the process-definition lookup as a repainted terminal view until interrupted, timed out, or retry-exhausted
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
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt get]({{ "/cli/c8volt_get" | relative_url }})	 - Inspect cluster, process, job, element, incident, tenant, and resource state

