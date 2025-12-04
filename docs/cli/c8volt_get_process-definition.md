---
title: "c8volt get process-definition"
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get process-definition

Get deployed process definitions

```
c8volt get process-definition [flags]
```

### Options

```
  -b, --bpmn-process-id string   BPMN process ID to filter process instances
  -h, --help                     help for process-definition
  -k, --key string               process definition key to fetch
      --latest                   fetch the latest version(s) of the given BPMN process(s)
  -v, --pd-version int32         process definition version
      --pd-version-tag string    process definition version tag
      --stat                     include process definition statistics
```

### Options inherited from parent commands

```
  -y, --auto-confirm               auto-confirm prompts for non-interactive use
      --backoff-max-retries int    max retry attempts (0 = unlimited)
      --backoff-timeout duration   overall timeout for the retry loop (default 2m0s)
      --config string              path to config file
      --debug                      enable debug logging, overwrites and is shorthand for --log-level=debug
  -j, --json                       output as JSON (where applicable)
      --keys-only                  output as keys only (where applicable), can be used for piping to other commands, like cancel or delete
      --log-format string          log format (json, plain, text) (default "plain")
      --log-level string           log level (debug, info, warn, error) (default "info")
      --log-with-source            include source file and line number in logs
      --no-err-codes               suppress error codes in error outputs
      --profile string             config active profile name to use (e.g. dev, prod)
  -q, --quiet                      suppress all output, except errors, overrides --log-level
      --tenant string              default tenant ID
```

### SEE ALSO

* [c8volt get](c8volt_get.md)	 - Get resources

