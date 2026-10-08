#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
pid=""
cleanup() {
  if [ -n "$pid" ]; then
    kill "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  fi
  rm -rf "$TMP"
}
trap cleanup EXIT
cat >"$TMP/server.py" <<'PY'
import json
import os
from http.server import HTTPServer, BaseHTTPRequestHandler
from pathlib import Path

class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_args):
        pass
    def do_POST(self):
        if self.path != "/v1/chat/completions":
            self.send_error(404)
            return
        length = int(self.headers.get("Content-Length", "0"))
        data = json.loads(self.rfile.read(length))
        if data["model"] != "opencode/mock":
            raise ValueError("unexpected model")
        if os.environ.get("MODE") == "error":
            status, payload = 503, {"error":{"code":"insufficient_quota"}}
        elif os.environ.get("MODE") == "empty":
            status, payload = 200, {"choices":[{"message":{"role":"assistant","content":""},"finish_reason":"stop"}]}
        elif os.environ.get("MODE") == "tool":
            status, payload = 200, {"choices":[{"message":{"role":"assistant","content":None,"tool_calls":[{"id":"abc"}]},"finish_reason":"tool_calls"}]}
        else:
            assert data["messages"][0]["content"] == 'Reply "AIHUB_OK" with a newline\\nplease'
            status, payload = 200, {"choices":[{"message":{"role":"assistant","content":"AIHUB_OK"},"finish_reason":"stop"}]}
        body = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type","application/json")
        self.send_header("Content-Length",str(len(body)))
        self.end_headers()
        self.wfile.write(body)

server=HTTPServer(("127.0.0.1",0),Handler)
Path(os.environ["PORT_FILE"]).write_text(str(server.server_port))
server.serve_forever()
PY
launch() {
  MODE="$1" PORT_FILE="$TMP/port" python3 "$TMP/server.py" &
  pid="$!"
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    [ -s "$TMP/port" ] && return
    sleep .2
  done
  echo "server did not start" >&2
  exit 1
}
stop() {
  kill "$pid"
  wait "$pid" 2>/dev/null || true
  pid=""
  rm "$TMP/port"
}
launch success
AIHUB_BASE="http://127.0.0.1:$(cat "$TMP/port")" \
 AIHUB_TEST_MODEL='opencode/mock' \
 AIHUB_TEST_PROMPT=$'Reply "AIHUB_OK" with a newline\\nplease' \
 bash "$ROOT/scripts/test-chat.sh" | grep -F 'PASS: HTTP 200 with non-empty assistant answer'
stop
for mode in empty tool error; do
  launch "$mode"
  if AIHUB_BASE="http://127.0.0.1:$(cat "$TMP/port")" \
     AIHUB_TEST_MODEL='opencode/mock' \
     bash "$ROOT/scripts/test-chat.sh" >/dev/null 2>&1; then
    echo "FAIL: chat smoke falsely accepted $mode response" >&2
    exit 1
  fi
  stop
done
echo "chat fixture passed: escaped JSON, genuine response, empty/tool/quota failure"
