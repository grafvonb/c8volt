---
title: "c8volt delete"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt delete

Delete process instances or definitions

### Synopsis

Delete process instances or process definitions.

Use this command family when workflow data should be removed. Leaf commands explain
what c8volt validates first, when confirmation is required, and which follow-up
command confirms the result.

```
c8volt delete [flags]
```

### Examples

```
  ./c8volt delete pi --key 2251799813711967 --force
  ./c8volt delete pi --state completed --count 200 --auto-confirm
  ./c8volt delete pd --bpmn-process-id C88_SimpleUserTask_Process --latest --auto-confirm
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
      --debug              enable debug logging, overwrites and is shorthand for --log-level=debug
  -j, --json               output as JSON (where applicable)
      --keys-only          output as keys only (where applicable), can be used for piping to other commands
      --log-level string   log level (debug, info, warn, error) (default "info")
      --no-indicator       disable transient terminal activity indicators
      --profile string     config active profile name to use (e.g. dev, prod)
  -q, --quiet              suppress all output, except errors, overrides --log-level
      --tenant string      tenant ID for tenant-aware command flows (overrides env, profile, and base config)
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt](c8volt)	 - Operate Camunda 8 workflows from the command line
* [c8volt delete process-definition](c8volt_delete_process-definition)	 - Delete process definition resources
* [c8volt delete process-instance](c8volt_delete_process-instance)	 - Delete process instances by key or filters

