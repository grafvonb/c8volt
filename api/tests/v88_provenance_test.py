#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later
"""Check the pinned Camunda 8.8 client using local artifacts only."""

import copy
import hashlib
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parents[2]
OUTPUT = ROOT / "internal/clients/camunda/v88/camunda"


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


class V88ProvenanceTest(unittest.TestCase):
    def test_operation_rename_preserves_the_rest_of_the_contract(self):
        before = {"components": {"schemas": {}}, "paths": {
            "/resources/{resourceKey}/deletion": {"post": {"operationId": "deleteResource"}}
        }}
        with tempfile.TemporaryDirectory() as temporary:
            source = Path(temporary) / "spec.yaml"
            source.write_text(yaml.safe_dump(before))
            subprocess.run([sys.executable, str(ROOT / "api/mutations/mutate-v88-operation-ids.py"), str(source)], check=True)
            output = source.with_name("spec-v88-opids-fixed.yaml")
            expected = copy.deepcopy(before)
            expected["paths"]["/resources/{resourceKey}/deletion"]["post"]["operationId"] = "deleteResourceOp"
            self.assertEqual(expected, yaml.safe_load(output.read_text()))
            command = [sys.executable, str(ROOT / "api/validate-v810-mutation-effects.py"),
                       str(source), str(output), "v88-opids-fixed"]
            subprocess.run(command, check=True, capture_output=True)
            expected["paths"]["/resources/{resourceKey}/deletion"]["post"]["summary"] = "unexpected edit"
            output.write_text(yaml.safe_dump(expected))
            rejected = subprocess.run(command, capture_output=True, text=True)
            self.assertNotEqual(0, rejected.returncode)
            self.assertIn("changed more than the client name", rejected.stderr)

    def test_pinned_source_and_artifact_hashes(self):
        provenance = json.loads((OUTPUT / "provenance.json").read_text())
        self.assertEqual(1, provenance["schemaVersion"])
        self.assertEqual("https://github.com/camunda/camunda.git", provenance["repository"])
        self.assertEqual("8.8.46", provenance["tag"])
        self.assertEqual("1cd0dcf5af730abd49ffc859e21d9da35178ee62", provenance["commit"])
        self.assertEqual("zeebe/gateway-protocol/src/main/proto/rest-api.yaml", provenance["sourceSpec"])
        self.assertEqual("bash api/refresh-clients.sh --target v88 --camunda-tag 8.8.46", provenance["command"])
        self.assertEqual({"redocly": "2.76.0", "oapi-codegen": "2.8.2"}, provenance["tools"])
        self.assertEqual(digest(OUTPUT / "client.gen.go"), provenance["generatedClientSha256"])
        self.assertEqual(digest(ROOT / "api/oapi-codegen.yaml"), provenance["generatorConfigSha256"])
        self.assertEqual([
            "api/mutations/mutate-search-query-schemas.py",
            "api/mutations/mutate-search-result-schemas.py",
            "api/mutations/mutate-fix-process-instance-filter-fields.py",
            "api/mutations/mutate-fix-jobresult-discriminator.py",
            "api/mutations/mutate-v88-operation-ids.py",
        ], [entry["path"] for entry in provenance["transformations"]])
        for entry in provenance["transformations"]:
            self.assertEqual(digest(ROOT / entry["path"]), entry["sha256"])


if __name__ == "__main__":
    unittest.main()
