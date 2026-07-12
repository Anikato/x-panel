#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
XPCTL="$ROOT_DIR/scripts/xpctl"
TMP_DIR="$(mktemp -d)"
INSTALL_DIR="$TMP_DIR/xpanel"
BIN_DIR="$TMP_DIR/bin"
DB_PATH="$INSTALL_DIR/data/db/xpanel.db"
BACKUP_ROOT="$INSTALL_DIR/backups/xpctl"
XPANEL_CALLS="$TMP_DIR/xpanel-calls"
SYSTEMCTL_CALLS="$TMP_DIR/systemctl-calls"
LAST_EXIT=0

cleanup() {
	rm -rf "$TMP_DIR"
}
trap cleanup EXIT

fail() {
	printf 'FAIL: %s\n' "$*" >&2
	exit 1
}

assert_exit() {
	[ "$LAST_EXIT" -eq "$1" ] || fail "exit=$LAST_EXIT, want $1"
}

assert_file_content() {
	[ "$(cat "$1")" = "$2" ] || fail "unexpected content in $1"
}

assert_backup_exists() {
	find "$BACKUP_ROOT" -type f -name xpanel.db -print -quit | grep -q . || fail "no database backup created"
}

run_xpctl() {
	set +e
	"$XPCTL" "$@" >"$TMP_DIR/stdout" 2>"$TMP_DIR/stderr"
	LAST_EXIT=$?
	set -e
}

mkdir -p "$INSTALL_DIR/data/db" "$BIN_DIR"
printf 'before\n' >"$DB_PATH"

cat >"$BIN_DIR/systemctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$SYSTEMCTL_CALLS"
if [ "${1:-}" = "is-active" ] && [ "${2:-}" = "--quiet" ]; then
	[ "${MOCK_SYSTEMCTL_STATE:-inactive}" = "active" ]
	exit
fi
exit 0
EOF

cat >"$BIN_DIR/flock" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [ "${1:-}" = "-n" ]; then
  exit 0
fi
shift
exec "$@"
EOF

cat >"$BIN_DIR/sqlite3" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
database="$1"
command="${2:-}"
case "$command" in
  ".backup "*)
    destination="${command#.backup }"
    destination="${destination#\'}"
    destination="${destination%\'}"
    mkdir -p "$(dirname "$destination")"
    cp "$database" "$destination"
    ;;
  "PRAGMA integrity_check;")
    printf 'ok\n'
    ;;
  *)
    printf 'unexpected sqlite command: %s\n' "$command" >&2
    exit 1
    ;;
esac
EOF

cat >"$INSTALL_DIR/xpanel" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$XPANEL_CALLS"
EOF

chmod +x "$BIN_DIR/systemctl" "$BIN_DIR/flock" "$BIN_DIR/sqlite3" "$INSTALL_DIR/xpanel"

export PATH="$BIN_DIR:$PATH"
export XPANEL_HOME="$INSTALL_DIR"
export XPANEL_DB="$DB_PATH"
export XPCTL_BACKUP_ROOT="$BACKUP_ROOT"
export XPANEL_CALLS
export SYSTEMCTL_CALLS

run_xpctl recover migrate
assert_exit 2
assert_file_content "$DB_PATH" "before"

export XPCTL_BACKUP_ROOT="$INSTALL_DIR/../outside"
run_xpctl backup db
assert_exit 2
[ ! -e "$TMP_DIR/outside" ] || fail "backup root traversal created a directory outside XPANEL_HOME"
export XPCTL_BACKUP_ROOT="$BACKUP_ROOT"

export MOCK_SYSTEMCTL_STATE=active
run_xpctl recover migrate --apply
assert_exit 2
assert_file_content "$DB_PATH" "before"

export MOCK_SYSTEMCTL_STATE=inactive
run_xpctl recover migrate --apply
assert_exit 0
assert_backup_exists
assert_file_content "$XPANEL_CALLS" "migrate"
if grep -q '^start ' "$SYSTEMCTL_CALLS"; then
	fail "recover migrate must not start the service"
fi

OUTSIDE_BACKUP="$TMP_DIR/outside.db"
printf 'outside\n' >"$OUTSIDE_BACKUP"
run_xpctl recover restore "$OUTSIDE_BACKUP" --yes
assert_exit 2
assert_file_content "$DB_PATH" "before"

SAFE_BACKUP_DIR="$BACKUP_ROOT/manual"
mkdir -p "$SAFE_BACKUP_DIR"
SAFE_BACKUP="$SAFE_BACKUP_DIR/xpanel.db"
printf 'restored\n' >"$SAFE_BACKUP"
run_xpctl recover restore "$SAFE_BACKUP" --yes
assert_exit 0
assert_file_content "$DB_PATH" "restored"

run_xpctl fix-migrations
assert_exit 2
assert_file_content "$DB_PATH" "restored"

printf 'PASS: xpctl recovery tests\n'
