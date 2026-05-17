---
title: "c8volt ops repair"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt ops repair

Discover repair and remediation workflows

### Synopsis

Discover repair and remediation workflows.

The repair command group lists target-specific remediation workflows for
incidents and process-instance selected incidents. Use a concrete target command
to provide keys, filters, dry-run controls, variable updates, job repair
options, and audit reports. This grouping command does not define target keys or
run remediation behavior by itself.

```
c8volt ops repair [flags]
```

### Examples

```
  ./c8volt ops repair --help
  ./c8volt capabilities --json
```

### Options

```
  -h, --help   help for repair
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

* [c8volt ops](c8volt_ops)	 - Discover high-level operational workflows
* [c8volt ops repair incident](c8volt_ops_repair_incident)	 - Repair incidents by key or filter
* [c8volt ops repair process-instance](c8volt_ops_repair_process-instance)	 - Repair incidents selected by process instances

