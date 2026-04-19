---
title: "c8volt capabilities"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt capabilities

Describe the public CLI contract for automation and discovery

### Synopsis

Describe the machine-readable c8volt command surface for automation.
Use this command to discover public command paths, visible flags, output modes, mutation behavior, contract support, and automation-mode support without scraping prose help.

Prefer `c8volt capabilities --json` when driving the CLI from AI agents, scripts, or CI. The human-facing command taxonomy and help output remain unchanged; plain output summarizes the public command surface for humans, while JSON is the repository-native discovery surface for automation, including whether each command currently supports `--automation` as the canonical non-interactive contract. Hidden shell-completion and internal helper commands stay out of this document.

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
      --automation          enable the canonical non-interactive contract for commands that explicitly support it
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

* [c8volt](c8volt)	 - Operate Camunda 8 with guided help and script-safe output modes

