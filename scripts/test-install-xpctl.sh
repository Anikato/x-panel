#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
INSTALLER="$ROOT_DIR/scripts/install-xpctl.sh"
TMP_DIR="$(mktemp -d)"
MIRROR="$TMP_DIR/mirror"
BIN_DIR="$TMP_DIR/bin"
TARGET="$TMP_DIR/target/xpctl"
UPDATE_URL="https://fixture.test"
CURL_CALLS="$TMP_DIR/curl-calls"
INSTALL_CALLS="$TMP_DIR/install-calls"
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

assert_empty_file() {
	[ ! -s "$1" ] || fail "expected empty file: $1"
}

assert_no_curl_request() {
	if grep -Fqx "$1" "$CURL_CALLS"; then
		fail "unexpected curl request: $1"
	fi
}

run_installer() {
	set +e
	XPCTL_BIN_PATH="$TARGET" XPCTL_UPDATE_URL="$UPDATE_URL" "$INSTALLER" "$@" >"$TMP_DIR/stdout" 2>"$TMP_DIR/stderr"
	LAST_EXIT=$?
	set -e
}

create_release() {
	local version="$1"
	local content="$2"
	local release_dir="$MIRROR/releases/$version"
	local stage_dir="$TMP_DIR/stage-$version"
	local archive="$release_dir/xpanel-$version-linux-amd64.tar.gz"

	mkdir -p "$release_dir" "$stage_dir"
	printf '%s\n' "$content" >"$stage_dir/xpctl"
	chmod +x "$stage_dir/xpctl"
	tar -czf "$archive" -C "$stage_dir" xpctl
	/usr/bin/shasum -a 256 "$archive" | awk '{print $1 "  " FILENAME}' FILENAME="$(basename "$archive")" >"$archive.sha256"
}

mkdir -p "$MIRROR/releases" "$BIN_DIR" "$(dirname "$TARGET")"
printf '{"version":"v-test"}\n' >"$MIRROR/releases/latest.json"
create_release "v-test" "candidate-v-test"
create_release "v-fixed" "candidate-v-fixed"

cat >"$BIN_DIR/id" <<'EOF'
#!/usr/bin/env bash
if [ "${1:-}" = "-u" ]; then
  printf '0\n'
  exit 0
fi
exit 1
EOF

cat >"$BIN_DIR/uname" <<'EOF'
#!/usr/bin/env bash
case "${1:-}" in
  -s) printf 'Linux\n' ;;
  -m) printf 'x86_64\n' ;;
  *) exit 1 ;;
esac
EOF

cat >"$BIN_DIR/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
output=""
url=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o)
      output="$2"
      shift 2
      ;;
    -*)
      shift
      ;;
    *)
      url="$1"
      shift
      ;;
  esac
done
printf '%s\n' "$url" >>"$CURL_CALLS"
path="${url#${XPCTL_UPDATE_URL}}"
source="$TEST_MIRROR$path"
[ -f "$source" ]
if [ -n "$output" ]; then
  cp "$source" "$output"
else
  cat "$source"
fi
EOF

cat >"$BIN_DIR/sha256sum" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [ "${1:-}" = "-c" ]; then
  checksum_file="$2"
  expected="$(awk 'NR == 1 {print $1}' "$checksum_file")"
  filename="$(awk 'NR == 1 {print $2}' "$checksum_file")"
  actual="$(/usr/bin/shasum -a 256 "$filename" | awk '{print $1}')"
  [ "$actual" = "$expected" ]
  exit
fi
/usr/bin/shasum -a 256 "$@"
EOF

cat >"$BIN_DIR/install" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$INSTALL_CALLS"
[ "$1" = "-m" ]
[ "$2" = "0755" ]
mkdir -p "$(dirname "$4")"
cp "$3" "$4"
chmod "$2" "$4"
EOF

chmod +x "$BIN_DIR/id" "$BIN_DIR/uname" "$BIN_DIR/curl" "$BIN_DIR/sha256sum" "$BIN_DIR/install"
export PATH="$BIN_DIR:$PATH"
export TEST_MIRROR="$MIRROR"
export CURL_CALLS
export INSTALL_CALLS

run_installer
assert_exit 0
assert_file_content "$TARGET" "candidate-v-test"
grep -Eq '^-m 0755 .+ .+$' "$INSTALL_CALLS" || fail "install was not called with mode 0755"

: >"$INSTALL_CALLS"
run_installer
assert_exit 0
assert_empty_file "$INSTALL_CALLS"

: >"$CURL_CALLS"
run_installer --version v-fixed
assert_exit 0
assert_file_content "$TARGET" "candidate-v-fixed"
assert_no_curl_request "$UPDATE_URL/releases/latest.json"

printf 'broken checksum\n' >"$MIRROR/releases/v-test/xpanel-v-test-linux-amd64.tar.gz.sha256"
printf 'keep-me\n' >"$TARGET"
: >"$INSTALL_CALLS"
run_installer --version v-test
assert_exit 1
assert_file_content "$TARGET" "keep-me"
assert_empty_file "$INSTALL_CALLS"

printf 'PASS: xpctl standalone installer tests\n'
