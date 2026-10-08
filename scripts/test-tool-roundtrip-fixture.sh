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
        mode = os.getenv("MODE")
        final = req["messages"][-1]["role"] == "tool"
        if final:
            assert req["messages"][1]["role"] == "assistant"
            ids = [x["id"] for x in req["messages"][1]["tool_calls"]]
            tool_outputs = req["messages"][2:]
            assert [x["tool_call_id"] for x in tool_outputs] == ids
            assert all("timestamp" in x["content"] for x in tool_outputs)
            result = {"choices":[{"message":{"role":"assistant","content":"TOOL_LOOP_OK"},"finish_reason":"stop"}]}
        elif mode == "notool":
            result = {"choices":[{"message":{"role":"assistant","content":"I did not call a tool."},"finish_reason":"stop"}]}
        else:
            calls = [{"id":"call-123","type":"function","function":{"name":"get_time","arguments":"{\"timezone\":\"Asia/Taipei\"}"}}]
            if mode == "multitool":
                calls.append({"id":"call-456","type":"function","function":{"name":"get_time","arguments":"{\"timezone\":\"UTC\"}"}})
            if mode == "invalid":
                calls.append({"id":"call-invalid","type":"function","function":{"name":"get_time","arguments":"not-json"}})
            result = {"choices":[{"message":{"role":"assistant","content":None,"tool_calls":calls},"finish_reason":"tool_calls"}]}
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
kill "$server_pid"
wait "$server_pid" 2>/dev/null || true
server_pid=""
rm "$tmp/port"
launch multitool
AIHUB_BASE="http://127.0.0.1:$(cat "$tmp/port")" \
  AIHUB_TEST_MODEL="mock/test" \
  bash scripts/test-tool-roundtrip.sh | grep "PASS: native function call"
kill "$server_pid"
wait "$server_pid" 2>/dev/null || true
server_pid=""
rm "$tmp/port"
launch invalid
if AIHUB_BASE="http://127.0.0.1:$(cat "$tmp/port")" \
   AIHUB_TEST_MODEL="mock/test" bash scripts/test-tool-roundtrip.sh >/dev/null 2>&1; then
  echo "expected malformed tool arguments to FAIL" >&2
  exit 1
fi
echo "tool-call roundtrip fixture passed (one tool, multiple tools, malformed tool, no-tool rejection)"
