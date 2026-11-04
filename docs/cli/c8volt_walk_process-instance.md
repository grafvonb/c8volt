---
title: "c8volt walk process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt walk process-instance

Inspect the parent/child tree of process instances

### Synopsis

Inspect the process-instance family around a key.

Use --parent for ancestry or --children for descendants; the default includes
both. Add --with-incidents, --with-vars, or --with-elements for details.
Add --with-listeners to --with-elements for listener jobs.

Alternatively, select by exact --bid (only c8.10+). It must identify one instance;
multiple matches are ambiguous.

```
c8volt walk process-instance [flags]
```

### Examples

```
  c8volt walk process-instance --key <process-instance-key>
  c8volt walk process-instance --key <process-instance-key> --with-incidents
  c8volt walk process-instance --key <process-instance-key> --with-elements --with-listeners
```

### Options

```
      --bid string                   exact business ID for one unambiguous starting instance (only c8.10+)
      --children                     show descendants from the selected process instance
      --flat                         render family output as a flat path instead of an ASCII tree
  -h, --help                         help for process-instance
      --incident-message-limit int   maximum characters to show for incident messages when --with-incidents is set; 0 disables truncation
      --incident-state string        incident state scope for --with-incidents: active, pending, resolved, migrated, unknown, all (default "active")
  -k, --key string                   start walking from this process instance key
      --parent                       show ancestry from the selected process instance toward the root
      --var-value-limit int          maximum characters to show for variable values when --with-vars is set; 0 disables truncation
      --with-elements                show runtime element instances for selected process-instance walks
      --with-incidents               show incident keys, states, and messages for selected process-instance walks
      --with-listeners               include runtime listener jobs; requires --with-elements
      --with-vars                    show process-instance-scope variables for selected process-instance walks
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

* [c8volt walk]({{ "/cli/c8volt_walk" | relative_url }})	 - Inspect process-instance relationships

