---
title: "c8volt cancel"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt cancel

Cancel running process instances

### Synopsis

Cancel running process instances.

Validates the affected tree, prompts before destructive changes, and waits for
observed cancellation.

```
c8volt cancel [flags]
```

### Examples

```
  c8volt cancel process-instance --key <process-instance-key> --dry-run
  c8volt cancel process-instance --bpmn-process-id order --state active --dry-run
```

### Options

```
  -h, --help   help for cancel
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

* [c8volt]({{ "/cli/c8volt/" | relative_url }})	 - Operate Camunda 8 workflows from the command line
* [c8volt cancel process-instance]({{ "/cli/c8volt_cancel_process-instance" | relative_url }})	 - Cancel process instances by key or filters

