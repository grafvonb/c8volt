---
title: "c8volt ops execute api-latency-test"
nav_exclude: true
---

## c8volt ops execute api-latency-test

Execute a bounded active API latency test

### Synopsis

Execute a bounded active API latency test.

The command deploys the version-matched SimpleUserTask fixture, creates process instances, measures create, read, and search-visibility latency under bounded load, then cleans up exact run-owned resources.

One concrete tenant is required. Use --dry-run to preview the plan without mutation and --no-cleanup to retain created resources. --count sets the total sample budget and --workers sets the maximum concurrency. Use --verbose for stage details and --report-file to save a Markdown or JSON report.

```
c8volt ops execute api-latency-test [flags]
```

### Examples

```
  ./c8volt ops execute api-latency-test --dry-run
  ./c8volt ops execute api-latency-test --count 20 --workers 4 --auto-confirm
  ./c8volt ops execute api-latency-test -n 7 -w 4 --dry-run
  ./c8volt --verbose ops execute api-latency-test --auto-confirm
  ./c8volt ops execute api-latency-test --no-cleanup --auto-confirm
```

### Options

```
  -n, --count int              primary process-instance create samples across active stages (default 20)
      --dry-run                validate and preview the active API latency plan without mutation
  -h, --help                   help for api-latency-test
      --no-cleanup             retain active API latency resources after execution
      --report-file string     write an API latency test report to the given path
      --report-format string   API latency test report format: markdown, json (default inferred from report-file extension)
  -w, --workers int            maximum closed-loop workers and final active stage width (default 4)
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

* [c8volt ops execute]({{ "/cli/c8volt_ops_execute" | relative_url }})	 - Discover predefined operational playbooks

