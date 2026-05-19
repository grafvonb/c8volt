# c8volt VHS recordings

This directory contains reproducible terminal recordings for c8volt docs.

The checked-in files are recording source only. The real `config.yaml` used by
recordings is generated into `/tmp/c8volt-vhs/bin/config.yaml` so tapes can show
normal commands without `--config`, while OAuth secrets stay out of the
repository. Tapes use `--no-indicator` on longer live operations to avoid
transient spinner frames in the captured terminal.

## Prerequisites

- `vhs`
- `ttyd`
- `ffmpeg`
- `go`
- a reachable Camunda 8.9 cluster
- OAuth client credentials for that cluster

On macOS, VHS can be installed with Homebrew:

```bash
brew install vhs
```

VHS requires `ttyd` and `ffmpeg` on `PATH`.

## Environment

Export the recording-only variables before rendering:

```bash
export C8VOLT_VHS_C89_BASE_URL="https://camunda.example.com"
export C8VOLT_VHS_OAUTH_TOKEN_URL="https://login.example.com/oauth/token"
export C8VOLT_VHS_OAUTH_CLIENT_ID="c8volt"
export C8VOLT_VHS_OAUTH_CLIENT_SECRET="..."
```

Optional scope variables are also supported when the target OAuth provider
requires explicit per-API scopes:

```bash
export C8VOLT_VHS_OAUTH_CAMUNDA_SCOPE="camunda-api.write"
export C8VOLT_VHS_OAUTH_OPERATE_SCOPE="operate-api.read"
export C8VOLT_VHS_OAUTH_TASKLIST_SCOPE="tasklist-api.read"
```

## Render

Render a screencast with its Make target or short alias:

| Screencast | Make target | Short alias |
| --- | --- | --- |
| Fast Start | `make demo-vhs-fast-start` | |
| Execute Smoke Test | `make demo-vhs-ops-execute-smoke-test` | `make demo-vhs-st` |
| Execute Retention Policy | `make demo-vhs-ops-execute-retention-policy` | `make demo-vhs-rp` |
| Purge Orphan Process Instances | `make demo-vhs-ops-purge-orphan-process-instances` | `make demo-vhs-opi` |
| Purge Process Instances With Incidents | `make demo-vhs-ops-purge-process-instances-with-incidents` | `make demo-vhs-piwi` |
| Purge All Process Definitions | `make demo-vhs-ops-purge-all-process-definitions` | `make demo-vhs-apd` |
| Repair Incident | `make demo-vhs-ops-repair-incident` | `make demo-vhs-inc` |
| Repair Process Instance | `make demo-vhs-ops-repair-process-instance` | `make demo-vhs-pi` |

The render script builds `./bin/c8volt`, creates `/tmp/c8volt-vhs`, copies the
binary into `/tmp/c8volt-vhs/bin/c8volt`, writes the recording-only
`/tmp/c8volt-vhs/bin/config.yaml`, and runs VHS from the repository root.

Generated media is written to:

- `docs/assets/screencasts/fast-start.gif`
- `docs/assets/screencasts/ops-execute-smoke-test.gif`
- `docs/assets/screencasts/ops-execute-retention-policy.gif`
- `docs/assets/screencasts/ops-purge-orphan-process-instances.gif`
- `docs/assets/screencasts/ops-purge-process-instances-with-incidents.gif`
- `docs/assets/screencasts/ops-purge-all-process-definitions.gif`
- `docs/assets/screencasts/ops-repair-incident.gif`
- `docs/assets/screencasts/ops-repair-process-instance.gif`
