# Research: Improve Cluster Info Output And Version Command

## Decision 1: Use human-readable defaults with JSON gated by `--json`

- **Decision**: Render topology and license as human-readable text by default, and keep structured result-envelope JSON only when `--json` is selected.
- **Rationale**: The issue asks for operator-friendly defaults while explicitly preserving machine-readable output through `--json`. The existing `pickMode()` and `renderJSONPayload` patterns already make this split clear for command handlers.
- **Alternatives considered**:
  - Always print JSON plus human text: rejected because it would break both interactive readability and machine parsing.
  - Add a new `--tree` flag for topology: rejected because the issue requests tree output as the default.

## Decision 2: Keep cluster version backed by topology retrieval

- **Decision**: Add `get cluster version` under `get cluster` and implement it by reusing the same topology retrieval used by `get cluster topology`.
- **Rationale**: The topology response already includes `GatewayVersion` and broker version/host details. Reusing it avoids a new service method, avoids another upstream call, and matches the issue requirement.
- **Alternatives considered**:
  - Add a separate service method: rejected because it would duplicate the same upstream topology call.
  - Parse version output from rendered topology text: rejected because commands should share domain data, not parse their own presentation output.

## Decision 3: Remove direct legacy topology command and aliases

- **Decision**: Remove `get cluster-topology` and its direct aliases (`ct`, `cluster-info`, `ci`) from the public command tree.
- **Rationale**: The clarification answer explicitly requests removing the old direct command. Keeping aliases would preserve the same legacy path under different spellings.
- **Alternatives considered**:
  - Keep `cluster-topology` as deprecated: rejected because it contradicts the clarification.
  - Hide but keep the command: rejected because scripts would still rely on the removed behavior and capability/help tests could miss the compatibility break.

## Decision 4: Add focused command-view helpers in `cmd`

- **Decision**: Put cluster topology tree, cluster version, and flat license formatting in a command-view helper file such as `cmd/cmd_views_cluster.go`.
- **Rationale**: Existing human rendering lives in `cmd` view helpers, while the cluster service and domain packages stay responsible for fetching and mapping data.
- **Alternatives considered**:
  - Add rendering methods to `internal/domain`: rejected because domain structs should remain presentation-agnostic.
  - Render inline in each Cobra handler: rejected because tests and future maintenance are cleaner with focused helpers.

## Decision 5: Refresh generated docs from command metadata

- **Decision**: Update Cobra command metadata and source docs, then regenerate CLI docs with `make docs-content`.
- **Rationale**: The repository convention says generated or generated-adjacent docs should be updated from source metadata rather than hand-edited.
- **Alternatives considered**:
  - Edit generated docs by hand: rejected because it risks drift from the live Cobra tree.

## Decision 6: Remove the generated legacy topology page after command removal

- **Decision**: Let `make docs-content` regenerate `docs/cli/` after `get cluster-topology` is removed, then delete `docs/cli/c8volt_get_cluster-topology.md` if the generator no longer emits it.
- **Rationale**: `docs-content` runs `go run -ldflags "$(LDFLAGS)" ./docsgen -out ./docs/cli -format markdown`, so generated CLI pages should reflect the live Cobra tree. Because generators may not remove stale files automatically, final polish must explicitly check that the legacy page is gone.
- **Alternatives considered**:
  - Keep a redirected legacy page: rejected because the contract says the command should no longer be documented.
