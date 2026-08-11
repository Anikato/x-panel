#!/usr/bin/env bash
set -euo pipefail

for file in AGENTS.md RELEASE.md docs/dashboard-agent-xpanel.md; do
	test -f "$file" || {
		echo "missing required document: $file" >&2
		exit 1
	}
done

rg -q 'RELEASE\.md' AGENTS.md README.md
rg -q 'docs/dashboard-agent-xpanel\.md' AGENTS.md README.md RELEASE.md
rg -q 'workflow_dispatch' RELEASE.md
rg -q 'xpanel\.qm\.mk/releases/latest\.json' RELEASE.md
rg -q 'CUSTOM_AGENT_VERSION' RELEASE.md
rg -q '不得.*覆盖同版本|禁止.*覆盖同版本' AGENTS.md RELEASE.md

if rg -n '固定版本的官方|捆绑.*官方.*Agent' README.md docs/nezha-agent.md Makefile scripts/package-nezha-agent.sh; then
	echo "active documentation still describes the customized Agent as an official pinned asset" >&2
	exit 1
fi

echo "release documentation contract passed"
