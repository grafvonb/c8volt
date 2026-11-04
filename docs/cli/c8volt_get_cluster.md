---
title: "c8volt get cluster"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get cluster

Inspect cluster-wide topology, version, and license information

### Synopsis

Inspect cluster topology, gateway and broker versions, or license information.

```
c8volt get cluster [flags]
```

### Examples

```
  c8volt get cluster topology
  c8volt get cluster version
  c8volt get cluster license
```

### Options

```
  -h, --help   help for cluster
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
* [c8volt get cluster license]({{ "/cli/c8volt_get_cluster_license" | relative_url }})	 - Show connected cluster license
* [c8volt get cluster topology]({{ "/cli/c8volt_get_cluster_topology" | relative_url }})	 - Inspect connected cluster topology
* [c8volt get cluster version]({{ "/cli/c8volt_get_cluster_version" | relative_url }})	 - Show connected cluster version

