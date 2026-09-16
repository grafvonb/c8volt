---
title: "c8volt walk process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt walk process-instance

Inspect the parent/child tree of process instances

### Synopsis

Inspect process-instance ancestry, descendants, or the full family.

Use --parent for ancestry or --children for descendants; the default scope is the full family. Explicit --key uses backend authorization without tenant filtering.

Add --with-incidents, --with-vars, or --with-elements for incident details, process-instance-scope variables, or runtime elements. Add --with-listeners to --with-elements for runtime listener jobs. Listener rows use s: for job creation (not worker execution start), e: for job end, and d: for an available deadline only while the state is exactly ACTIVATED; missing times are omitted.

When an ancestor is missing but reachable family data remains, walk returns the available family. Direct single-resource lookups remain strict.

```
c8volt walk process-instance [flags]
```

### Examples

```
  ./c8volt walk process-instance --key <process-instance-key>
  ./c8volt walk process-instance --key <process-instance-key> --with-incidents
  ./c8volt walk process-instance --key <process-instance-key> --with-vars
  ./c8volt walk process-instance --key <process-instance-key> --with-elements
  ./c8volt walk process-instance --key <process-instance-key> --with-elements --with-listeners
  ./c8volt walk process-instance --key <process-instance-key> --flat
  ./c8volt walk process-instance --key <process-instance-key> --parent
```

### Options

```
      --children                     show descendants from the selected process instance
      --flat                         render family output as a flat path instead of an ASCII tree
  -h, --help                         help for process-instance
      --incident-message-limit int   maximum characters to show for incident messages when --with-incidents is set; 0 disables truncation
      --incident-state string        incident state scope for --with-incidents: active, pending, resolved, migrated, unknown, all (default "active")
  -k, --key string                   start walking from this process instance key
      --parent                       show ancestry from the selected process instance toward the root
      --var-value-limit int          maximum characters to show for variable values when --with-vars is set; 0 disables truncation
      --with-elements                show runtime element instances for keyed process-instance walks
      --with-incidents               show incident keys, states, and messages for keyed process-instance walks
      --with-listeners               include runtime listener jobs; requires --with-elements
      --with-vars                    show process-instance-scope variables for keyed process-instance walks
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

* [c8volt walk]({{ "/cli/c8volt_walk" | relative_url }})	 - Inspect process-instance relationships

