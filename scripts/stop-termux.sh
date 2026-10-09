#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

# Stop ONLY processes from this AhB installation. Never stop Termux itself.
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
cd "$ROOT"

is_ahb_exe() {
  case "$1" in
    "$ROOT/bin/hubd"|"$ROOT/bin/opencode2api"|"$ROOT/bin/agent2api-server"|"$ROOT/bin/deepseek2api"|"$ROOT/bin/grok2api"|"$ROOT/bin/kiro-go"|"$ROOT/bin/copilot2api"|"$ROOT/bin/freebuff2api")
      return 0 ;;
    *) return 1 ;;
  esac
}

collect_ahb_pids() {
  local proc pid exe cmdline cwd
  for proc in /proc/[0-9]*; do
    [ -d "$proc" ] || continue
    pid="${proc##*/}"
    [ "$pid" = "$$" ] && continue
    exe="$(readlink "$proc/exe" 2>/dev/null || true)"
    if [ -n "$exe" ] && is_ahb_exe "$exe"; then
      printf '%s\n' "$pid"
      continue
    fi

    # FreeBuff's shell wrapper execs shared Termux Node. Match exact argv.
    case "$exe" in
      */node|*/nodejs)
        cmdline="$(tr '\0' '\n' < "$proc/cmdline" 2>/dev/null || true)"
        case $'\n'"$cmdline"$'\n' in
          *$'\n'"$ROOT/data/freebuff/gateway/server.js"$'\n'*)
            printf '%s\n' "$pid" ;;
        esac
        ;;
      */python|*/python[0-9]*)
        # Optional Kimi Web runs shared Python: require exact cwd + run.py.
        cwd="$(readlink "$proc/cwd" 2>/dev/null || true)"
        if [ "$cwd" = "$ROOT/data/kimiweb/source" ]; then
          cmdline="$(tr '\0' '\n' < "$proc/cmdline" 2>/dev/null || true)"
          case $'\n'"$cmdline"$'\n' in
            *$'\n'run.py$'\n'*) printf '%s\n' "$pid" ;;
          esac
        fi
        ;;
    esac
  done
}

pids="$(collect_ahb_pids | sort -nu | tr '\n' ' ')"
if [ -n "${pids// /}" ]; then
  echo "Stopping AhB-owned processes: $pids"
  kill -TERM $pids 2>/dev/null || true
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    remaining=""
    for pid in $pids; do
      if kill -0 "$pid" 2>/dev/null; then
        remaining="$remaining $pid"
      fi
    done
    [ -z "${remaining// /}" ] && break
    sleep 1
  done
  remaining=""
  for pid in $pids; do
    if kill -0 "$pid" 2>/dev/null; then
      remaining="$remaining $pid"
    fi
  done
  # Never force-kill a process while it may be committing account data.
  if [ -n "$remaining" ]; then
    echo "ERROR: AhB process(es) did not stop cleanly:$remaining" >&2
    exit 1
  fi
else
  echo "No AhB processes running"
fi

rm -f data/hubd.pid
if command -v termux-wake-unlock >/dev/null 2>&1; then
  termux-wake-unlock || true
fi
echo "AhB stopped"
