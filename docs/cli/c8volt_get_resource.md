---
title: "c8volt get resource"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get resource

Get a resource by key

### Synopsis

Retrieve deployed resource metadata with --id.

Use the resourceKey from the resource's deployment metadata, rather than
resourceId, deploymentKey, or processDefinitionKey. Keys-only output contains
the resource ID; JSON includes its resource key. For BPMN XML, use
get process-definition with --key and --xml.

```
c8volt get resource [flags]
```

### Examples

```
  c8volt get resource --id <resource-key>
  c8volt --json get resource --id <resource-key>
```

### Options

```
  -h, --help        help for resource
  -i, --id string   resource key to fetch
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

* [c8volt get]({{ "/cli/c8volt_get" | relative_url }})	 - Inspect cluster, process, job, element, incident, tenant, and resource state

