---
title: "c8volt deploy process-definition"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt deploy process-definition

Deploy BPMN process definition files

### Synopsis

Deploy BPMN process definition files to Camunda.

Deployment uses the configured tenant, or the default tenant when none is configured. --all-tenants is not supported because deployment requires one destination tenant.

By default c8volt waits for deployment confirmation. Use --run to start one process instance for each deployed definition.

```
c8volt deploy process-definition [flags]
```

### Examples

```
  c8volt deploy process-definition --file order.bpmn
  c8volt deploy process-definition --file order.bpmn --run
  c8volt --tenant tenant-a deploy process-definition --file order.bpmn
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

* [c8volt deploy]({{ "/cli/c8volt_deploy" | relative_url }})	 - Deploy BPMN resources to Camunda

