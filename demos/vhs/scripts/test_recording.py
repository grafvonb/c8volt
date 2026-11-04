#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

"""Isolated recording preflight tests. Never run VHS or contact a real cluster."""

from contextlib import contextmanager
from copy import deepcopy
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import threading
import unittest
from unittest.mock import patch
from urllib.parse import parse_qs, urlparse

import recording


ROOT = Path(__file__).resolve().parents[3]
PROFILE = "recording ' c810"
TOPOLOGY = {
    "gatewayVersion": "8.10.6", "clusterSize": 1,
    "partitionsCount": 1, "replicationFactor": 1,
    "brokers": [{"nodeId": 0, "host": "test-owned-broker", "port": 26501,
                 "version": "8.10.6",
                 "partitions": [{"partitionId": 1, "role": "leader", "health": "healthy"}]}],
}


@contextmanager
def fake_target(mode="none", topology=None, reject=False):
    requests = []

    class Handler(BaseHTTPRequestHandler):
        def log_message(self, *args):
            pass

        def reply(self, status, body, cookie=False):
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            if cookie:
                self.send_header("Set-Cookie", "sessionid=test-session; Path=/; HttpOnly")
            self.end_headers()
            self.wfile.write(json.dumps(body).encode())

        def do_POST(self):
            parsed = urlparse(self.path)
            body = self.rfile.read(int(self.headers.get("Content-Length", 0)))
            requests.append(("POST", parsed.path))
            if mode == "oauth2" and parsed.path == "/token":
                form = parse_qs(body.decode())
                valid = form.get("client_secret") == ["private-secret"]
                self.reply(200 if valid and not reject else 401,
                           {"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600})
            elif mode == "cookie" and parsed.path == "/api/login":
                valid = parse_qs(parsed.query).get("password") == ["private-secret"]
                self.reply(200 if valid and not reject else 401, {}, cookie=valid and not reject)
            else:
                self.reply(404, {})

        def do_GET(self):
            requests.append(("GET", self.path))
            valid = (mode == "none" or
                     mode == "oauth2" and self.headers.get("Authorization") == "Bearer test-token" or
                     mode == "cookie" and self.headers.get("Cookie") == "sessionid=test-session")
            if self.path != "/v2/topology":
                self.reply(404, {})
            elif reject or not valid:
                self.reply(401, {})
            else:
                self.reply(200, topology if topology is not None else TOPOLOGY)

    server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        yield "http://127.0.0.1:" + str(server.server_port), requests
    finally:
        server.shutdown()
        server.server_close()
        thread.join()


class RecordingTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.build = tempfile.TemporaryDirectory(prefix="c8volt-vhs-tests-")
        cls.binary = Path(cls.build.name) / "c8volt"
        env = os.environ.copy()
        env.update(GOFLAGS="", GOPROXY="off", GOSUMDB="off")
        result = subprocess.run(["go", "build", "-o", str(cls.binary), "."], cwd=ROOT,
                                env=env, capture_output=True, text=True)
        if result.returncode:
            cls.build.cleanup()
            raise RuntimeError(result.stderr)

    @classmethod
    def tearDownClass(cls):
        cls.build.cleanup()

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="c8volt-vhs-fixture-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        self.home = self.root / "home"
        self.home.mkdir()
        # Do not inherit user profiles, credentials, proxies, or Go flags.
        self.env = {"PATH": os.environ["PATH"], "HOME": str(self.home),
                    "C8VOLT_APP_AUTOMATION": "true", "C8VOLT_APP_NO_ERR_CODES": "true"}
        self.tape = self.root / "test.tape"
        self.tape.write_text('Env HOME "/tmp/c8volt-vhs/home"\nType "cd /tmp/c8volt-vhs/bin"\n')

    def config(self, url, mode="none"):
        auth = {"mode": mode}
        if mode == "oauth2":
            # The existing OAuth client appends /token to its configured server URL.
            auth[mode] = {"token_url": url, "client_id": "test-client", "client_secret": "private-secret"}
        elif mode == "cookie":
            auth[mode] = {"base_url": url, "username": "test-user", "password": "private-secret"}
        path = self.root / "config with ' spaces.json"
        path.write_text(json.dumps({
            "active_profile": PROFILE,
            "app": {"camunda_version": "8.9"},
            "profiles": {PROFILE: {"app": {"camunda_version": "8.10"}, "auth": auth,
                                   "apis": {"camunda_api": {"base_url": url}}}},
        }))
        return path

    def prepare(self, config, *args, env=None):
        before = set(Path("/tmp").glob("c8volt-vhs-*"))
        result = subprocess.run([sys.executable, str(ROOT / "demos/vhs/scripts/recording.py"),
                                 str(self.binary), str(self.tape), "--config", str(config), *args],
                                env=env or self.env, cwd=self.root, capture_output=True, text=True)
        if result.returncode == 0:
            workspace = Path(result.stdout.strip())
            self.addCleanup(shutil.rmtree, workspace, True)
        else:
            self.assertEqual(result.stdout, "")
            self.assertEqual(set(Path("/tmp").glob("c8volt-vhs-*")), before)
        self.assertNotIn("private-secret", result.stdout + result.stderr)
        return result

    def test_all_auth_modes_use_private_config_and_selected_profile(self):
        for mode in ("none", "oauth2", "cookie"):
            with self.subTest(mode=mode), fake_target(mode) as (url, requests):
                source = self.config(url, mode)
                env = self.env.copy()
                args = []
                if mode == "oauth2":
                    config = json.loads(source.read_text())
                    config["active_profile"] = "missing"
                    source.write_text(json.dumps(config))
                    env["C8VOLT_ACTIVE_PROFILE"] = PROFILE
                elif mode == "cookie":
                    env["C8VOLT_ACTIVE_PROFILE"] = "missing"
                    args = ["--profile", PROFILE]
                original = source.read_bytes()
                result = self.prepare(source, *args, env=env)
                self.assertEqual(result.returncode, 0, result.stderr)
                workspace = Path(result.stdout.strip())
                self.assertEqual((workspace / "config.json").read_bytes(), original)
                self.assertEqual(source.read_bytes(), original)
                self.assertEqual(workspace.stat().st_mode & 0o777, 0o700)
                self.assertEqual((workspace / "config.json").stat().st_mode & 0o777, 0o600)
                self.assertNotIn("private-secret", (workspace / "bin/c8volt").read_text())
                self.assertNotIn("/tmp/c8volt-vhs/", (workspace / "recording.tape").read_text())
                self.assertIn(str(workspace / "home"), (workspace / "recording.tape").read_text())
                self.assertEqual(requests.count(("GET", "/v2/topology")), 1)
                # The recording wrapper uses the same config even when the original changes.
                source.write_text("corrupted after preflight")
                recorded = subprocess.run([str(workspace / "bin/c8volt"), "--no-version-check", "--json",
                                           "config", "test-connection"], env=env,
                                          capture_output=True, text=True)
                self.assertEqual(recorded.returncode, 0, recorded.stderr)
                self.assertEqual(json.loads(recorded.stdout)["profile"], PROFILE)
                self.assertEqual(requests.count(("GET", "/v2/topology")), 2)
                if mode != "none":
                    self.assertEqual(sum(method == "POST" for method, _ in requests), 2)

    def test_failure_gates_and_cleanup(self):
        cases = ["corrupt", "missing-profile", "mismatched-config", "wrong-gateway", "wrong-broker", "unhealthy", "denied", "unreachable", "bad-auth"]
        for case in cases:
            with self.subTest(case=case):
                topology = deepcopy(TOPOLOGY)
                if case == "wrong-gateway":
                    topology["gatewayVersion"] = "8.9.20"
                if case == "wrong-broker":
                    topology["brokers"][0]["version"] = "8.9.20"
                if case == "unhealthy":
                    topology["brokers"][0]["partitions"][0]["health"] = "unhealthy"
                mode = "oauth2" if case == "bad-auth" else "none"
                with fake_target(mode, topology, reject=case in ("denied", "bad-auth")) as (url, requests):
                    # TCP port zero cannot be a listening service; no external target is involved.
                    source = self.config("http://127.0.0.1:0" if case == "unreachable" else url, mode)
                    env = self.env.copy()
                    args = []
                    if case == "corrupt":
                        source.write_text("{")
                    elif case == "missing-profile":
                        args = ["--profile", "missing"]
                    elif case == "mismatched-config":
                        env["C8VOLT_APP_CAMUNDA_VERSION"] = "8.9"
                    result = self.prepare(source, *args, env=env)
                    self.assertNotEqual(result.returncode, 0)
                    if case in ("corrupt", "missing-profile"):
                        self.assertEqual(requests, [])

    def test_config_discovery_and_selection(self):
        with patch.dict(os.environ, self.env, clear=True), patch("pathlib.Path.cwd", return_value=self.root):
            xdg = self.root / "xdg"
            os.environ["XDG_CONFIG_HOME"] = str(xdg)
            home_config = self.home / ".config/c8volt/config.yaml"
            home_config.parent.mkdir(parents=True)
            home_config.touch()
            self.assertEqual(recording.resolve_config(), home_config)
            xdg_config = xdg / "c8volt/config.json"
            xdg_config.parent.mkdir(parents=True)
            xdg_config.touch()
            self.assertEqual(recording.resolve_config(), xdg_config)
            local = self.root / "config.yaml"
            local.touch()
            self.assertEqual(recording.resolve_config(), local)
            os.environ["C8VOLT_CONFIG"] = str(home_config)
            self.assertEqual(recording.resolve_config(), home_config)
            self.assertEqual(recording.resolve_config(str(xdg_config)), xdg_config)
            with self.assertRaises(recording.RecordingError):
                recording.resolve_config(str(self.root / "missing"))

    def test_renderer_gates_vhs_and_cleans_up_on_success_or_failure(self):
        # A temporary repo and fake tools exercise orchestration without recording.
        scripts = self.root / "demos/vhs/scripts"
        scripts.mkdir(parents=True)
        for name in ("render.sh", "prepare-recording-env.sh", "check-vhs.sh", "recording.py"):
            shutil.copy2(ROOT / "demos/vhs/scripts" / name, scripts / name)
        shutil.copyfile(ROOT / "demos/vhs/fast-start.tape", scripts.parent / "fast-start.tape")
        tools = self.root / "bin"
        tools.mkdir()
        shutil.copy2(self.binary, tools / "c8volt")
        for name in ("make", "go", "ttyd", "ffmpeg"):
            tool = tools / name
            tool.write_text("#!/bin/sh\nexit 0\n")
            tool.chmod(0o700)
        marker = self.root / "fake-vhs-invocation"
        fake_vhs = tools / "vhs"
        fake_vhs.write_text('#!/bin/sh\nprintf "%s" "$1" > "$TEST_VHS_MARKER"\nexit "$TEST_VHS_EXIT"\n')
        fake_vhs.chmod(0o700)
        env = dict(self.env, PATH=str(tools) + os.pathsep + self.env["PATH"],
                   TEST_VHS_MARKER=str(marker), TEST_VHS_EXIT="0")
        source = self.root / "invalid.json"
        source.write_text("{")
        result = subprocess.run([str(scripts / "render.sh"), "fast-start", "--config", str(source)],
                                env=env, cwd=self.root, capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(marker.exists())
        with fake_target() as (url, _):
            source = self.config(url)
            for exit_code in (0, 7):
                env["TEST_VHS_EXIT"] = str(exit_code)
                result = subprocess.run([str(scripts / "render.sh"), "fast-start", "--config", str(source)],
                                        env=env, cwd=self.root, capture_output=True, text=True)
                self.assertEqual(result.returncode, exit_code, result.stderr)
                self.assertTrue(marker.exists())
                self.assertFalse(Path(marker.read_text()).parent.exists())
                marker.unlink()

    def test_partition_health_values(self):
        result = {"ok": True, "warnings": None, "cluster": {
            "gateway_version": "8.10.6", "brokers": 1,
            "broker_details": [{"version": "8.10.6", "partitions": []}]}}
        broker = result["cluster"]["broker_details"][0]
        for health in ("healthy", "HEALTHY", "Healthy"):
            broker["partitions"] = [{"id": 1, "role": "leader", "health": health}]
            with self.subTest(health=health):
                self.assertEqual(recording.validate_target(json.dumps(result)), "8.10.6")
        for partition in ({"health": "unhealthy"}, {"health": "UNHEALTHY"}, {"health": "unknown"},
                          {"health": "dead"}, {"health": ""}, {"health": None}, {"health": 1},
                          {"health": []}, {}, None):
            # One healthy partition must not hide another unhealthy or unverified partition.
            broker["partitions"] = [{"health": "healthy"}, partition]
            with self.subTest(partition=partition), self.assertRaises(recording.RecordingError):
                recording.validate_target(json.dumps(result))

    def test_fail_closed_on_incomplete_or_invalid_connection_output(self):
        valid = {"ok": True, "warnings": None, "cluster": {
            "gateway_version": "8.10.0-alpha4", "brokers": 1,
            "broker_details": [{"version": "8.10.0-alpha4", "partitions": [{"health": "HEALTHY"}]}]}}
        self.assertEqual(recording.validate_target(json.dumps(valid)), "8.10.0-alpha4")
        for version in ("8.10", "8.10.1", "v8.10.2", "8.10.0-alpha4+build.12"):
            with self.subTest(version=version):
                self.assertTrue(recording.is_c810(version))
        valid["warnings"] = []
        # A broker with no assigned partitions is allowed.
        valid["cluster"]["brokers"] = 2
        valid["cluster"]["broker_details"].append({"version": "8.10.0", "partitions": []})
        self.assertEqual(recording.validate_target(json.dumps(valid)), "8.10.0-alpha4")
        for output in ("not JSON", "{}", "[]", "null", json.dumps(valid) + "{}"):
            with self.subTest(output=output), self.assertRaises(recording.RecordingError):
                recording.validate_target(output)
        for version in ("8.8.0", "8.9.20", "8.11.0", "8.100.0", "", None):
            invalid = deepcopy(valid)
            invalid["cluster"]["gateway_version"] = version
            with self.subTest(version=version), self.assertRaises(recording.RecordingError):
                recording.validate_target(json.dumps(invalid))
        for mutation in ({"warnings": ["version mismatch"]}, {"ok": False},
                         {"cluster": {"gateway_version": "8.10", "brokers": 0, "broker_details": []}}):
            with self.subTest(mutation=mutation), self.assertRaises(recording.RecordingError):
                recording.validate_target(json.dumps(dict(valid, **mutation)))


if __name__ == "__main__":
    unittest.main()
