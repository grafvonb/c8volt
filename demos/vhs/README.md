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

Render the fast-start screencast:

```bash
make demo-vhs-fast-start
```

The render script builds `./bin/c8volt`, creates `/tmp/c8volt-vhs`, copies the
binary into `/tmp/c8volt-vhs/bin/c8volt`, writes the recording-only
`/tmp/c8volt-vhs/bin/config.yaml`, and runs VHS from the repository root.

Generated media is written to:

- `docs/assets/screencasts/fast-start.gif`
