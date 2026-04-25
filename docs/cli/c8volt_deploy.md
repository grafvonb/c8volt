---
title: "c8volt deploy"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt deploy

Deploy state-changing resources such as BPMN definitions

### Synopsis

Deploy state-changing resources such as BPMN definitions.

Use this command family when you want c8volt to upload deployable assets into Camunda.
Choose `deploy process-definition` for local files or `embed deploy` for
bundled fixtures. Child commands explain when deployment waits for confirmation by
default, how `--no-wait` changes the completion contract, and what to inspect next.

```
c8volt deploy [flags]
```

### Examples

```
  ./c8volt deploy process-definition --help
  ./c8volt deploy process-definition --file ./order-process.bpmn
  ./c8volt embed deploy --all --run
```

### Options

```
  -h, --help   help for deploy
```

### Options inherited from parent commands

```
  -y, --auto-confirm       auto-confirm prompts for non-interactive use
      --automation         enable the canonical non-interactive contract for commands that explicitly support it
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

* [c8volt](c8volt)	 - Operate Camunda 8 with guided help and script-safe output modes
* [c8volt deploy process-definition](c8volt_deploy_process-definition)	 - Deploy BPMN process definition files

