---
title: "c8volt get cluster version"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get cluster version

Show connected cluster version

### Synopsis

Get the connected Camunda gateway version.

Use --with-brokers to include broker versions.

```
c8volt get cluster version [flags]
```

### Examples

```
  ./c8volt get cluster version
  ./c8volt get cluster version --with-brokers
  ./c8volt get cluster version --json
```

### Options

```
  -h, --help           help for version
      --with-brokers   include broker versions
```

### Options inherited from parent commands

```
      --all-tenants        clear configured tenant filtering and search all tenants visible to the authenticated user; mutually exclusive with --tenant
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
      --tenant string      tenant ID for discovery/search, selection, create, deploy, and run flows; explicit empty values can clear configured discovery filters, and explicit keys/IDs remain backend-authorized
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt get cluster]({{ "/cli/c8volt_get_cluster" | relative_url }})	 - Inspect cluster-wide topology, version, and license information

