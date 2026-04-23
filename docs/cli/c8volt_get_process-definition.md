---
title: "c8volt get process-definition"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get process-definition

List or fetch deployed process definitions

### Synopsis

List or fetch deployed process definitions.

Use this read-only command to inspect deployed BPMN models by key, BPMN process
ID, version selectors, or the latest deployed version. Default output is aimed
at human review; prefer `--json` when chaining the result into scripts or
AI-assisted workflows. Use `--xml` only when you need the raw BPMN XML for a
single definition selected by `--key`. When `--stat` is enabled,
Camunda `8.8` reports process-definition element statistics and omits the
`in:` field because incident-bearing process-instance counts are not available
from its native process-definition statistics endpoint. Camunda `8.9` enriches
`ac` and `in:<count>` from native process-instance statistics for the
exact process definition version; `cp` and `cx` keep their existing
process-definition statistics meaning.
Camunda `8.7` rejects statistics because the generated client surface does
not provide the same native statistics endpoints.

```
c8volt get process-definition [flags]
```

### Examples

```
  ./c8volt get pd --latest
  ./c8volt get pd --bpmn-process-id C88_SimpleUserTask_Process --latest
  ./c8volt get pd --key 2251799813686017 --json
  ./c8volt get pd --key 2251799813686017 --xml
```

### Options

```
  -b, --bpmn-process-id string   BPMN process ID to filter process instances
  -h, --help                     help for process-definition
  -k, --key string               process definition key to fetch
      --latest                   fetch the latest version(s) of the given BPMN process(s)
      --pd-version int32         process definition version
      --pd-version-tag string    process definition version tag
      --stat                     include process definition statistics; 8.9 adds active/incident instance stats, 8.7 unsupported
      --xml                      output the selected process definition as raw XML (requires --key and no other filters)
```

### Options inherited from parent commands

```
  -y, --auto-confirm               auto-confirm prompts for non-interactive use
      --automation                 enable the canonical non-interactive contract for commands that explicitly support it
      --backoff-max-retries int    max retry attempts (0 = unlimited)
      --backoff-timeout duration   overall timeout for the retry loop (default 2m0s)
      --config string              path to config file
      --debug                      enable debug logging, overwrites and is shorthand for --log-level=debug
  -j, --json                       output as JSON (where applicable)
      --keys-only                  output as keys only (where applicable), can be used for piping to other commands
      --log-format string          log format (json, plain, text) (default "plain")
      --log-level string           log level (debug, info, warn, error) (default "info")
      --log-with-source            include source file and line number in logs
      --no-err-codes               suppress error codes in error outputs
      --profile string             config active profile name to use (e.g. dev, prod)
  -q, --quiet                      suppress all output, except errors, overrides --log-level
      --tenant string              tenant ID for tenant-aware command flows (overrides env, profile, and base config)
  -v, --verbose                    adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt get](c8volt_get)	 - Read cluster, process, and resource state without changing it

