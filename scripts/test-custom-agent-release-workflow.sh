#!/usr/bin/env bash
set -euo pipefail

workflow=".github/workflows/release.yml"

rg -q '^  workflow_dispatch:' "$workflow"
rg -q '^      version:' "$workflow"
rg -q 'inputs\.version' "$workflow"
rg -q '^concurrency:' "$workflow"
rg -q 'cancel-in-progress: false' "$workflow"

if rg -q '^  push:' "$workflow"; then
  echo "release workflow must not publish from a pushed tag" >&2
  exit 1
fi

rg -q 'vars\.CUSTOM_AGENT_REPO' "$workflow"
rg -q 'vars\.CUSTOM_AGENT_VERSION' "$workflow"
rg -q 'gh release download' "$workflow"
rg -q -- '--repo "\$\{CUSTOM_AGENT_REPO\}"' "$workflow"
rg -q -- '--pattern "nezha-agent_linux_\$\{ARCH\}\.zip"' "$workflow"
rg -q -- '--pattern "checksums\.txt"' "$workflow"
rg -q 'SOURCE_DATE_EPOCH=.*git show -s --format=%ct' "$workflow"
rg -q -- '--sort=name' "$workflow"
rg -q 'gzip -n' "$workflow"
rg -q 'targetCommitish' "$workflow"
rg -q 'target_commit.*GITHUB_SHA|GITHUB_SHA.*target_commit' "$workflow"
go_pin_count="$( { rg -n "go-version: '1\.26\.0'" "$workflow" || true; } | wc -l | tr -d ' ')"
test "$go_pin_count" -eq 2
rg -q "node-version: '24\.12\.0'" "$workflow"
rg -q -- '--json isDraft' "$workflow"
rg -q 'Agent release must be published|Agent release is a draft' "$workflow"
if rg -q 'vars\.UPDATE_BASE_URL' "$workflow"; then
	echo "production update origin must not be configurable" >&2
	exit 1
fi
rg -q 'UPDATE_BASE_URL: https://xpanel\.qm\.mk' "$workflow"
rg -q 'expected_prefix = f"https://xpanel\.qm\.mk/releases/\{expected_version\}/"' "$workflow"
rg -q 'for arch in \("linux-amd64", "linux-arm64"\)' "$workflow"
rg -q '\[\[ "\$UPDATE_SERVER_PATH" =~ \^/' "$workflow"
rg -q "prerelease:.*contains\(env\.RELEASE_VERSION, '-'\)" "$workflow"

rg -q 'draft: true' "$workflow"
rg -q 'gh release edit.*--draft=false' "$workflow"
rg -q '\.staging/' "$workflow"
rg -q 'update-server-release\.sh.*stage' "$workflow"
rg -q 'update-server-release\.sh.*activate' "$workflow"
rg -q 'update-server-release\.sh.*rollback' "$workflow"
rg -q 'releases/latest\.json' "$workflow"
rg -q 'bash scripts/test-release-documentation\.sh' "$workflow"

activated_line="$(rg -n '^[[:space:]]+activated=true$' "$workflow" | cut -d: -f1)"
activate_line="$(rg -n 'update-server-release\.sh.*activate' "$workflow" | tail -n 1 | cut -d: -f1)"
if [ -z "$activated_line" ] || [ -z "$activate_line" ] || [ "$activated_line" -ge "$activate_line" ]; then
	echo "rollback must be armed before remote activation starts" >&2
	exit 1
fi

if rg -q 'github\.com/nezha(hq)?/agent/releases/download' "$workflow"; then
  echo "release workflow must not fall back to the official Agent repository" >&2
  exit 1
fi

echo "custom Agent release workflow contract passed"
