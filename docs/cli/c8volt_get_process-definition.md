---
title: "c8volt get process-definition"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get process-definition

List or fetch deployed process definitions

```
c8volt get process-definition [flags]
```

### Examples

```
  ./c8volt get pd --latest
  ./c8volt get pd --bpmn-process-id C88_SimpleUserTask_Process --latest
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
      --stat                     include process definition statistics
      --xml                      output the selected process definition as raw XML (requires --key and no other filters)
```

### Options inherited from parent commands

```
  -y, --auto-confirm               auto-confirm prompts for non-interactive use
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
      --tenant string              default tenant ID
  -v, --verbose                    adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt get](c8volt_get)	 - Get resources

