#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later
"""Preserve c8volt's existing resource-deletion client name for Camunda 8.8."""

from pathlib import Path
import sys

import yaml


source = Path(sys.argv[1])
document = yaml.safe_load(source.read_text(encoding="utf-8"))
operation = document["paths"]["/resources/{resourceKey}/deletion"]["post"]
if operation.get("operationId") != "deleteResource":
    raise SystemExit("expected the Camunda 8.8 deleteResource operation")
for methods in document["paths"].values():
    for candidate in methods.values():
        if isinstance(candidate, dict) and candidate.get("operationId") == "deleteResourceOp":
            raise SystemExit("deleteResourceOp is already defined")
operation["operationId"] = "deleteResourceOp"
output = source.with_name(source.stem + "-v88-opids-fixed.yaml")
output.write_text(yaml.safe_dump(document, sort_keys=False, allow_unicode=True), encoding="utf-8")
