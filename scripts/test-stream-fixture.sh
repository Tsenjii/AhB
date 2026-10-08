#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"
pid=""
cleanup() {
  if [ -n "$pid" ]; then kill "$pid" 2>/dev/null || true; wait "$pid" 2>/dev/null || true; fi
  rm -rf "$tmp"
}
trap cleanup EXIT
cat > "$tmp/mock.py" <<'PY'
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path
import json, os
class Handler(BaseHTTPRequestHandler):
    def log_message(self,*args):pass
    def do_POST(self):
        if self.path!='/v1/chat/completions': return self.send_error(404)
        size=int(self.headers.get('Content-Length','0'))
        body=json.loads(self.rfile.read(size))
        assert body['stream'] is True
        if os.getenv("MODE")=="error":
            status=503
            raw=b'{"error":{"code":"quota"}}'
        else:
            status=200
            lines=[]
            if os.getenv("MODE")!="empty":
                lines.append('data: {"choices":[{"delta":{"content":"A star shines."}}]}')
            if os.getenv("MODE")!="no_done":
                lines.append('data: [DONE]')
            raw=('\n\n'.join(lines)+'\n\n').encode()
        self.send_response(status)
        self.send_header('Content-Type','text/event-stream' if status==200 else 'application/json')
        self.send_header('Content-Length',str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)
server=HTTPServer(('127.0.0.1',0),Handler)
Path(os.environ['PORT_FILE']).write_text(str(server.server_port))
server.serve_forever()
PY
start(){
  MODE="$1" PORT_FILE="$tmp/port" python3 "$tmp/mock.py" &
  pid="$!"
  for _ in {1..10}; do [ -s "$tmp/port" ] && break; sleep .2; done
  test -s "$tmp/port"
}
stop(){
  kill "$pid"
  wait "$pid" 2>/dev/null || true
  pid=""
  rm "$tmp/port"
}
start good
AIHUB_BASE="http://127.0.0.1:$(cat "$tmp/port")" AIHUB_TEST_MODEL="opencode/mock" \
  bash "$ROOT/scripts/test-stream.sh" | grep -F 'PASS: opencode/mock Chat Completions SSE'
stop
for mode in empty no_done error; do
  start "$mode"
  if AIHUB_BASE="http://127.0.0.1:$(cat "$tmp/port")" AIHUB_TEST_MODEL="opencode/mock" \
    bash "$ROOT/scripts/test-stream.sh" > "$tmp/out" 2>&1; then
    echo "FAIL: stream diagnostic falsely accepted $mode" >&2
    exit 1
  fi
  stop
done
echo "SSE fixture passed (real assistant content, complete [DONE], no-empty, no-error)"
