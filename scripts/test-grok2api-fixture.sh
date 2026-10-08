#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
server_pid=""
cleanup(){
  if [ -n "$server_pid" ]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  rm -rf "$TMP"
}
trap cleanup EXIT

mkdir -p "$TMP/AhB/scripts" "$TMP/AhB/bin" "$TMP/AhB/data/grok2api"
cp "$ROOT/scripts/test-grok2api-termux.sh" "$ROOT/scripts/test-tool-roundtrip.sh" "$TMP/AhB/scripts/"
printf '#!/bin/sh\nexit 0\n' >"$TMP/AhB/bin/grok2api"
chmod +x "$TMP/AhB/bin/grok2api"
printf 'private-test-key\n' >"$TMP/AhB/data/grok2api/client-key.txt"
chmod 600 "$TMP/AhB/data/grok2api/client-key.txt"

cat >"$TMP/server.py" <<'PY'
import json
import os
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path

class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_args):
        pass

    def send_json(self, status, payload):
        content = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(content)))
        self.end_headers()
        self.wfile.write(content)

    def do_GET(self):
        if self.path == "/healthz":
            return self.send_json(200, {"ok": True})
        if self.path == "/readyz":
            return self.send_json(200, {"ready": True, "state": "ready", "components": {"grok_build": {"state": "ready"}}})
        if self.path == "/v1/models":
            if self.headers.get("Authorization") != "Bearer private-test-key":
                return self.send_json(401, {"error":"missing client key"})
            if os.getenv("MODE") == "nomodel":
                return self.send_json(200, {"data": []})
            return self.send_json(200, {"data":[{"id":"test-tool-model"}]})
        self.send_error(404)

    def do_POST(self):
        if self.path != "/v1/chat/completions":
            return self.send_json(404, {"error":"path"})
        n=int(self.headers.get("Content-Length","0"))
        obj=json.loads(self.rfile.read(n))
        if obj.get("stream"):
            raw=b'data: {"choices":[{"delta":{"content":"Hello"}}]}\n\ndata: [DONE]\n\n'
            self.send_response(200)
            self.send_header("Content-Type","text/event-stream")
            self.send_header("Content-Length", str(len(raw)))
            self.end_headers()
            self.wfile.write(raw)
            return
        msgs=obj["messages"]
        if msgs[-1]["role"] == "tool":
            if msgs[-1].get("tool_call_id") != "call-123":
                return self.send_json(400, {"error":"tool id mismatch"})
            content="TOOL_LOOP_OK"
            reason="stop"
            tool_calls=None
        elif obj.get("tools"):
            if os.getenv("MODE") == "notool":
                content="No call"
                reason="stop"
                tool_calls=None
            else:
                content=None
                reason="tool_calls"
                tool_calls=[{"id":"call-123","type":"function","function":{"name":"get_time","arguments":"{\"timezone\":\"Asia/Taipei\"}"}}]
        else:
            content="GROK_AHB_OK"
            reason="stop"
            tool_calls=None
        message={"role":"assistant","content":content}
        if tool_calls is not None:
            message["tool_calls"]=tool_calls
        self.send_json(200, {"choices":[{"message":message,"finish_reason":reason}]})

server=HTTPServer(("127.0.0.1",0), Handler)
Path(os.environ["PORT_FILE"]).write_text(str(server.server_port))
server.serve_forever()
PY
run_server(){
  MODE="$1" PORT_FILE="$TMP/port" python3 "$TMP/server.py" &
  server_pid="$!"
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    [ -s "$TMP/port" ] && break
    sleep .2
  done
  test -s "$TMP/port"
}
stop_server(){
  kill "$server_pid"
  wait "$server_pid" 2>/dev/null || true
  server_pid=""
  rm -f "$TMP/port"
}
run_server success
url="http://127.0.0.1:$(cat "$TMP/port")"
( cd "$TMP/AhB"; AIHUB_BASE="$url" AIHUB_GROK_BASE="$url" \
  bash scripts/test-grok2api-termux.sh ) | grep "Metadata-only checks finished."
( cd "$TMP/AhB"; AIHUB_BASE="$url" AIHUB_GROK_BASE="$url" \
  AIHUB_TEST_MODEL="grok/test-tool-model" bash scripts/test-grok2api-termux.sh ) \
  | grep "GROK ACCEPTANCE PASS"
stop_server
run_server notool
url="http://127.0.0.1:$(cat "$TMP/port")"
if ( cd "$TMP/AhB"; AIHUB_BASE="$url" AIHUB_GROK_BASE="$url" \
  AIHUB_TEST_MODEL="grok/test-tool-model" bash scripts/test-grok2api-termux.sh ) \
  >/dev/null 2>&1; then
  echo "FAIL: no-tool response was incorrectly accepted" >&2
  exit 1
fi
stop_server
run_server nomodel
url="http://127.0.0.1:$(cat "$TMP/port")"
if ( cd "$TMP/AhB"; AIHUB_BASE="$url" AIHUB_GROK_BASE="$url" \
  bash scripts/test-grok2api-termux.sh ) >/dev/null 2>&1; then
  echo "FAIL: no accounts/models was incorrectly accepted" >&2
  exit 1
fi
echo "Grok2API diagnostics fixture passed (metadata, chat/SSE/tools, no-tool and empty-model failures)"
