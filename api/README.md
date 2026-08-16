# Camunda Client Generation

This directory contains the scripts and patch helpers used to generate Go clients from Camunda OpenAPI specifications.

## Upstream sources

- `/v2` Camunda API: fetched from `camunda/camunda`
- legacy component APIs such as Operate, Tasklist, and Administration SM: fetched from `camunda/camunda-docs`

The local fetch scripts pin to a fetched Git ref so generation stays reproducible.

## Main scripts

- `1-fetch-camunda-product-v2-spec.sh`
  Fetches the `/v2` OpenAPI source from the Camunda product repository.
- `1-fetch-camunda-docs-api-specs.sh`
  Fetches the legacy component API specs from `camunda-docs`.
- `2-bundle-camunda-docs-v2-spec.sh`
  Bundles the legacy docs-repo `/v2` spec into a single YAML file.
- `3-generate-clients-from-fetched-specs.sh`
  Regenerates the checked-in Go clients under `internal/clients/` from already fetched sources.
- `refresh-clients.sh`
  Runs the full workflow: fetch upstream specs and regenerate the checked-in Go clients.
- `generate-go-client.sh`
  Runs `oapi-codegen` for a single spec file and output path.

These wrappers still point to the docs-backed fetch flow.

## Mutation helpers

Some generated `/v2` types still need local spec patching before `oapi-codegen` produces usable Go types.

The Python mutation helpers are stored in `api/mutations/`.

Currently used for the product-repo `/v2` flow:

- `mutations/mutate-search-query-schemas.py`
  Preserves `filter` and `sort` fields on generated search request types.
- `mutations/mutate-search-result-schemas.py`
  Preserves `items` fields on generated search result types.
- `mutations/mutate-fix-process-instance-filter-fields.py`
  Preserves process-instance-specific filter fields that would otherwise collapse into the base filter type.
- `mutations/mutate-fix-jobresult-discriminator.py`
  Restores discriminator-related `type` fields needed for correct generated models.

Other patch helpers in this directory may still be used for older or component-specific generation paths.

## Typical workflow

1. Fetch upstream specs:
   - `bash api/1-fetch-camunda-product-v2-spec.sh`
   - `bash api/1-fetch-camunda-docs-api-specs.sh`
2. Regenerate clients:
   - `bash api/3-generate-clients-from-fetched-specs.sh`
3. Validate:
   - `make test`

Full workflow:

1. Fetch and regenerate in one step:
   - `bash api/refresh-clients.sh`
2. Validate:
   - `make test`

## Camunda 8.10 isolated baseline

Camunda 8.10 uses one c8volt compatibility identity, `8.10`, backed by one
checked-in unified client under `internal/clients/camunda/v810/camunda`. The
initial active source baseline is Camunda `8.10.0-alpha4` at peeled commit
`4a76f06c9df8ad0a6c64fe88b92c618c2ebf2ef6`.

Reproduce the active baseline from the repository root with the command recorded
in provenance:

- `bash api/refresh-clients.sh --target v810 --camunda-tag 8.10.0-alpha4`

The command publishes only these V810 artifacts:

- `internal/clients/camunda/v810/camunda/client.gen.go`
- `internal/clients/camunda/v810/camunda/provenance.json`

`provenance.json` is deterministic and records the source evidence needed for
audit:

- schema version
- upstream repository
- Camunda tag
- peeled commit
- source OpenAPI path
- source spec SHA-256
- ordered mutation script paths and SHA-256 values
- Redocly and `oapi-codegen` versions
- prepared spec SHA-256
- generated client SHA-256
- canonical reproduction command

The source OpenAPI path is
`zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml`. The ordered mutation
chain is:

1. `api/mutations/mutate-search-query-schemas.py`
2. `api/mutations/mutate-search-result-schemas.py`
3. `api/mutations/mutate-fix-process-instance-filter-fields.py`
4. `api/mutations/mutate-fix-jobresult-discriminator.py`
5. `api/mutations/mutate-fix-camunda-v2-operation-id-collisions.py`

To advance the active baseline to a later Camunda 8.10 alpha, release candidate,
or final tag, keep the same target and replace only the tag value:

- `bash api/refresh-clients.sh --target v810 --camunda-tag 8.10.0-rc1`
- `bash api/refresh-clients.sh --target v810 --camunda-tag 8.10.0`

Do not create `v810alpha`, `v810rc`, `v810final`, or another operator version
identity. Later baselines replace the existing `v810/camunda` client and
provenance in place while the user-facing c8volt version remains `8.10`.

The isolated V810 generator validates target path, tag shape, initial tag commit,
required tools, mutation effects, generated symbols, generated-client compile,
provenance identity, and generated-client hash before publication. Publication
stages replacement files next to the target directory, backs up the previous
`v810/camunda` directory, and restores that backup if the final move fails.

The following protected paths must remain unchanged during V810 generation:

- `internal/clients/camunda/v87`
- `internal/clients/camunda/v88`
- `internal/clients/camunda/v89`
- `integration`

Run the V810 generation guards before accepting a baseline update:

- `bash api/tests/v810_generation_test.sh`
- `python3 api/tests/v810_provenance_test.py`
- `bash api/tests/v810_repository_boundary_test.sh`

## Commit mode

`api/refresh-clients.sh` and `api/3-generate-clients-from-fetched-specs.sh` support `--commit`.

Example:

- `bash api/refresh-clients.sh --commit`

When enabled, the script:

- stages the generated client files under `internal/clients/`
- creates a Conventional Commit
- includes the fetched `camunda` and `camunda-docs` refs in the commit body

Optional source pinning:

- `bash api/refresh-clients.sh --camunda-tag 8.8.19 --camunda-docs-tag 8.8.196`
