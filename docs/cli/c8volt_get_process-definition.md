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

`--stat` requires Camunda `8.8` or `8.9` and prints exact-version
counts. Camunda `8.7` does not support native statistics.

```
c8volt get process-definition [flags]
```

### Examples

```
  ./c8volt get pd --latest
  ./c8volt get pd --bpmn-process-id C88_SimpleUserTask_Process --latest
  ./c8volt get pd --key <process-definition-key> --json
  ./c8volt get pd --key <process-definition-key> --xml
```

### Options

```
  -b, --bpmn-process-id string   BPMN process ID to filter process instances
  -h, --help                     help for process-definition
  -k, --key string               process definition key to fetch
      --latest                   fetch the latest version(s) of the given BPMN process(s)
      --pd-version int32         process definition version
      --pd-version-tag string    process definition version tag
      --stat                     include process definition statistics; 8.8/8.9 include incident counts, 8.7 unsupported
      --xml                      output the selected process definition as raw XML (requires --key and no other filters)
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
      --tenant string      tenant ID for tenant-aware command flows (overrides env, profile, and base config)
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt get](c8volt_get)	 - Inspect cluster, process, and resource state

