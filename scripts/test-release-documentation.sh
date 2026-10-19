#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
tag="$(sed -n 's/^ARG AHB_RELEASE_TAG=//p' Dockerfile)"
[[ "$tag" =~ ^linux-[0-9a-f]{12}$ ]] || { echo 'invalid Docker release tag' >&2; exit 1; }
grep -Fq "AHB_RELEASE_TAG: \${AHB_RELEASE_TAG:-$tag}" compose.yaml || { echo "compose/Dockerfile release mismatch" >&2; exit 1; }
grep -Fq "defaults to \`$tag\`" docs/DOCKER.md || { echo "Docker guide mismatch" >&2; exit 1; }
grep -Fq "releases/tag/$tag" README.md || { echo "README missing compiled baseline" >&2; exit 1; }
# Docker's independently verified pin may lag the latest published native release.
# Scope the native check to the release table so historical links cannot mask drift.
native_section="$(sed -n '/^## 最新版本與編譯狀態$/,/^## Android 與 Linux 版本差異$/p' README.md)"
native_sha="$(printf '%s\n' "$native_section" | sed -nE 's/^\*\*目前已發布的原生版本：`([0-9a-f]{12})`.*/\1/p')"
[[ "$native_sha" =~ ^[0-9a-f]{12}$ ]] || { echo "README native release SHA missing or invalid" >&2; exit 1; }
printf '%s\n' "$native_section" | grep -Fq "releases/tag/linux-$native_sha" || { echo "README native release link mismatch" >&2; exit 1; }
printf '%s\n' "$native_section" | grep -Fq "blob/prebuilt/source-commit.txt" || { echo "README Android source provenance link missing" >&2; exit 1; }
# Match the published Android source SHA against the native release pin.
source_rows="$(printf '%s\n' "$native_section" | grep -F 'blob/prebuilt/source-commit.txt')"
[[ "$(printf '%s\n' "$source_rows" | wc -l | tr -d ' ')" == "1" ]] || { echo "README Android provenance row duplicated" >&2; exit 1; }
[[ "$source_rows" == '| Android `prebuilt` |'* ]] || { echo "README Android provenance row malformed" >&2; exit 1; }
[[ "$source_rows" =~ `([0-9a-f]{40})` ]] || { echo "README Android source commit SHA missing or invalid" >&2; exit 1; }
source_sha="${BASH_REMATCH[1]}"
sha_count="$(printf '%s\n' "$source_rows" | grep -oE '`[0-9a-f]{40}`' | wc -l | tr -d ' ')"
[[ "$sha_count" == "1" ]] || { echo "README Android provenance has multiple source SHAs" >&2; exit 1; }
[[ "${source_sha:0:12}" == "$native_sha" ]] || { echo "README Android source SHA differs from native release" >&2; exit 1; }
grep -Fq "DeepSeek Web: no native admin WebUI" README.md || { echo "DeepSeek UI guidance mismatch" >&2; exit 1; }
if grep -Fq "DeepSeek2API's original management UI" README.md; then echo "obsolete DeepSeek UI" >&2; exit 1; fi
grep -Fq "v3.0.1" docs/PROVIDERS.md || { echo "Agent2API docs stale" >&2; exit 1; }
grep -Fq "Rust HTTP-only" docs/PROVIDERS.md || { echo "Duck docs stale" >&2; exit 1; }
echo "release documentation parity: PASS"
