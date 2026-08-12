# Contract: Isolated Alpha Client Generation

## Canonical invocation

```bash
bash api/refresh-clients.sh --target v810alpha --camunda-tag 8.10.0-alpha4
```

The exact option spelling may follow the existing shell parser, but this invocation is the documented and tested reproduction contract.

## Required tuple

| Item | Required value |
|------|----------------|
| Target | `v810alpha` |
| Upstream repository | `https://github.com/camunda/camunda.git` |
| Tag | `8.10.0-alpha4` |
| Resolved commit | `4a76f06c9df8ad0a6c64fe88b92c618c2ebf2ef6` |
| Source | `zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml` |
| Client output | `internal/clients/camunda/v810alpha/camunda/client.gen.go` |
| Provenance output | `internal/clients/camunda/v810alpha/camunda/provenance.json` |

## Guard order

1. Parse and validate target, tag, source, and destination.
2. Reject legacy docs options or any output outside the alpha directory.
3. Capture fingerprints for generated v8.7, v8.8, and v8.9 directories.
4. Resolve the tag and verify the expected commit.
5. Fetch sources into a temporary target-specific directory.
6. Bundle and mutate the source in temporary storage.
7. Generate the client and provenance into temporary files.
8. Validate generated syntax, provenance completeness, and stable-tree fingerprints.
9. Publish only the two alpha files.
10. Recheck stable-tree fingerprints and fail if any changed.

No repository output may be written before steps 1-4 pass.

## Ordered transformations

1. Redocly bundle of the relocated v2 specification.
2. `mutate-search-query-schemas.py`
3. `mutate-search-result-schemas.py`
4. `mutate-fix-process-instance-filter-fields.py`
5. `mutate-fix-jobresult-discriminator.py`
6. `mutate-fix-camunda-v2-operation-id-collisions.py`
7. oapi-codegen `types,client` generation for package `camunda`

## Provenance schema

```json
{
  "repository": "https://github.com/camunda/camunda.git",
  "tag": "8.10.0-alpha4",
  "commit": "4a76f06c9df8ad0a6c64fe88b92c618c2ebf2ef6",
  "sourceSpec": "zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml",
  "mutations": ["ordered repository-relative script names"],
  "generator": {"name": "oapi-codegen", "version": "2.5.0"},
  "command": "bash api/refresh-clients.sh --target v810alpha --camunda-tag 8.10.0-alpha4"
}
```

## Failure contract

- Wrong/missing tag: non-zero exit before repository writes.
- Unknown target: non-zero exit before fetch or writes.
- Target/output mismatch: non-zero exit before generation.
- Commit mismatch: non-zero exit before generation.
- Missing generator or bundler: non-zero exit with the missing tool named.
- Mutation or generation failure: non-zero exit; existing alpha output remains intact.
- Stable generated-tree difference: non-zero exit and no alpha publication.

## Stable workflow compatibility

Invoking `bash api/refresh-clients.sh` without `--target v810alpha` retains the existing all-client behavior. Alpha mode skips legacy docs fetching, auth generation, and all stable client outputs.
