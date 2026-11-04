---
title: "c8volt ops execute smoke-test"
nav_exclude: true
---

## c8volt ops execute smoke-test

Execute a cluster smoke test workflow

### Synopsis

Test the connection, deploy a bundled fixture, start instances, and inspect
their process families.

Cleanup removes created instances and fixture definitions unless unrelated
instances still use them. Use --dry-run to preview the steps or --report-file
to save the result. This workflow creates and deletes cluster data.

```
c8volt ops execute smoke-test [flags]
```

### Examples

```
  c8volt ops execute smoke-test --dry-run
  c8volt ops execute smoke-test --report-file smoke-test.md
```

### Options

```
  -n, --count int              number of process instances to create from the deployed smoke-test definition (default 1)
      --dry-run                validate the smoke-test plan without submitting mutation requests
      --fail-fast              stop scheduling smoke-test work after the first error
  -h, --help                   help for smoke-test
      --no-cleanup             retain created process instances and all deployed fixture process definitions
      --no-wait                return after cleanup requests are accepted without deletion confirmation
      --no-worker-limit        use all queued smoke-test jobs as workers when --workers is unset
      --report-file string     write an audit report to the given path
      --report-format string   audit report format: markdown, json (default inferred from report-file extension)
  -w, --workers int            maximum concurrent workers; omit for an automatic limit
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

* [c8volt ops execute]({{ "/cli/c8volt_ops_execute" | relative_url }})	 - Discover predefined operational playbooks

