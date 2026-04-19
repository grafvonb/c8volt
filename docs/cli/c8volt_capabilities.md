---
title: "c8volt capabilities"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt capabilities

Describe machine-readable CLI capabilities

### Synopsis

Describe the machine-readable c8volt command surface for automation.
Use this command to discover command paths, flags, output modes, mutation behavior, and contract support without scraping prose help.

Prefer `c8volt capabilities --json` when driving the CLI from AI agents, scripts, or CI. The human-facing command taxonomy and help output remain unchanged; plain output summarizes the command surface for humans, while JSON is the repository-native discovery surface for automation.

```
c8volt capabilities [flags]
```

### Examples

```
  ./c8volt capabilities
  ./c8volt capabilities --json
```

### Options

```
  -h, --help   help for capabilities
```

### Options inherited from parent commands

```
  -y, --auto-confirm        auto-confirm prompts for non-interactive use
      --config string       path to config file
      --debug               enable debug logging, overwrites and is shorthand for --log-level=debug
  -j, --json                output as JSON (where applicable)
      --keys-only           output as keys only (where applicable), can be used for piping to other commands
      --log-format string   log format (json, plain, text) (default "plain")
      --log-level string    log level (debug, info, warn, error) (default "info")
      --log-with-source     include source file and line number in logs
      --no-err-codes        suppress error codes in error outputs
      --profile string      config active profile name to use (e.g. dev, prod)
  -q, --quiet               suppress all output, except errors, overrides --log-level
      --tenant string       tenant ID for tenant-aware command flows (overrides env, profile, and base config)
  -v, --verbose             adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt](c8volt)	 - c8volt: Camunda 8 Operations CLI

