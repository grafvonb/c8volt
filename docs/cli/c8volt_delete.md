---
title: "c8volt delete"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt delete

Delete process instances or definitions

### Synopsis

Delete process instances or process definitions.

Leaf commands validate scope, require confirmation for destructive steps, and
show verification examples.

```
c8volt delete [flags]
```

### Examples

```
  ./c8volt delete process-instance --key <process-instance-key> --force
  ./c8volt delete process-instance --state terminated --batch-size 250 --limit 5 --dry-run
  ./c8volt delete process-definition --bpmn-process-id <bpmn-process-id> --latest --auto-confirm
```

### Options

```
  -h, --help   help for delete
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
      --tenant string      tenant ID for discovery/search, selection, create, deploy, and run flows; explicit keys/IDs remain backend-authorized
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt]({{ "/cli/c8volt/" | relative_url }})	 - Operate Camunda 8 workflows from the command line
* [c8volt delete process-definition]({{ "/cli/c8volt_delete_process-definition" | relative_url }})	 - Delete process definition resources
* [c8volt delete process-instance]({{ "/cli/c8volt_delete_process-instance" | relative_url }})	 - Delete process instances by key or filters

