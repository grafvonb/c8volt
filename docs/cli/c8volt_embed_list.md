---
title: "c8volt embed list"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt embed list

List bundled BPMN fixture files

### Synopsis

List bundled BPMN fixture files.

Use before `embed deploy` or `embed export` to get exact file names.

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

* [c8volt embed](c8volt_embed)	 - Use bundled BPMN fixtures

