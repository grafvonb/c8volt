---
title: "c8volt resolve"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt resolve

Resolve operational incidents

### Synopsis

Resolve operational incidents.

The incident command resolves known incident keys and reports each target
independently. Resolution is state-changing and waits for confirmation by
default unless a leaf command supports an explicit opt-out.

```
c8volt resolve [flags]
```

### Examples

```
  ./c8volt resolve incident --key 2251799813685249
  ./c8volt resolve inc --key 2251799813685249 --key 2251799813685250
  printf '%s\n' 2251799813685249 2251799813685250 | ./c8volt resolve inc -
```

### Options

```
  -h, --help   help for resolve
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

* [c8volt](c8volt)	 - Operate Camunda 8 workflows from the command line
* [c8volt resolve incident](c8volt_resolve_incident)	 - Resolve incidents by key
* [c8volt resolve process-instance](c8volt_resolve_process-instance)	 - Resolve process-instance incidents by key

