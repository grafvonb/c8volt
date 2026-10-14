#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

"""Validate deterministic provenance for the isolated Camunda v8.10 client."""

from __future__ import annotations

import hashlib
import json
import os
import re
import shutil
import subprocess
import tempfile
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
EXPECTED_TAG = "8.10.3"
EXPECTED_COMMIT = "3961b79c3fdaf43c48691ed77672d6db42d0abcc"
EXPECTED_SOURCE_SPEC_SHA256 = "857109dabcb998801f749282836a6d2af1a137284d538161801f0c348ae57837"
EXPECTED_PREPARED_SPEC_SHA256 = "461a1cc051a021a3cf937dbfd28a92537cf9b8b4381bcfb9ad6ab6e1d1f91d1b"
EXPECTED_GENERATED_CLIENT_SHA256 = "589351ac972ab595af721f2e9647a273819e56a9601f7af2aaaf98e6def2f319"
EXPECTED_OUTPUT_PARTS = ("internal", "clients", "camunda", "v810", "camunda")
EXPECTED_TRANSFORMATIONS = {
    "api/mutations/mutate-search-query-schemas.py": "3687396193df5ade934b1b275ae525801e6f9b27d92f23cdd81ae205898b031a",
    "api/mutations/mutate-search-result-schemas.py": "4a20bdb68eb30e050010daeb49cdee4df117cacc5ea2f6012a66cf2b924bda55",
    "api/mutations/mutate-fix-process-instance-filter-fields.py": "4bdf30a8b58a8c1a6a39882d90e41528dacc592fe8baa6cb6c6c5ed4d807ae64",
    "api/mutations/mutate-fix-jobresult-discriminator.py": "529946717b8a19f75594bcd5e21f370b283911549e628b1cce06edb7e416530f",
    "api/mutations/mutate-fix-camunda-v2-operation-id-collisions.py": "ffefa6aecb897d32fca4f3af93a26bc1127554e5119d2e2579ca8cdf5b45bfd9",
}
EXPECTED_TOOLS = {"redocly": "2.29.2", "oapi-codegen": "2.5.0"}
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


def canonical_command_for_tag(tag: str) -> str:
    return f"bash api/refresh-clients.sh --target v810 --camunda-tag {tag}"


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


def fingerprint_tree(root: Path) -> str:
    """Hash file paths and contents so protected-client changes are observable."""
    digest = hashlib.sha256()
    for path in sorted(item for item in root.rglob("*") if item.is_file()):
        digest.update(path.relative_to(root).as_posix().encode("utf-8") + b"\0")
        digest.update(bytes.fromhex(sha256_file(path)))
    return digest.hexdigest()


def apply_working_api_changes(checkout: Path) -> None:
    """Overlay current API edits so local validation exercises the proposed guard."""
    diff = subprocess.check_output(
        ["git", "diff", "--binary", "HEAD", "--", "api"], cwd=REPO_ROOT
    )
    if diff:
        subprocess.run(["git", "apply"], cwd=checkout, input=diff, check=True)

    untracked = subprocess.check_output(
        ["git", "ls-files", "--others", "--exclude-standard", "-z", "--", "api"],
        cwd=REPO_ROOT,
    ).split(b"\0")
    for raw_path in filter(None, untracked):
        relative = Path(os.fsdecode(raw_path))
        destination = checkout / relative
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(REPO_ROOT / relative, destination)


class V810ProvenanceTest(unittest.TestCase):
    """Contract checks for reproducible v8.10 generation evidence."""

    def assert_v810_identity_invariant(self, provenance: dict[str, Any]) -> None:
        """Baseline provenance must keep the single v810 target identity."""
        tag = provenance["tag"]

        self.assertRegex(tag, r"^8\.10([.-].*)?$")
        self.assertEqual(canonical_command_for_tag(tag), provenance["command"])
        self.assertIn("--target v810", provenance["command"])
        self.assertNotIn("v810alpha", provenance["command"])
        self.assertNotIn("v810rc", provenance["command"])
        self.assertEqual(EXPECTED_OUTPUT_PARTS, GENERATED_CLIENT_PATH.parent.parts[-5:])
        self.assertEqual(EXPECTED_OUTPUT_PARTS, PROVENANCE_PATH.parent.parts[-5:])

    def test_schema_and_canonical_source_identity(self) -> None:
        """The provenance file must identify exactly one pinned v8.10 baseline."""
        provenance = load_provenance()

        self.assertEqual(EXPECTED_KEYS, set(provenance))
        self.assertEqual(1, provenance["schemaVersion"])
        self.assertEqual("https://github.com/camunda/camunda.git", provenance["repository"])
        self.assertEqual(EXPECTED_TAG, provenance["tag"])
        self.assertEqual(EXPECTED_COMMIT, provenance["commit"])
        self.assertEqual(
            "zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml",
            provenance["sourceSpec"],
        )
        self.assertEqual(canonical_command_for_tag(EXPECTED_TAG), provenance["command"])
        self.assert_v810_identity_invariant(provenance)

    def test_future_baseline_commands_keep_v810_identity(self) -> None:
        """Later patch tags still update the same v810 artifacts."""
        provenance = load_provenance()

        for tag in ("8.10.4", "8.10.5"):
            with self.subTest(tag=tag):
                updated = dict(provenance)
                updated["tag"] = tag
                updated["command"] = canonical_command_for_tag(tag)
                self.assert_v810_identity_invariant(updated)

    def test_transformations_are_ordered_and_hashed(self) -> None:
        """The mutation chain is recorded in the same order the generator applies it."""
        provenance = load_provenance()
        transformations = provenance["transformations"]

        self.assertEqual(list(EXPECTED_TRANSFORMATIONS), [item["path"] for item in transformations])
        for item in transformations:
            self.assertEqual({"path", "sha256"}, set(item))
            self.assertRegex(item["sha256"], SHA256_RE)
            self.assertEqual(EXPECTED_TRANSFORMATIONS[item["path"]], item["sha256"])
            self.assertEqual(sha256_file(REPO_ROOT / item["path"]), item["sha256"])

        self.assertEqual(EXPECTED_TOOLS, provenance["tools"])

    def test_recorded_hashes_match_current_artifacts(self) -> None:
        """Content hashes must agree with the checked-in generated client."""
        provenance = load_provenance()

        self.assertRegex(provenance["sourceSpecSha256"], SHA256_RE)
        self.assertRegex(provenance["preparedSpecSha256"], SHA256_RE)
        self.assertRegex(provenance["generatedClientSha256"], SHA256_RE)
        self.assertEqual(EXPECTED_SOURCE_SPEC_SHA256, provenance["sourceSpecSha256"])
        self.assertEqual(EXPECTED_PREPARED_SPEC_SHA256, provenance["preparedSpecSha256"])
        self.assertEqual(EXPECTED_GENERATED_CLIENT_SHA256, provenance["generatedClientSha256"])
        self.assertEqual(
            sha256_file(GENERATED_CLIENT_PATH),
            provenance["generatedClientSha256"],
        )
        self.assertRegex(
            GENERATED_CLIENT_PATH.read_text(encoding="utf-8", errors="replace"),
            r"(?m)^package camunda$",
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

    @unittest.skipUnless(
        os.environ.get("C8VOLT_V810_NETWORK_REPRODUCIBILITY") == "1",
        "set C8VOLT_V810_NETWORK_REPRODUCIBILITY=1 for isolated network regeneration",
    )
    def test_networked_regeneration_is_diff_free_in_disposable_checkout(self) -> None:
        """Pinned regeneration must be deterministic without touching the developer checkout."""
        with tempfile.TemporaryDirectory(prefix="c8volt-v810-repro-") as temp_dir:
            checkout = Path(temp_dir) / "checkout"
            subprocess.run(
                ["git", "worktree", "add", "--quiet", "--detach", str(checkout), "HEAD"],
                cwd=REPO_ROOT,
                check=True,
            )
            try:
                apply_working_api_changes(checkout)
                protected = checkout / "internal/clients/camunda"
                before = {
                    version: fingerprint_tree(protected / version)
                    for version in ("v88", "v89")
                }
                command = [
                    "bash",
                    "api/refresh-clients.sh",
                    "--target",
                    "v810",
                    "--camunda-tag",
                    EXPECTED_TAG,
                ]
                for _ in range(2):
                    result = subprocess.run(
                        command,
                        cwd=checkout,
                        text=True,
                        stdout=subprocess.PIPE,
                        stderr=subprocess.STDOUT,
                        check=False,
                    )
                    self.assertEqual(0, result.returncode, result.stdout)

                generated_dir = protected / "v810/camunda"
                self.assertEqual(
                    sha256_file(GENERATED_CLIENT_PATH),
                    sha256_file(generated_dir / "client.gen.go"),
                )
                self.assertEqual(
                    PROVENANCE_PATH.read_bytes(),
                    (generated_dir / "provenance.json").read_bytes(),
                )
                self.assertEqual(
                    before,
                    {
                        version: fingerprint_tree(protected / version)
                        for version in ("v88", "v89")
                    },
                )
            finally:
                subprocess.run(
                    ["git", "worktree", "remove", "--force", str(checkout)],
                    cwd=REPO_ROOT,
                    check=False,
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL,
                )


if __name__ == "__main__":
    unittest.main()
