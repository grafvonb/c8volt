---
title: "c8volt deploy process-definition"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt deploy process-definition

Deploy BPMN process definition files

### Synopsis

Deploy BPMN process definition files and report the deployed definitions.

By default c8volt waits for deployment confirmation. Use --run to start one process instance for each deployed definition.

Add --no-wait to verify later with `get pd`.

```
c8volt deploy process-definition [flags]
```

### Examples

```
  ./c8volt embed export --file processdefinitions/C88_SimpleUserTaskProcess.bpmn --out ./fixtures
  ./c8volt deploy pd --file ./fixtures/processdefinitions/C88_SimpleUserTaskProcess.bpmn
  ./c8volt deploy pd --file ./fixtures/processdefinitions/C88_SimpleUserTaskProcess.bpmn --run
  ./c8volt deploy pd --file ./fixtures/processdefinitions/C88_SimpleUserTaskProcess.bpmn --no-wait
  ./c8volt get pd --bpmn-process-id C88_SimpleUserTask_Process --latest --json
```

### Options

```
  -f, --file strings   paths to BPMN/YAML file(s) or '-' for stdin
  -h, --help           help for process-definition
      --no-wait        return after deployment is accepted
      --run            start one process instance per deployed definition
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

* [c8volt deploy](c8volt_deploy)	 - Deploy BPMN resources to Camunda

