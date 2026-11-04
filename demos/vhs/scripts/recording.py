#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

"""Prepare a private recording copy and check its target before VHS starts."""

import argparse
import json
import os
from pathlib import Path
import re
import shlex
import shutil
import subprocess
import sys
import tempfile


class RecordingError(Exception):
    pass


def resolve_config(explicit=None):
    selected = explicit if explicit is not None else os.environ.get("C8VOLT_CONFIG")
    if selected:
        path = Path(selected).expanduser().resolve()
        if not path.is_file():
            raise RecordingError("configuration file not found; check --config or C8VOLT_CONFIG")
        return path

    directories = [Path.cwd()]
    if os.environ.get("XDG_CONFIG_HOME"):
        directories.append(Path(os.environ["XDG_CONFIG_HOME"]) / "c8volt")
    directories.extend([Path.home() / ".config/c8volt", Path.home() / ".c8volt", Path("/etc/c8volt")])
    # Match Viper's config-file search order; parsing remains owned by c8volt.
    extensions = ("json", "toml", "yaml", "yml", "properties", "props", "prop", "hcl", "tfvars", "dotenv", "env", "ini")
    for directory in directories:
        for extension in extensions:
            path = directory / ("config." + extension)
            if path.is_file():
                return path.resolve()
    raise RecordingError("no existing configuration found; set C8VOLT_CONFIG or pass --config")


def is_c810(version):
    return isinstance(version, str) and re.fullmatch(
        r"v?8\.10(?:\.\d+)?(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?", version
    ) is not None


def validate_target(output):
    try:
        result = json.loads(output)
    except (ValueError, UnicodeError):
        raise RecordingError("connection check did not return valid JSON") from None
    if not isinstance(result, dict) or result.get("ok") is not True:
        raise RecordingError("connection check did not confirm the target")
    cluster = result.get("cluster")
    if not isinstance(cluster, dict) or not is_c810(cluster.get("gateway_version")):
        raise RecordingError("recording requires a Camunda 8.10 gateway")
    if "warnings" not in result or result["warnings"] not in (None, []):
        raise RecordingError("configuration and target versions differ; select a profile configured for Camunda 8.10")
    brokers = cluster.get("broker_details")
    if not isinstance(brokers, list) or not brokers or cluster.get("brokers") != len(brokers):
        raise RecordingError("connection check did not confirm running brokers")
    partition_count = 0
    for broker in brokers:
        if not isinstance(broker, dict) or not is_c810(broker.get("version")):
            raise RecordingError("recording requires Camunda 8.10 on every broker")
        partitions = broker.get("partitions")
        if not isinstance(partitions, list):
            raise RecordingError("connection check did not confirm broker partitions")
        partition_count += len(partitions)
        for partition in partitions:
            health = partition.get("health") if isinstance(partition, dict) else None
            if not isinstance(health, str) or health.lower() != "healthy":
                raise RecordingError("target has an unhealthy or unknown partition state")
    if not partition_count:
        raise RecordingError("connection check did not confirm running partitions")
    return cluster["gateway_version"]


def check_target(binary, env):
    for args, description in [
        (["config", "validate"], "configuration validation"),
        (["--no-version-check", "--json", "config", "test-connection"], "connection check"),
    ]:
        result = subprocess.run([str(binary), *args], env=env, stdout=subprocess.PIPE, check=False)
        if result.returncode:
            raise RecordingError(description + " failed; recording was not started")
    return validate_target(result.stdout)


def prepare(binary, tape, config_path=None, profile=None):
    source = resolve_config(config_path)
    if profile is None:
        profile = os.environ.get("C8VOLT_ACTIVE_PROFILE")
    workspace = Path(tempfile.mkdtemp(prefix="c8volt-vhs-", dir="/tmp"))
    try:
        (workspace / "bin").mkdir()
        (workspace / "home/.config").mkdir(parents=True)
        (workspace / "reports").mkdir()
        real_binary = workspace / "bin/c8volt.real"
        shutil.copyfile(binary, real_binary)
        real_binary.chmod(0o700)
        snapshot = workspace / ("config" + source.suffix)
        shutil.copyfile(source, snapshot)
        snapshot.chmod(0o600)

        args = [str(real_binary), "--config", str(snapshot), "--log-format", "plain-time", "--log-level", "info",
                "--automation=false", "--auto-confirm=false", "--no-err-codes=false"]
        if profile is not None:
            args.extend(["--profile", profile])
        wrapper = workspace / "bin/c8volt"
        wrapper.write_text("#!/usr/bin/env bash\nexec " + shlex.join(args) + ' "$@"\n')
        wrapper.chmod(0o700)

        env = os.environ.copy()
        env["HOME"] = str(workspace / "home")
        env["XDG_CONFIG_HOME"] = str(workspace / "home/.config")
        version = check_target(wrapper, env)
        # Tapes keep their readable, stable source paths; only this private copy changes.
        (workspace / "recording.tape").write_text(tape.read_text().replace("/tmp/c8volt-vhs", str(workspace)))
        print("ok: recording target is reachable and healthy, Camunda " + version, file=sys.stderr)
        return workspace
    except BaseException:
        shutil.rmtree(workspace)
        raise


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("binary", type=Path)
    parser.add_argument("tape", type=Path)
    parser.add_argument("--config")
    parser.add_argument("--profile")
    args = parser.parse_args()
    try:
        print(prepare(args.binary, args.tape, args.config, args.profile))
    except (RecordingError, OSError) as error:
        print("error: " + str(error), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
