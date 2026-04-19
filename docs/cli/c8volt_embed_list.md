---
title: "c8volt embed list"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt embed list

List embedded (virtual) files containing process definitions

### Synopsis

List embedded (virtual) files containing process definitions.

Use this read-only command to discover which BPMN resources are packaged into the c8volt binary before exporting or deploying them. Prefer `--json` when another tool needs the file names programmatically, and add `--details` when you need the full embedded paths.

```
c8volt embed list [flags]
```

### Examples

```
  ./c8volt embed list
  ./c8volt embed list --details
  ./c8volt --json embed list
```

### Options

```
      --details   show full embedded file paths
  -h, --help      help for list
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

* [c8volt embed](c8volt_embed)	 - Inspect, export, or deploy embedded BPMN resources

