---
title: "c8volt deploy"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt deploy

Deploy BPMN resources to Camunda

### Synopsis

Deploy BPMN resources to Camunda.

Use `deploy pd` for local BPMN files or stdin. Use `embed deploy` for bundled
fixtures.

```
c8volt deploy [flags]
```

### Examples

```
  ./c8volt embed export --file processdefinitions/C88_SimpleUserTaskProcess.bpmn --out ./fixtures
  ./c8volt deploy pd --file ./fixtures/processdefinitions/C88_SimpleUserTaskProcess.bpmn --run
  ./c8volt embed deploy --all --run
```

### Options

```
  -h, --help   help for deploy
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
* [c8volt deploy process-definition](c8volt_deploy_process-definition)	 - Deploy BPMN process definition files

