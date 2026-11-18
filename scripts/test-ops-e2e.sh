#!/usr/bin/env bash
# Run against locally built Linux binaries. Requires Docker, Python 3, and a
# locally installed Node image. All three containers use fresh fixture data.
set -euo pipefail
if [[ $# != 1 ]]; then
    echo "Usage: $0 <directory containing dashboard, agent and xpanel binaries>" >&2
    exit 2
fi
image="${XPANEL_E2E_IMAGE:-node:24-bookworm-slim}"
docker image inspect "$image" >/dev/null
script_dir="$(cd "$(dirname "$0")" && pwd)"
bundle="$(cd "$1" && pwd)"
root="$(mktemp -d /var/tmp/xpanel-ops.XXXXXXXX)"
name="xpanel-ops-${root##*.}"
created=()
net_created=0
cleanup() {
    local exit_code=$?
    if (( exit_code != 0 )); then
        for instance in "${created[@]}"; do docker logs --tail 20 "$instance" >&2 || true; done
    fi
    for instance in "${created[@]}"; do docker rm -f "$instance" >/dev/null 2>&1 || true; done
    if (( net_created )); then docker network rm "$name" >/dev/null 2>&1 || true; fi
    rm -rf -- "$root"
}
trap cleanup EXIT
install -m 755 "$bundle/dashboard" "$bundle/agent" "$bundle/xpanel" "$root/"
install -m 644 "$script_dir/ops-e2e.mjs" "$root/smoke.mjs"
docker network create --internal "$name" >/dev/null
net_created=1
mkdir -p "$root/state" "$root/a" "$root/b" "$root/fake-systemd"
for node in a b; do
    mkdir -m 700 "$root/$node/db"
    cat > "$root/$node/config.yaml" <<'YAML'
system:
  data_dir: /opt/xpanel
  db_path: /opt/xpanel/db/test.db
  port: "17777"
  mode: release
YAML
    python3 - "$root/$node/db/test.db" <<'PY'
import sqlite3, sys
db = sqlite3.connect(sys.argv[1])
db.execute("CREATE TABLE settings (id integer primary key, key text unique, value text, about text, created_at datetime, updated_at datetime)")
db.execute("INSERT INTO settings (key,value) VALUES ('AppearanceConfig','fixture-personal')")
db.commit()
PY
done
# Check upgrade dispatch and recheck without fetching releases or changing systemd.
cat > "$root/fake-systemd/systemd-run" <<'SH'
#!/bin/sh
case "$*" in *"/opt/xpanel/xpanel update --latest"*) printf 'fixture upgrade queued\n' ;; *) exit 2 ;; esac
SH
cat > "$root/fake-systemd/systemctl" <<'SH'
#!/bin/sh
case "$1 $2" in
    "show xpanel-upgrade-"*) printf 'ActiveState=active\nSubState=exited\nResult=success\nExecMainStatus=0\n' ;;
    "is-active xpanel") printf 'active\n' ;;
    *) exit 3 ;;
esac
SH
chmod 755 "$root/fake-systemd/systemd-run" "$root/fake-systemd/systemctl"

secret="isolated-agent-smoke-secret-32-bytes"
jwt="isolated-dashboard-smoke-key-at-least-32-bytes"
cat > "$root/state/config.yaml" <<YAML
listen_host: 0.0.0.0
listen_port: 18080
agent_secret_key: "$secret"
site_name: Isolated X-Panel smoke test
YAML
docker_run() {
    local instance="$1"
    shift
    docker run -d --pull=never --name "$instance" --network "$name" "$@" >/dev/null
    created+=("$instance")
}
dashboard="$name-dashboard"
docker_run "$dashboard" --network-alias dashboard \
    --mount "type=bind,src=$root,dst=/stage,readonly" --mount "type=bind,src=$root/state,dst=/state" \
    -e NZ_JWTSECRETKEY="$jwt" \
    -e GIN_MODE=release -w /state --entrypoint /stage/dashboard "$image" -c /state/config.yaml -db /state/dashboard.db
start_agent() {
    local node="$1"
    local uuid="$2"
    cat > "$root/$node/agent.yaml" <<YAML
server: dashboard:18080
dns: ["127.0.0.11:53"]
custom_ip_api: ["http://dashboard:18080/api/v1/setting"]
client_secret: "$secret"
uuid: "$uuid"
node_role: xpanel
tls: false
disable_command_execute: false
disable_auto_update: true
report_delay: 1
debug: true
YAML
    docker_run "$name-$node" --mount "type=bind,src=$root/$node,dst=/opt/xpanel" \
      --mount "type=bind,src=$root,dst=/stage,readonly" --mount "type=bind,src=$root/xpanel,dst=/opt/xpanel/xpanel,readonly" \
      --entrypoint /bin/sh -w /opt/xpanel "$image" -ec '
        PATH="/stage/fake-systemd:$PATH" exec /stage/agent -c /opt/xpanel/agent.yaml
      '
}
ready() {
    for _ in $(seq 1 50); do
        if docker exec "$dashboard" node -e 'fetch("http://127.0.0.1:18080/api/v1/setting").then(r=>{if(!r.ok)process.exit(1)}).catch(()=>process.exit(1))' >/dev/null 2>&1; then return; fi
        sleep 0.4
    done
    docker logs --tail 20 "$dashboard"
    return 1
}
ready
start_agent a 11111111-2222-4333-8444-555555555555
start_agent b 11111111-2222-4333-8444-666666666666
docker exec "$dashboard" node /stage/smoke.mjs registered
docker stop -t 3 "$name-b" >/dev/null
sleep 1
docker exec "$dashboard" node /stage/smoke.mjs apply
docker restart -t 3 "$dashboard" >/dev/null
ready
docker start "$name-b" >/dev/null
sleep 3
docker exec "$dashboard" node /stage/smoke.mjs retry
docker exec "$dashboard" node /stage/smoke.mjs upgrade
for node in a b; do
    python3 - "$root/$node/db/test.db" <<'PY'
import json, sqlite3, sys
db = sqlite3.connect("file:" + sys.argv[1] + "?mode=ro", uri=True)
assert db.execute("select value from settings where key = 'AppearanceConfig'").fetchone()[0] == "fixture-personal"
profile = json.loads(db.execute("select value from settings where key = 'ManagedAppearanceConfig'").fetchone()[0])
assert profile["themeId"] == "harbor" and profile["mode"] == "dark"
PY
done
echo "Separate X-Panel fixture databases verified; personal settings preserved."
