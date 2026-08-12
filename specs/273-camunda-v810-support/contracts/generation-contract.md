# Contract: Isolated Camunda V810 Client Generation

## Canonical Invocation

```bash
bash api/refresh-clients.sh --target v810 --camunda-tag 8.10.0-alpha4
```

Later 8.10 alpha, RC, and final tags use the same target and replace output in place.

## Initial Required Tuple

| Item | Required value |
|------|----------------|
| Target | `v810` |
| Repository | `https://github.com/camunda/camunda.git` |
| Tag | `8.10.0-alpha4` |
| Peeled commit | `4a76f06c9df8ad0a6c64fe88b92c618c2ebf2ef6` |
| Source | `zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml` |
| Package | `camunda` |
| Client output | `internal/clients/camunda/v810/camunda/client.gen.go` |
| Provenance | `internal/clients/camunda/v810/camunda/provenance.json` |

## Ordered Preparation

1. Validate target, tag, commit policy, and target path before writes.
2. Fingerprint protected v87-v89 generated trees.
3. Resolve the peeled tag commit and verify the initial pin.
4. Sparse-fetch product v2 source into a `mktemp` directory with cleanup trap.
5. Bundle the v2 spec in temporary storage.
6. Apply in order:
   1. `api/mutations/mutate-search-query-schemas.py`
   2. `api/mutations/mutate-search-result-schemas.py`
   3. `api/mutations/mutate-fix-process-instance-filter-fields.py`
   4. `api/mutations/mutate-fix-jobresult-discriminator.py`
   5. `api/mutations/mutate-fix-camunda-v2-operation-id-collisions.py`
7. Assert transformation effects; generate package `camunda` in temporary storage.
8. Validate Go syntax/compile and required symbols.
9. Create deterministic provenance and verify hashes.
10. Recheck protected fingerprints and repository output allowlist.
11. Publish the client/provenance as one replacement unit; recheck protected trees.

No repository output is written before source identity and target validation succeeds.

## Provenance Shape

```json
{
  "schemaVersion": 1,
  "repository": "https://github.com/camunda/camunda.git",
  "tag": "8.10.0-alpha4",
  "commit": "4a76f06c9df8ad0a6c64fe88b92c618c2ebf2ef6",
  "sourceSpec": "zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml",
  "sourceSpecSha256": "<sha256>",
  "transformations": [{"path": "api/mutations/<script>.py", "sha256": "<sha256>"}],
  "tools": {"redocly": "<exact version>", "oapi-codegen": "2.5.0"},
  "preparedSpecSha256": "<sha256>",
  "generatedClientSha256": "<sha256>",
  "command": "bash api/refresh-clients.sh --target v810 --camunda-tag 8.10.0-alpha4"
}
```

The transformations array contains all five scripts in execution order. No timestamps, host/temp paths, or SSH-only repository spelling are recorded.

## Failure Contract

- Unknown/missing target, invalid tag, commit mismatch, or output escape: non-zero before writes.
- Missing tool: non-zero naming the dependency.
- Transformation no-op, required-symbol absence, generation/syntax/compile/provenance failure: non-zero; existing v810 output intact.
- Protected-tree/output-allowlist difference: non-zero with no publication.

## Compatibility and Determinism

- No-target refresh preserves existing all-client behavior.
- V810 mode skips legacy docs/auth generation and creates no removed component clients.
- An identical second run produces no diff.
- Baseline updates change existing v810 output only; no `v810alpha`/`v810rc` family is created.
