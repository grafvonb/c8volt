---
title: "c8volt walk process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt walk process-instance

Inspect the parent/child tree of process instances

### Synopsis

Inspect the parent/child tree of process instances.

Choose --parent for ancestry, --children for descendants, and --family for the combined view. Add --tree with --family for an ASCII tree.

Add --with-incidents to keyed walks to show incident keys and messages below matching process-instance rows.

When an ancestor is missing but reachable family data still exists, walk returns the partial tree plus a warning. Direct single-resource lookups stay strict.

```
c8volt walk process-instance [flags]
```

### Examples

```
  ./c8volt walk pi --key 2251799813711967 --family
  ./c8volt walk pi --key 2251799813711967 --family --tree
  ./c8volt walk pi --key 2251799813711967 --family --with-incidents
  ./c8volt walk pi --key 2251799813711977 --parent
  ./c8volt --json walk pi --key 2251799813711967 --children --with-incidents
```

### Options

```
      --children         shorthand for --mode=children
      --family           shorthand for --mode=family
  -h, --help             help for process-instance
  -k, --key string       start walking from this process instance key
      --mode string      walk mode: parent, children, family (default "children")
      --parent           shorthand for --mode=parent
      --tree             render family mode as an ASCII tree (only valid with --family)
      --with-incidents   show incident keys and messages for keyed process-instance walks
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

