# Camunda client generation

Use the dedicated flow to regenerate the Camunda 8.10 client:

```sh
bash api/refresh-clients.sh --target v810 --camunda-tag 8.10.3
```

Run generation from the repository root. It fetches the pinned upstream OpenAPI source, applies local mutations, and generates Go code. It does not contact a running Camunda cluster.

## Client inventory

| Version | Runtime clients | Refresh |
| --- | --- | --- |
| 8.10 | Unified Camunda API | `--target v810 --camunda-tag <tag>` |
| 8.9 | Unified API, Tasklist | Default refresh |
| 8.8 | Unified API, Operate, Tasklist | Default refresh |

The default `bash api/refresh-clients.sh` refreshes retained 8.8/8.9 clients and OAuth2. It does not refresh 8.10. Do not remove component clients with runtime consumers.

## Update the 8.10 generated client

1. Choose the upstream tag and run the dedicated refresh command.
2. Check `internal/clients/camunda/v810/camunda/provenance.json` and `client.gen.go`.
3. Align `toolx/camunda_baseline.go` with the provenance tag, commit, and release status.
4. Run the isolated guards below, relevant Go tests, and `make docs-content`.

The generated client uses upstream tag `8.10.3`, commit `3961b79c3fdaf43c48691ed77672d6db42d0abcc`, for reproducible generation. A later source pin replaces the same `v810/camunda` client; runtime selection remains `8.10`.

The source is `zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml` in `camunda/camunda`. Provenance records source and generated hashes, ordered mutation hashes, tool versions, and the reproduction command. Publication stages files and restores the previous directory if the final move fails. Older client directories remain unchanged.

## Validation

These checks use local files or test-owned fixtures:

```sh
bash api/tests/v810_generation_test.sh
python3 api/tests/v810_provenance_test.py
bash api/tests/v810_repository_boundary_test.sh
go test ./internal/clients/camunda -run '^TestCamundaClient' -count=1
```

The boundary guard can also wrap generation:

```sh
bash api/tests/v810_repository_boundary_test.sh -- bash api/refresh-clients.sh --target v810 --camunda-tag 8.10.3
```

Networked reproducibility is separate. `C8VOLT_V810_NETWORK_REPRODUCIBILITY=1 python3 api/tests/v810_provenance_test.py` fetches the pinned tag and generates twice in a temporary worktree using the recorded tools. It does not refresh the developer checkout.

## Generation scripts

| Script | Purpose |
| --- | --- |
| `1-fetch-camunda-product-v2-spec.sh` | Fetch unified API specs from `camunda/camunda` |
| `1-fetch-camunda-docs-api-specs.sh` | Fetch legacy component specs from `camunda/camunda-docs` |
| `3-generate-clients-from-fetched-specs.sh` | Generate retained 8.8/8.9 and OAuth2 clients from fetched specs |
| `generate-go-client.sh` | Generate one client with `oapi-codegen` |
| `2-bundle-camunda-docs-v2-spec.sh` | Optional legacy bundler, outside the current refresh pipeline |

The 8.10 mutation chain under `api/mutations/` preserves search queries, search results, process-instance filters, and job-result discrimination, then resolves operation-ID collisions. Keep its order and hashes aligned with provenance. Prefer changing this chain and regenerating over editing generated clients.

`--commit` on the refresh and retained-generation scripts stages generated clients and creates a Conventional Commit with upstream refs. Use it only when a commit is intended.
