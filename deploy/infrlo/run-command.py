#!/usr/bin/env python3
"""Infrlo Run Command: authenticated Python ingress + one cold-start Linux gateway.

This mode intentionally runs only OpenCode for a disposable 512 MiB proof of
concept. Other gateways can be enabled later on a persistent, supported host.
"""
import base64
import hmac
from http.client import HTTPConnection
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import secrets
import shutil
import signal
import subprocess
import sys
import threading
import time

REPO = Path(__file__).resolve().parents[2]
STAGED = REPO / ".infrlo-native" / "AhB"
PORT = int(os.environ.get("PORT", "8080"))
SECRET = os.environ.get("AHB_PUBLIC_TOKEN", "")
USERNAME = os.environ.get("AHB_PUBLIC_USER", "ahb")
MAX_REQUEST = 8 * 1024 * 1024


def secret_matches(candidate):
    return isinstance(candidate, str) and bool(SECRET) and hmac.compare_digest(
        candidate.encode("utf-8"), SECRET.encode("utf-8")
    )


def setup():
    if not STAGED.joinpath("bin", "hubd").is_file():
        raise RuntimeError("Linux AhB binary not installed; check Infrlo Build Command logs")
    state = os.environ.get("AHB_STATE_DIR", "").strip()
    root = Path(state).expanduser().resolve() / "AhB" if state else STAGED
    if root != STAGED and not root.joinpath("bin", "hubd").exists():
        root.parent.mkdir(parents=True, exist_ok=True)
        shutil.copytree(STAGED, root, dirs_exist_ok=True)
    root.joinpath("data", "opencode").mkdir(parents=True, exist_ok=True)
    root.joinpath("logs").mkdir(exist_ok=True)

    def load_or_create_secret(path):
        if not path.is_file() or not path.read_text().strip():
            path.write_text(secrets.token_hex(24) + "\n")
            path.chmod(0o600)
        return path.read_text().strip()

    hub_key = load_or_create_secret(root / "data" / "hub-local-key.txt")
    web_key = load_or_create_secret(root / "data" / "opencode" / "webui-password.txt")

    cfgpath = root / "config.json"
    if cfgpath.is_file():
        config = json.loads(cfgpath.read_text())
    else:
        config = json.loads((root / "config.example.json").read_text())
    config["listen"] = "127.0.0.1:8317"
    config["allow_lan"] = False
    config["resources"] = {"max_running_sidecars": 1, "idle_stop_seconds": 120}
    for provider in config["providers"]:
        provider["enabled"] = provider["id"] == "opencode"
        if provider["id"] == "opencode":
            provider["start_mode"] = "on_demand"
            provider.setdefault("headers", {})["Authorization"] = "Bearer " + hub_key
    cfgpath.write_text(json.dumps(config, ensure_ascii=False, indent=2))
    cfgpath.chmod(0o600)

    opencfg = root / "data" / "opencode" / "config.json"
    if opencfg.is_file():
        raw = opencfg.read_text()
    else:
        raw = (root / "configs" / "opencode2api.json").read_text()
    raw = raw.replace("__AIHUB_SERVER_KEY__", hub_key)
    raw = raw.replace("hub-local-opencode", hub_key)
    raw = raw.replace("__AIHUB_WEB_PASSWORD__", web_key)
    opencfg.write_text(raw)
    opencfg.chmod(0o600)
    return root


def authorized(headers):
    authorization = headers.get("Authorization", "")
    if authorization.startswith("Bearer "):
        return secret_matches(authorization[7:])
    if authorization.startswith("Basic "):
        try:
            raw = base64.b64decode(authorization[6:], validate=True).decode("utf-8")
            user, token = raw.split(":", 1)
            return hmac.compare_digest(user, USERNAME) and secret_matches(token)
        except (ValueError, UnicodeError):
            return False
    return secret_matches(headers.get("x-api-key", ""))


class Gateway(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.0"
    server_version = "AhB-SourceHost"
    sys_version = ""

    def log_message(self, fmt, *args):
        # Never print credentials, URLs, query strings or request bodies.
        pass

    def do_GET(self):
        self.handle_api()

    def do_HEAD(self):
        self.handle_api()

    def do_POST(self):
        self.handle_api()

    def do_PUT(self):
        self.handle_api()

    def do_DELETE(self):
        self.handle_api()

    def reply(self, status, payload, headers=None):
        data = json.dumps(payload, separators=(",", ":")).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Cache-Control", "no-store")
        if headers:
            for name, value in headers.items():
                self.send_header(name, value)
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        if self.command != "HEAD":
            self.wfile.write(data)

    def handle_api(self):
        route = self.path.split("?", 1)[0]
        if route == "/healthz" and self.command in ("GET", "HEAD"):
            if len(SECRET) < 32:
                self.reply(200, {"status": "setup_required"})
                return
            try:
                c = HTTPConnection("127.0.0.1", 8317, timeout=2)
                c.request("GET", "/healthz")
                r = c.getresponse()
                code = r.status
                r.read()
                c.close()
                self.reply(200 if code == 200 else 503, {"status": "ok" if code == 200 else "starting"})
            except OSError:
                self.reply(503, {"status": "starting"})
            return

        # If Infrlo has no secret-management screen, boot in locked mode.
        # NEVER leak free upstream account endpoints as an anonymous API.
        if len(SECRET) < 32 or "\n" in SECRET or "\r" in SECRET:
            self.reply(503, {"error": "Private environment variable AHB_PUBLIC_TOKEN (32+ chars) is required"})
            return
        if not authorized(self.headers):
            self.reply(401, {"error": "Authentication required"}, {"WWW-Authenticate": 'Basic realm="AhB"'})
            return
        if route == "/api/control" or route.startswith("/api/control/"):
            self.reply(403, {"error": "Private localhost controls are unavailable through cloud ingress"})
            return
        if not self.path.startswith("/") or self.path.startswith("//"):
            self.reply(400, {"error": "Invalid request target"})
            return
        length = self.headers.get("Content-Length")
        if self.headers.get("Transfer-Encoding") or (self.command in ("POST", "PUT") and length is None):
            self.reply(411, {"error": "Content-Length required"})
            return
        try:
            size = int(length or "0")
            if size < 0 or size > MAX_REQUEST:
                self.reply(413, {"error": "Request too large"})
                return
        except ValueError:
            self.reply(400, {"error": "Invalid Content-Length"})
            return
        body = self.rfile.read(size) if size else None
        deny = {"host", "authorization", "x-api-key", "cookie", "proxy-authorization",
                "connection", "transfer-encoding", "content-length", "forwarded", "te", "upgrade"}
        outbound = {k: v for k, v in self.headers.items() if k.lower() not in deny
                    and not k.lower().startswith("x-forwarded-")}
        outbound["Host"] = "127.0.0.1:8317"
        outbound["Connection"] = "close"
        connection = HTTPConnection("127.0.0.1", 8317, timeout=600)
        try:
            connection.request(self.command, self.path, body=body, headers=outbound)
            response = connection.getresponse()
            self.send_response(response.status)
            response_deny = {"transfer-encoding", "connection", "set-cookie", "keep-alive", "proxy-authenticate", "trailer"}
            for key, value in response.getheaders():
                if key.lower() not in response_deny:
                    self.send_header(key, value)
            self.send_header("Connection", "close")
            self.end_headers()
            if self.command == "HEAD":
                return
            # read1 yields currently available bytes for SSE, not entire stream.
            while True:
                fragment = response.read1(16 * 1024)
                if not fragment:
                    break
                self.wfile.write(fragment)
                self.wfile.flush()
        except (OSError, ConnectionError) as exc:
            if not self.wfile.closed:
                try:
                    self.reply(502, {"error": "Internal AhB service unavailable"})
                except OSError:
                    pass
        finally:
            connection.close()
            self.close_connection = True


def main():
    if not 0 < PORT < 65536:
        raise RuntimeError("invalid PORT")
    process = None
    if len(SECRET) >= 32 and "\n" not in SECRET and "\r" not in SECRET:
        root = setup()
        process = subprocess.Popen([str(root / "bin" / "hubd"), "-config", str(root / "config.json")],
                                   cwd=str(root), env={k: v for k, v in os.environ.items()
                                                       if k not in ("AHB_PUBLIC_TOKEN", "AHB_PUBLIC_USER")})
    else:
        print("AhB locked: set private AHB_PUBLIC_TOKEN (32+ chars), then restart to enable API.", flush=True)
    server = ThreadingHTTPServer(("0.0.0.0", PORT), Gateway)
    server.daemon_threads = True
    def shutdown(*_):
        server.shutdown()
    signal.signal(signal.SIGINT, shutdown)
    signal.signal(signal.SIGTERM, shutdown)
    print("AhB command-mode gateway started on platform PORT", PORT, flush=True)
    try:
        server.serve_forever(poll_interval=0.2)
    finally:
        server.server_close()
        if process and process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print("AhB Infrlo runner failed:", str(exc), file=sys.stderr)
        sys.exit(1)
