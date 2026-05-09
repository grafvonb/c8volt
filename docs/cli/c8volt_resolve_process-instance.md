---
title: "c8volt resolve process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt resolve process-instance

Resolve process-instance incidents by key

### Synopsis

Resolve process-instance incidents by key.

The command accepts repeated --key values or newline-separated keys from stdin with '-'. For each unique process instance, c8volt expands to the process-instance family, discovers active incidents at command start for direct incidents on in-scope instances, resolves that fixed incident set, and reports process instances with no active incidents as skipped.

By default c8volt validates the affected root and descendant instances and asks for confirmation before resolving active incidents in the family. Use --dry-run to preview the family scope and incident resolution plan without submitting mutations.

By default c8volt waits until the initially discovered incidents are no longer active by polling process-instance incident lookup through the incident service.

```
c8volt resolve process-instance [flags]
```

### Examples

```
  ./c8volt resolve process-instance --key 2251799813685250
  ./c8volt resolve pi --key 2251799813685250 --key 2251799813685260
  printf '%s\n' 2251799813685250 2251799813685260 | ./c8volt resolve process-instance -
  printf '%s\n' 2251799813685250 | ./c8volt resolve pi --key 2251799813685260 -
```

### Options

```
      --dry-run           preview process-instance incident resolutions without submitting mutation
      --fail-fast         stop scheduling new process-instance resolutions after the first error
  -h, --help              help for process-instance
  -k, --key strings       process instance key(s) to resolve; repeat or combine with stdin '-'
      --no-wait           return after resolution requests are accepted without incident confirmation
      --no-worker-limit   disable limiting the number of workers to GOMAXPROCS when --workers > 1
  -w, --workers int       maximum concurrent workers when resolving multiple process instances (default: min(count, GOMAXPROCS))
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

* [c8volt resolve](c8volt_resolve)	 - Resolve operational incidents

