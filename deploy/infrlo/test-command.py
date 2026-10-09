#!/usr/bin/env python3
"""Offline regression tests for the command-runner's privacy and SSE semantics."""
import importlib.util
import json
from http.client import HTTPConnection
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
import threading
import time
import unittest

script = Path(__file__).with_name("run-command.py")
spec = importlib.util.spec_from_file_location("ahb_infrlo_source_runner", script)
runner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(runner)

TOKEN = "ci-only-secret-1234567890abcdef-1234567890abcdef"


class FakeHub(BaseHTTPRequestHandler):
    def log_message(self, *_):
        pass

    def do_GET(self):
        if self.path == "/healthz":
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b"ok")
            return
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps({
            "path": self.path,
            "authorization": self.headers.get("Authorization"),
            "x-api-key": self.headers.get("x-api-key"),
            "x-forwarded-for": self.headers.get("X-Forwarded-For"),
        }).encode())

    def do_POST(self):
        if self.path == "/v1/chat/completions":
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream")
            self.end_headers()
            self.wfile.write(b'data: {"choices":[{"delta":{"content":"ok"}}]}\n\n')
            self.wfile.flush()
            time.sleep(0.02)
            self.wfile.write(b"data: [DONE]\n\n")
            self.wfile.flush()
            return
        self.do_GET()


class CommandRunnerTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.upstream = ThreadingHTTPServer(("127.0.0.1", 8317), FakeHub)
        cls.upstream.daemon_threads = True
        cls.gateway = ThreadingHTTPServer(("127.0.0.1", 0), runner.Gateway)
        cls.gateway.daemon_threads = True
        cls.port = cls.gateway.server_address[1]
        for server in (cls.upstream, cls.gateway):
            threading.Thread(target=server.serve_forever, daemon=True).start()

    @classmethod
    def tearDownClass(cls):
        for server in (cls.gateway, cls.upstream):
            server.shutdown()
            server.server_close()

    def request(self, method, path, headers=None, body=None):
        con = HTTPConnection("127.0.0.1", self.port, timeout=5)
        con.request(method, path, body=body, headers=headers or {})
        res = con.getresponse()
        status, raw = res.status, res.read()
        con.close()
        return status, raw

    def test_00_default_or_injected_port(self):
        import os
        self.assertEqual(runner.PORT, int(os.environ.get("PORT") or "5000"))

    def test_01_locked_mode(self):
        runner.SECRET = ""
        self.assertEqual(json.loads(self.request("GET", "/healthz")[1])["status"], "setup_required")
        self.assertEqual(self.request("GET", "/v1/models")[0], 503)
        self.assertEqual(self.request("GET", "/ui", {"Authorization": "Bearer " + TOKEN})[0], 503)

    def test_02_auth_and_forwarding(self):
        runner.SECRET = TOKEN
        status, raw = self.request("GET", "/healthz")
        self.assertEqual((status, json.loads(raw)["status"]), (200, "ok"))
        self.assertEqual(self.request("GET", "/v1/models")[0], 401)
        self.assertEqual(self.request("GET", "/v1/models", {"Authorization": "Bearer wrong"})[0], 401)
        status, raw = self.request("GET", "/v1/models", {
            "Authorization": "Bearer " + TOKEN,
            "X-Forwarded-For": "untrusted-ip",
        })
        self.assertEqual(status, 200)
        self.assertEqual(json.loads(raw), {
            "path": "/v1/models", "authorization": None, "x-api-key": None, "x-forwarded-for": None,
        })
        from base64 import b64encode
        self.assertEqual(self.request("GET", "/ui", {
            "Authorization": "Basic " + b64encode(("ahb:" + TOKEN).encode()).decode()
        })[0], 200)
        self.assertEqual(self.request("GET", "/v1/messages", {"x-api-key": TOKEN})[0], 200)
        self.assertEqual(self.request("POST", "/api/control/wake/opencode", {
            "Authorization": "Bearer " + TOKEN,
        }, b"{}")[0], 403)

    def test_03_sse_complete(self):
        runner.SECRET = TOKEN
        status, raw = self.request("POST", "/v1/chat/completions", {
            "Authorization": "Bearer " + TOKEN, "Content-Type": "application/json"
        }, b"{}")
        self.assertEqual(status, 200)
        self.assertIn(b'"content":"ok"', raw)
        self.assertIn(b"data: [DONE]\n\n", raw)


if __name__ == "__main__":
    unittest.main(verbosity=2)
