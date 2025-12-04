---
title: "c8volt get variable"
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get variable

Get a variable by its name from a process instance

```
c8volt get variable [flags]
```

### Options

```
  -h, --help   help for variable
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

