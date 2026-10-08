#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
tmp="$(mktemp -d)"
cleanup(){
  if [ -n "${server_pid:-}" ]; then kill "$server_pid" 2>/dev/null || true; fi
  rm -rf "$tmp"
}
trap cleanup EXIT

cat >"$tmp/server.py" <<'PY'
import json
import os
from http.server import BaseHTTPRequestHandler, HTTPServer

class Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        pass

    def do_POST(self):
        if self.path != "/v1/chat/completions":
            self.send_error(404)
            return
        n = int(self.headers.get("Content-Length", "0"))
        req = json.loads(self.rfile.read(n))
        final = req["messages"][-1]["role"] == "tool"
        if final:
            assert req["messages"][-2]["role"] == "assistant"
            assert req["messages"][-1]["tool_call_id"] == "call-123"
            assert "timestamp" in req["messages"][-1]["content"]
            result = {"choices":[{"message":{"role":"assistant","content":"TOOL_LOOP_OK"},"finish_reason":"stop"}]}
        elif os.getenv("MODE") == "notool":
            result = {"choices":[{"message":{"role":"assistant","content":"I did not call a tool."},"finish_reason":"stop"}]}
        else:
            result = {"choices":[{"message":{"role":"assistant","content":None,"tool_calls":[{"id":"call-123","type":"function","function":{"name":"get_time","arguments":"{\"timezone\":\"Asia/Taipei\"}"}}]},"finish_reason":"tool_calls"}]}
        payload=json.dumps(result).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

sock=HTTPServer(("127.0.0.1",0),Handler)
with open(os.environ["PORT_FILE"],"w") as f:
    f.write(str(sock.server_port))
sock.serve_forever()
PY
launch(){
  MODE="$1" PORT_FILE="$tmp/port" python3 "$tmp/server.py" &
  server_pid=$!
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    [ -s "$tmp/port" ] && return
    sleep 0.2
  done
  echo "fixture server did not start" >&2
  exit 1
}
launch success
AIHUB_BASE="http://127.0.0.1:$(cat "$tmp/port")" \
  AIHUB_TEST_MODEL="mock/test" \
  bash scripts/test-tool-roundtrip.sh | grep "PASS: native function call"
kill "$server_pid"
wait "$server_pid" 2>/dev/null || true
server_pid=""
rm "$tmp/port"
launch notool
if AIHUB_BASE="http://127.0.0.1:$(cat "$tmp/port")" \
   AIHUB_TEST_MODEL="mock/test" bash scripts/test-tool-roundtrip.sh >/dev/null 2>&1; then
  echo "expected no-tool response to FAIL" >&2
  exit 1
fi
echo "tool-call roundtrip fixture passed (success plus no-tool rejection)"
