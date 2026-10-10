#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
tag="$(sed -n 's/^ARG AHB_RELEASE_TAG=//p' Dockerfile)"
[[ "$tag" =~ ^linux-[0-9a-f]{12}$ ]] || { echo 'invalid Docker release tag' >&2; exit 1; }
grep -Fq "AHB_RELEASE_TAG: \${AHB_RELEASE_TAG:-$tag}" compose.yaml || { echo "compose/Dockerfile release mismatch" >&2; exit 1; }
grep -Fq "defaults to \`$tag\`" docs/DOCKER.md || { echo "Docker guide mismatch" >&2; exit 1; }
grep -Fq "releases/tag/$tag" README.md || { echo "README missing compiled baseline" >&2; exit 1; }
grep -Fq "DeepSeek Web: no native admin WebUI" README.md || { echo "DeepSeek UI guidance mismatch" >&2; exit 1; }
if grep -Fq "DeepSeek2API\x27s original management UI" README.md; then echo "obsolete DeepSeek UI" >&2; exit 1; fi
grep -Fq "v3.0.1" docs/PROVIDERS.md || { echo "Agent2API docs stale" >&2; exit 1; }
grep -Fq "Rust HTTP-only" docs/PROVIDERS.md || { echo "Duck docs stale" >&2; exit 1; }
echo "release documentation parity: PASS"
