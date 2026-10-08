#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

is_ahb_exe() {
  case "$1" in
    "$ROOT/bin/hubd"|"$ROOT/bin/opencode2api"|"$ROOT/bin/freebuff2api"|"$ROOT/bin/agent2api-server"|"$ROOT/bin/deepseek2api"|"$ROOT/bin/grok2api"|"$ROOT/bin/kiro-go") return 0 ;;
    *) return 1 ;;
  esac
}

collect_ahb_pids() {
  local proc pid exe
  for proc in /proc/[0-9]*; do
    pid="${proc##*/}"
    [ "$pid" = "$$" ] && continue
    exe="$(readlink "$proc/exe" 2>/dev/null || true)"
    if [ -n "$exe" ] && is_ahb_exe "$exe"; then
      printf '%s\n' "$pid"
    fi
  done
}

pids="$(collect_ahb_pids | sort -u | tr '\n' ' ')"
if [ -n "${pids// /}" ]; then
  echo "stopping AhB processes: $pids"
  kill $pids 2>/dev/null || true
  for _ in 1 2 3 4 5; do
    sleep 1
    remaining=""
    for pid in $pids; do
      if kill -0 "$pid" 2>/dev/null; then
        remaining="$remaining $pid"
      fi
    done
    [ -z "${remaining// /}" ] && break
  done
  for pid in $pids; do
    if kill -0 "$pid" 2>/dev/null; then
      kill -9 "$pid" 2>/dev/null || true
    fi
  done
else
  echo "no AhB processes running"
fi

rm -f data/hubd.pid

if command -v termux-wake-unlock >/dev/null 2>&1; then
  termux-wake-unlock || true
fi

echo "AhB stopped"
