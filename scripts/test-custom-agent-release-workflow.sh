#!/usr/bin/env bash
set -euo pipefail

workflow=".github/workflows/release.yml"

grep -Eq '^  workflow_dispatch:' "$workflow"
grep -Eq '^      version:' "$workflow"
grep -Eq 'inputs\.version' "$workflow"
grep -Eq '^concurrency:' "$workflow"
grep -Eq 'cancel-in-progress: false' "$workflow"

if grep -Eq '^  push:' "$workflow"; then
  echo "release workflow must not publish from a pushed tag" >&2
  exit 1
fi

grep -Eq 'vars\.CUSTOM_AGENT_REPO' "$workflow"
grep -Eq 'vars\.CUSTOM_AGENT_VERSION' "$workflow"
grep -Eq 'gh release download' "$workflow"
grep -Eq -- '--repo "\$\{CUSTOM_AGENT_REPO\}"' "$workflow"
grep -Eq -- '--pattern "nezha-agent_linux_\$\{ARCH\}\.zip"' "$workflow"
grep -Eq -- '--pattern "checksums\.txt"' "$workflow"
grep -Eq 'SOURCE_DATE_EPOCH=.*git show -s --format=%ct' "$workflow"
grep -Eq -- '--sort=name' "$workflow"
grep -Eq 'gzip -n' "$workflow"
grep -Eq 'targetCommitish' "$workflow"
grep -Eq 'target_commit.*GITHUB_SHA|GITHUB_SHA.*target_commit' "$workflow"
go_pin_count="$( { grep -En "go-version: '1\.26\.0'" "$workflow" || true; } | wc -l | tr -d ' ')"
test "$go_pin_count" -eq 2
grep -Eq "node-version: '24\.12\.0'" "$workflow"
grep -Eq -- '--json isDraft' "$workflow"
grep -Eq 'Agent release must be published|Agent release is a draft' "$workflow"
if grep -Eq 'vars\.UPDATE_BASE_URL' "$workflow"; then
	echo "production update origin must not be configurable" >&2
	exit 1
fi
grep -Eq 'UPDATE_BASE_URL: https://xpanel\.qm\.mk' "$workflow"
grep -Eq 'expected_prefix = f"https://xpanel\.qm\.mk/releases/\{expected_version\}/"' "$workflow"
grep -Eq 'for arch in \("linux-amd64", "linux-arm64"\)' "$workflow"
grep -Eq '\[\[ "\$UPDATE_SERVER_PATH" =~ \^/' "$workflow"
grep -Eq "prerelease:.*contains\(env\.RELEASE_VERSION, '-'\)" "$workflow"

grep -Eq 'draft: true' "$workflow"
grep -Eq 'gh release edit.*--draft=false' "$workflow"
grep -Eq '\.staging/' "$workflow"
grep -Eq 'update-server-release\.sh.*stage' "$workflow"
grep -Eq 'update-server-release\.sh.*activate' "$workflow"
grep -Eq 'update-server-release\.sh.*rollback' "$workflow"
grep -Eq 'releases/latest\.json' "$workflow"
grep -Eq 'bash scripts/test-release-documentation\.sh' "$workflow"

activated_line="$(grep -En '^[[:space:]]+activated=true$' "$workflow" | cut -d: -f1)"
activate_line="$(grep -En 'update-server-release\.sh.*activate' "$workflow" | tail -n 1 | cut -d: -f1)"
if [ -z "$activated_line" ] || [ -z "$activate_line" ] || [ "$activated_line" -ge "$activate_line" ]; then
	echo "rollback must be armed before remote activation starts" >&2
	exit 1
fi

if grep -Eq 'github\.com/nezha(hq)?/agent/releases/download' "$workflow"; then
  echo "release workflow must not fall back to the official Agent repository" >&2
  exit 1
fi

echo "custom Agent release workflow contract passed"
