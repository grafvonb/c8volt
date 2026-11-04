---
title: "c8volt config show"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt config show

Show effective configuration

### Synopsis

Show effective configuration with secrets redacted.

Settings resolve as: flag > environment > profile > base config > default.

```
c8volt config show [flags]
```

### Examples

```
  c8volt config show
  c8volt --profile production config show
```

### Options

```
  -h, --help       help for show
      --template   compatibility shortcut: print a blank configuration template
      --validate   compatibility shortcut: validate the effective configuration and exit with an error code if invalid
```

### Options inherited from parent commands

```
      --all-tenants          search all accessible tenants; cannot combine with --tenant
  -y, --auto-confirm         auto-confirm prompts for non-interactive use
      --automation           enable non-interactive mode for commands that explicitly support it
      --config string        path to config file
      --debug                enable debug logging
  -j, --json                 output as JSON (where applicable)
      --keys-only            output keys only (where applicable)
      --log-level string     log level (debug, info, warn, error) (default "info")
      --no-indicator         disable transient terminal activity indicators
      --no-version-check     skip the automatic advisory gateway version check
      --profile string       config active profile name to use (e.g. dev, prod)
  -q, --quiet                suppress output except errors
      --request-rate float   maximum shared request starts per second (fractions allowed; 0 disables pacing)
      --tenant string        tenant for discovery and creation; explicit keys use backend authorization
      --timeout duration     HTTP request timeout (default 30s)
  -v, --verbose              show additional output
```

### SEE ALSO

* [c8volt config]({{ "/cli/c8volt_config" | relative_url }})	 - Inspect and validate c8volt configuration

