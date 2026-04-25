---
title: "c8volt deploy process-definition"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt deploy process-definition

Deploy BPMN process definition files

### Synopsis

Deploy BPMN process definition files and report the deployed definitions.

By default c8volt waits until the deployment is confirmed before returning success. Use --no-wait when accepted deployment work should return immediately, then verify the resulting definitions with `get process-definition`, or start a follow-up instance with --run when a smoke test should happen right away.

Default output stays operator-oriented. Use --json for the shared result envelope and pair it with --automation on supported non-interactive paths.

```
c8volt deploy process-definition [flags]
```

### Examples

```
  ./c8volt deploy pd --file ./order-process.bpmn
  ./c8volt deploy pd --file ./order-process.bpmn --run
  ./c8volt --automation --json deploy pd --file ./order-process.bpmn --no-wait
  ./c8volt get pd --bpmn-process-id order-process --latest --json
  ./c8volt deploy pd --file - < ./order-process.bpmn
```

### Options

```
  -f, --file strings   paths to BPMN/YAML file(s) or '-' for stdin
  -h, --help           help for process-definition
      --no-wait        skip waiting for the deployment to be fully processed
      --run            run single process instance without vars after deploying process definition(s)
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

* [c8volt deploy](c8volt_deploy)	 - Deploy state-changing resources such as BPMN definitions

