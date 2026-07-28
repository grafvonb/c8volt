---
title: "c8volt get resource"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get resource

Get a resource by ID

### Synopsis

Get a single resource by ID.

Requires --id. The ID must be a Camunda resource ID; process-definition keys and deployment response keys are not resource IDs.

Tenant contract: explicit --id resource targets are backend-authorized admin input; returned tenant metadata may differ from the selected tenant.

```
c8volt get resource [flags]
```

### Examples

```
  ./c8volt get resource --id <resource-id>
  ./c8volt --json get resource --id <resource-id>
  ./c8volt --keys-only get resource --id <resource-id>
```

### Options

```
  -h, --help        help for resource
  -i, --id string   resource ID to fetch
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

* [c8volt get]({{ "/cli/c8volt_get" | relative_url }})	 - Inspect cluster, process, job, element, incident, tenant, and resource state

