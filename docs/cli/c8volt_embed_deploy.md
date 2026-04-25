---
title: "c8volt embed deploy"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt embed deploy

Deploy bundled BPMN fixtures

### Synopsis

Deploy bundled BPMN fixtures.

Use `--all` for the configured Camunda version, or pass one or more `--file` values from `embed list`. By default c8volt waits for deployment confirmation. Add `--run` when you want one process instance started after deployment.

```
c8volt embed deploy [flags]
```

### Examples

```
  ./c8volt embed list
  ./c8volt embed deploy --all
  ./c8volt embed deploy --file processdefinitions/C88_SimpleUserTaskProcess.bpmn
  ./c8volt embed deploy --all --run
```

### Options

```
      --all            deploy all embedded files for the configured Camunda version
  -f, --file strings   embedded file(s) to deploy (repeatable)
  -h, --help           help for deploy
      --no-wait        skip waiting for the deployment to be fully processed
      --run            run single process instance without vars after deploying process definition(s)
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

* [c8volt embed](c8volt_embed)	 - Use bundled BPMN fixtures

