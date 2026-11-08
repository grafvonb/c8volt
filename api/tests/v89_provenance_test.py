#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later
"""Check the pinned Camunda 8.9 client without contacting a running cluster."""

import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
OUTPUT = ROOT / "internal/clients/camunda/v89/camunda"


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


class V89ProvenanceTest(unittest.TestCase):
    def test_pinned_source_and_artifact_hashes(self):
        provenance = json.loads((OUTPUT / "provenance.json").read_text())
        self.assertEqual(1, provenance["schemaVersion"])
        self.assertEqual("https://github.com/camunda/camunda.git", provenance["repository"])
        self.assertEqual("8.9.28", provenance["tag"])
        self.assertEqual("2bfd882b69363aa4bbdab4ae2925c010b5100476", provenance["commit"])
        self.assertEqual("zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml", provenance["sourceSpec"])
        self.assertEqual("bash api/refresh-clients.sh --target v89 --camunda-tag 8.9.28", provenance["command"])
        self.assertEqual({"redocly": "2.76.0", "oapi-codegen": "2.8.2"}, provenance["tools"])
        self.assertEqual(digest(OUTPUT / "client.gen.go"), provenance["generatedClientSha256"])
        self.assertEqual(digest(ROOT / "api/oapi-codegen.yaml"), provenance["generatorConfigSha256"])
        paths = [entry["path"] for entry in provenance["transformations"]]
        self.assertEqual([
            "api/mutations/mutate-search-query-schemas.py",
            "api/mutations/mutate-search-result-schemas.py",
            "api/mutations/mutate-fix-process-instance-filter-fields.py",
            "api/mutations/mutate-fix-jobresult-discriminator.py",
            "api/mutations/mutate-fix-camunda-v2-operation-id-collisions.py",
        ], paths)
        for entry in provenance["transformations"]:
            self.assertEqual(digest(ROOT / entry["path"]), entry["sha256"])

    def test_rejects_wrong_release_line_before_publication(self):
        names = ("client.gen.go", "provenance.json")
        before = {name: digest(OUTPUT / name) for name in names}
        with tempfile.TemporaryDirectory(prefix="c8volt-v89-tools-") as folder:
            for name in ("go", "git", "redocly"):
                tool = Path(folder) / name
                tool.write_text("#!/bin/sh\nexit 81\n")
                tool.chmod(0o755)
            environment = dict(os.environ, PATH=folder + os.pathsep + os.defpath)
            result = subprocess.run([
                "bash", "api/refresh-clients.sh", "--target", "v89", "--camunda-tag", "8.10.9",
            ], cwd=ROOT, env=environment, capture_output=True, text=True)
        self.assertNotEqual(0, result.returncode)
        self.assertIn("Invalid Camunda tag for v89", result.stderr)
        self.assertEqual(before, {name: digest(OUTPUT / name) for name in names})


if __name__ == "__main__":
    unittest.main()
