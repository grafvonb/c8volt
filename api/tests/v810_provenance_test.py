#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

"""Validate deterministic provenance for the isolated Camunda v8.10 client."""

from __future__ import annotations

import hashlib
import json
import os
import re
import subprocess
import unittest
from pathlib import Path
from typing import Any


REPO_ROOT = Path(__file__).resolve().parents[2]
PROVENANCE_PATH = (
    REPO_ROOT / "internal/clients/camunda/v810/camunda/provenance.json"
)
GENERATED_CLIENT_PATH = (
    REPO_ROOT / "internal/clients/camunda/v810/camunda/client.gen.go"
)
CANONICAL_COMMAND = (
    "bash api/refresh-clients.sh --target v810 --camunda-tag 8.10.0-alpha4"
)
EXPECTED_COMMIT = "4a76f06c9df8ad0a6c64fe88b92c618c2ebf2ef6"
EXPECTED_TRANSFORMATIONS = [
    "api/mutations/mutate-search-query-schemas.py",
    "api/mutations/mutate-search-result-schemas.py",
    "api/mutations/mutate-fix-process-instance-filter-fields.py",
    "api/mutations/mutate-fix-jobresult-discriminator.py",
    "api/mutations/mutate-fix-camunda-v2-operation-id-collisions.py",
]
EXPECTED_KEYS = {
    "schemaVersion",
    "repository",
    "tag",
    "commit",
    "sourceSpec",
    "sourceSpecSha256",
    "transformations",
    "tools",
    "preparedSpecSha256",
    "generatedClientSha256",
    "command",
}
DISALLOWED_KEY_FRAGMENTS = (
    "time",
    "date",
    "timestamp",
    "generated_at",
    "generatedAt",
    "host",
    "hostname",
    "tmp",
    "temp",
)
SHA256_RE = re.compile(r"^[0-9a-f]{64}$")


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def load_provenance() -> dict[str, Any]:
    with PROVENANCE_PATH.open("r", encoding="utf-8") as handle:
        return json.load(handle)


def walk_json(value: Any) -> list[Any]:
    values = [value]
    if isinstance(value, dict):
        for item in value.values():
            values.extend(walk_json(item))
    elif isinstance(value, list):
        for item in value:
            values.extend(walk_json(item))
    return values


class V810ProvenanceTest(unittest.TestCase):
    """Contract checks for reproducible v8.10 generation evidence."""

    def test_schema_and_canonical_source_identity(self) -> None:
        """The provenance file must identify exactly one pinned v8.10 baseline."""
        provenance = load_provenance()

        self.assertEqual(EXPECTED_KEYS, set(provenance))
        self.assertEqual(1, provenance["schemaVersion"])
        self.assertEqual("https://github.com/camunda/camunda.git", provenance["repository"])
        self.assertEqual("8.10.0-alpha4", provenance["tag"])
        self.assertEqual(EXPECTED_COMMIT, provenance["commit"])
        self.assertEqual(
            "zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml",
            provenance["sourceSpec"],
        )
        self.assertEqual(CANONICAL_COMMAND, provenance["command"])

    def test_transformations_are_ordered_and_hashed(self) -> None:
        """The mutation chain is recorded in the same order the generator applies it."""
        provenance = load_provenance()
        transformations = provenance["transformations"]

        self.assertEqual(EXPECTED_TRANSFORMATIONS, [item["path"] for item in transformations])
        for item in transformations:
            self.assertEqual({"path", "sha256"}, set(item))
            self.assertRegex(item["sha256"], SHA256_RE)
            self.assertEqual(sha256_file(REPO_ROOT / item["path"]), item["sha256"])

    def test_recorded_hashes_match_current_artifacts(self) -> None:
        """Content hashes must agree with the checked-in generated client."""
        provenance = load_provenance()

        self.assertRegex(provenance["sourceSpecSha256"], SHA256_RE)
        self.assertRegex(provenance["preparedSpecSha256"], SHA256_RE)
        self.assertRegex(provenance["generatedClientSha256"], SHA256_RE)
        self.assertEqual(
            sha256_file(GENERATED_CLIENT_PATH),
            provenance["generatedClientSha256"],
        )

    def test_provenance_has_no_nondeterministic_fields_or_paths(self) -> None:
        """Machine-readable evidence must avoid timestamps and host-local paths."""
        provenance = load_provenance()

        for key in provenance:
            normalized = key.lower()
            for fragment in DISALLOWED_KEY_FRAGMENTS:
                self.assertNotIn(fragment.lower(), normalized)

        for value in walk_json(provenance):
            if isinstance(value, str):
                self.assertNotIn(str(Path.home()), value)
                self.assertNotIn("/tmp/", value)
                self.assertNotIn("/var/folders/", value)
                self.assertNotIn("\\Temp\\", value)

    def test_second_generation_run_is_diff_free(self) -> None:
        """Running the canonical generator twice must not change published artifacts."""
        for _ in range(2):
            result = subprocess.run(
                ["bash", "api/refresh-clients.sh", "--target", "v810", "--camunda-tag", "8.10.0-alpha4"],
                cwd=REPO_ROOT,
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                check=False,
            )
            self.assertEqual(0, result.returncode, result.stdout)

        diff = subprocess.check_output(
            [
                "git",
                "diff",
                "--",
                "internal/clients/camunda/v810/camunda/client.gen.go",
                "internal/clients/camunda/v810/camunda/provenance.json",
            ],
            cwd=REPO_ROOT,
            text=True,
        )
        self.assertEqual("", diff)


if __name__ == "__main__":
    unittest.main()
