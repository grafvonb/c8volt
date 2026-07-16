---
title: "c8volt get element"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get element

Inspect or search runtime element instances

### Synopsis

Inspect or search Camunda runtime element instances.

Use --key with an elementInstanceKey to inspect one runtime BPMN element execution record directly. The `ei` alias follows the compact element-instance tag used in human output. Search mode uses filters such as --pi-key, --element-id, --state, --type, --pd-key, and --bpmn-process-id. Search mode pages through matching runtime elements by default with the standard paging controls. --batch-size tunes per-page discovery requests only, --limit intentionally caps total returned elements, and --total returns only the matching count. Use --json for the stable element payload. Element lookup and search are supported for Camunda 8.8 and 8.9; Camunda 8.7 returns an unsupported-version error.

```
c8volt get element [flags]
```

### Examples

```
  ./c8volt get ei -k <element-instance-key>
  ./c8volt get ei --pi-key <process-instance-key> --limit 10
  ./c8volt get element --pi-key <process-instance-key> --total
  ./c8volt --json get ei --pi-key <process-instance-key> --limit 5
  ./c8volt --json get element --key <element-instance-key>
```

### Options

```
  -n, --batch-size int32         number of elements to fetch per page (max limit 1000 enforced by server) (default 1000)
  -b, --bpmn-process-id string   BPMN process ID to filter in search mode
      --element-id string        BPMN element ID to filter in search mode
  -h, --help                     help for element
  -k, --key string               element instance key for exact lookup; omit to list or search runtime elements
  -l, --limit int32              maximum number of elements to return in search mode
      --pd-key string            process definition key to filter in search mode
      --pi-key string            process instance key to filter in search mode
  -s, --state string             runtime element state to filter in search mode; case-insensitive
      --total                    return only the numeric total of matching elements
      --type string              runtime element type to filter in search mode; case-insensitive
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
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt get](c8volt_get)	 - Inspect cluster, process, job, element, incident, tenant, and resource state

