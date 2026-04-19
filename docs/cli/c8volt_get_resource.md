---
title: "c8volt get resource"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get resource

Get a resource by id

### Synopsis

Get a single resource by id.
It requires --id to select exactly one resource and renders the standard single-resource view.

Use this read-only command when you already know the resource id and need the stored metadata or payload reference. Default output stays human-oriented. Use --json for the shared machine-readable result envelope or --keys-only when chaining ids into other commands.

```
c8volt get resource [flags]
```

### Examples

```
  ./c8volt get resource --id resource-id-123
  ./c8volt --json get resource --id resource-id-123
  ./c8volt --keys-only get resource --id resource-id-123
```

### Options

```
  -h, --help        help for resource
  -i, --id string   resource id to fetch
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

