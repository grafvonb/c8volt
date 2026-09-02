---
title: "c8volt ops analyse api-latency"
nav_exclude: true
---

## c8volt ops analyse api-latency

Analyse API latency without changing cluster state

### Synopsis

Analyse API latency without changing cluster state.

The command is read-only. It measures cluster topology, process-definition search/read, and process-instance search/read paths in bounded closed-loop stages. Search-derived keyed reads reuse keys returned by the measured searches when available and supported.

--count is the total primary sample-cycle budget across all stages. --workers is the maximum closed-loop worker count and final stage width. The count must be large enough to exercise every stage width.

Default output is compact stage evidence with findings, notices, limitations, and outcome. JSON output uses the shared command envelope. Keys-only output is not meaningful for this diagnostic and is rejected.

```
c8volt ops analyse api-latency [flags]
```

### Examples

```
  ./c8volt ops analyse api-latency
  ./c8volt ops analyse api-latency --count 20 --workers 4
  ./c8volt ops analyse api-latency -n 6 -w 2
  ./c8volt ops analyse api-latency --json
```

### Options

```
  -n, --count int              primary sample cycles to measure across all read-only stages (default 20)
  -h, --help                   help for api-latency
      --report-file string     write an API latency report to the given path
      --report-format string   API latency report format: markdown, json (default inferred from report-file extension)
  -w, --workers int            maximum closed-loop workers and final stage width (default 4)
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

* [c8volt ops analyse]({{ "/cli/c8volt_ops_analyse" | relative_url }})	 - Discover read-only operational analyses

