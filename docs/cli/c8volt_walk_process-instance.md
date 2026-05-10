---
title: "c8volt walk process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt walk process-instance

Inspect the parent/child tree of process instances

### Synopsis

Inspect the parent/child tree of process instances.

By default, walk shows the full process-instance family as an ASCII tree. Use --parent for ancestry, --children for descendants, or --flat for a path-style family view.

Add --with-incidents and/or --with-vars to keyed walks to show incident details and process-instance-scope variables below matching rows.

When an ancestor is missing but reachable family data still exists, walk returns the partial tree plus a warning. Direct single-resource lookups stay strict.

```
c8volt walk process-instance [flags]
```

### Examples

```
  ./c8volt walk pi --key <process-instance-key>
  ./c8volt walk pi --key <process-instance-key> --with-incidents
  ./c8volt walk pi --key <process-instance-key> --with-vars
  ./c8volt walk pi --key <process-instance-key> --with-vars --with-incidents
  ./c8volt walk pi --key <process-instance-key> --with-incidents --incident-message-limit 80
  ./c8volt walk pi --key <process-instance-key> --with-incidents --incident-state all
  ./c8volt walk pi --key <process-instance-key> --flat
  ./c8volt walk pi --key <process-instance-key> --parent
  ./c8volt --json walk pi --key <process-instance-key> --children --with-incidents
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
      --with-incidents               show incident keys, states, and messages for keyed process-instance walks
      --with-vars                    show process-instance-scope variables for keyed process-instance walks
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

* [c8volt walk](c8volt_walk)	 - Inspect process-instance relationships

