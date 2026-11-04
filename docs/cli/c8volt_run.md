---
title: "c8volt run"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt run

Start process instances

### Synopsis

Start process instances and wait until they are visible.

Creation confirmation does not mean the process has completed. Use expect
process-instance when execution must reach a particular state.

```
c8volt run [flags]
```

### Examples

```
  c8volt run process-instance --bpmn-process-id order
  c8volt run process-instance --bpmn-process-id order --set-vars '{"customerId":"1234"}'
```

### Options

```
  -h, --help   help for run
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
* [c8volt run process-instance]({{ "/cli/c8volt_run_process-instance" | relative_url }})	 - Start process instances and confirm creation

