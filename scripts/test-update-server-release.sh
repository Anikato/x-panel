#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT="$ROOT_DIR/scripts/update-server-release.sh"
TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/xpanel-update-release.XXXXXX")"
trap 'rm -rf "$TMP_ROOT"' EXIT

file_mode() {
	stat -c '%a' "$1" 2>/dev/null || stat -f '%Lp' "$1"
}

LIVE="$TMP_ROOT/live"
STAGE="$LIVE/.staging/run-1"
VERSION="v0.7.83"
mkdir -p "$LIVE/releases" "$STAGE/releases/$VERSION"
printf '{"version":"v0.7.82"}\n' >"$LIVE/releases/latest.json"
printf '{"version":"v0.7.82"}\n' >"$LIVE/version.json"
printf 'old installer\n' >"$LIVE/install-online.sh"
printf 'old xpctl installer\n' >"$LIVE/install-xpctl.sh"
chmod 0750 "$LIVE/install-online.sh"
chmod 0751 "$LIVE/install-xpctl.sh"

for arch in amd64 arm64; do
	name="xpanel-$VERSION-linux-$arch.tar.gz"
	printf 'package-%s\n' "$arch" >"$STAGE/releases/$VERSION/$name"
	(cd "$STAGE/releases/$VERSION" && sha256sum "$name" >"$name.sha256")
	cp "$STAGE/releases/$VERSION/$name" "$STAGE/$name"
	cp "$STAGE/releases/$VERSION/$name.sha256" "$STAGE/$name.sha256"
done
printf '{"version":"v0.7.83"}\n' >"$STAGE/releases/latest.json"
printf '{"version":"v0.7.83"}\n' >"$STAGE/version.json"
printf 'new installer\n' >"$STAGE/install-online.sh"
printf 'new xpctl installer\n' >"$STAGE/install-xpctl.sh"
chmod 0755 "$STAGE/install-online.sh" "$STAGE/install-xpctl.sh"

mkdir -p "$TMP_ROOT/path-alias"
if multiline_output="$(bash "$SCRIPT" stage "$LIVE" "$STAGE" $'v0.7.83\ninvalid' 2>&1)"; then
	echo "stage unexpectedly accepted a multi-line VERSION" >&2
	exit 1
fi
grep -q 'invalid VERSION' <<<"$multiline_output"
if bash "$SCRIPT" stage "$TMP_ROOT/path-alias/../live" "$STAGE" "$VERSION"; then
	echo "stage unexpectedly accepted a ROOT path containing a dot segment" >&2
	exit 1
fi

bash "$SCRIPT" stage "$LIVE" "$STAGE" "$VERSION"
grep -q 'v0.7.82' "$LIVE/releases/latest.json"
grep -q 'old installer' "$LIVE/install-online.sh"
grep -q 'old xpctl installer' "$LIVE/install-xpctl.sh"
test "$(file_mode "$LIVE/install-online.sh")" = 750
test "$(file_mode "$LIVE/install-xpctl.sh")" = 751
test -f "$LIVE/releases/$VERSION/xpanel-$VERSION-linux-amd64.tar.gz"

bash "$SCRIPT" activate "$LIVE" "$STAGE" "$VERSION"
grep -q 'v0.7.83' "$LIVE/releases/latest.json"
grep -q 'new installer' "$LIVE/install-online.sh"
grep -q 'new xpctl installer' "$LIVE/install-xpctl.sh"
test "$(file_mode "$LIVE/install-online.sh")" = 755
test "$(file_mode "$LIVE/install-xpctl.sh")" = 755

bash "$SCRIPT" rollback "$LIVE" "$STAGE" "$VERSION"
grep -q 'v0.7.82' "$LIVE/releases/latest.json"
grep -q 'old installer' "$LIVE/install-online.sh"
grep -q 'old xpctl installer' "$LIVE/install-xpctl.sh"
test "$(file_mode "$LIVE/install-online.sh")" = 750
test "$(file_mode "$LIVE/install-xpctl.sh")" = 751
for arch in amd64 arm64; do
	name="xpanel-$VERSION-linux-$arch.tar.gz"
	cp "$STAGE/$name" "$LIVE/$name"
	cp "$STAGE/$name.sha256" "$LIVE/$name.sha256"
done

bash "$SCRIPT" stage "$LIVE" "$STAGE" "$VERSION"
bash "$SCRIPT" activate "$LIVE" "$STAGE" "$VERSION"
bash "$SCRIPT" rollback "$LIVE" "$STAGE" "$VERSION"
for arch in amd64 arm64; do
	name="xpanel-$VERSION-linux-$arch.tar.gz"
	test -f "$LIVE/$name"
	test -f "$LIVE/$name.sha256"
done

bash "$SCRIPT" stage "$LIVE" "$STAGE" "$VERSION"
rm -f "$STAGE/releases/latest.json"
if bash "$SCRIPT" activate "$LIVE" "$STAGE" "$VERSION"; then
	echo "activate unexpectedly succeeded without staged latest.json" >&2
	exit 1
fi
grep -q 'v0.7.82' "$LIVE/version.json"
grep -q 'v0.7.82' "$LIVE/releases/latest.json"
grep -q 'old installer' "$LIVE/install-online.sh"
grep -q 'old xpctl installer' "$LIVE/install-xpctl.sh"

printf '{"version":"v0.7.83"}\n' >"$STAGE/releases/latest.json"
bash "$SCRIPT" activate "$LIVE" "$STAGE" "$VERSION"
bash "$SCRIPT" finalize "$LIVE" "$STAGE" "$VERSION"
test ! -e "$STAGE"
grep -q 'v0.7.83' "$LIVE/releases/latest.json"

BROKEN_LIVE="$TMP_ROOT/broken-live"
BROKEN_VERSION="v0.7.84"
BROKEN_STAGE="$BROKEN_LIVE/.staging/run-copy-failure"
mkdir -p "$BROKEN_LIVE/releases" "$BROKEN_STAGE/releases/$BROKEN_VERSION"
printf '{"version":"v0.7.83"}\n' >"$BROKEN_LIVE/releases/latest.json"
printf '{"version":"v0.7.83"}\n' >"$BROKEN_LIVE/version.json"
printf 'old installer\n' >"$BROKEN_LIVE/install-online.sh"
printf 'old xpctl installer\n' >"$BROKEN_LIVE/install-xpctl.sh"
printf '{"version":"v0.7.84"}\n' >"$BROKEN_STAGE/releases/latest.json"
printf '{"version":"v0.7.84"}\n' >"$BROKEN_STAGE/version.json"
printf 'new installer\n' >"$BROKEN_STAGE/install-online.sh"
printf 'new xpctl installer\n' >"$BROKEN_STAGE/install-xpctl.sh"
for arch in amd64 arm64; do
	name="xpanel-$BROKEN_VERSION-linux-$arch.tar.gz"
	printf 'package-%s\n' "$arch" >"$BROKEN_STAGE/releases/$BROKEN_VERSION/$name"
	(cd "$BROKEN_STAGE/releases/$BROKEN_VERSION" && sha256sum "$name" >"$name.sha256")
	cp "$BROKEN_STAGE/releases/$BROKEN_VERSION/$name" "$BROKEN_STAGE/$name"
	cp "$BROKEN_STAGE/releases/$BROKEN_VERSION/$name.sha256" "$BROKEN_STAGE/$name.sha256"
done

FAKE_BIN="$TMP_ROOT/fake-bin"
mkdir -p "$FAKE_BIN"
printf '%s\n' \
	'#!/usr/bin/env bash' \
	'if [ "${1:-}" = "-a" ] && [[ "${3:-}" == */releases/.v*.new-* ]]; then' \
	'  mkdir -p "$3"' \
	'  printf partial >"$3/partial"' \
	'  exit 1' \
	'fi' \
	'exec "$REAL_CP" "$@"' >"$FAKE_BIN/cp"
chmod +x "$FAKE_BIN/cp"
REAL_CP="$(command -v cp)"
export REAL_CP
if PATH="$FAKE_BIN:$PATH" bash "$SCRIPT" stage "$BROKEN_LIVE" "$BROKEN_STAGE" "$BROKEN_VERSION"; then
	echo "stage unexpectedly succeeded after an injected version copy failure" >&2
	exit 1
fi
test ! -e "$BROKEN_LIVE/releases/$BROKEN_VERSION"
if find "$BROKEN_LIVE/releases" -maxdepth 1 -name ".$BROKEN_VERSION.new-*" -print -quit | grep -q .; then
	echo "partial temporary version directory was not cleaned" >&2
	exit 1
fi
rm -f "$FAKE_BIN/cp"

REAL_INSTALL="$(command -v install)"
export REAL_INSTALL
printf '%s\n' \
	'#!/usr/bin/env bash' \
	'case "${4:-}" in' \
	'  */xpanel-v0.7.84-linux-amd64.tar.gz | */.xpanel-v0.7.84-linux-amd64.tar.gz.new.*)' \
	'    printf partial >"$4"' \
	'    exit 1' \
	'    ;;' \
	'esac' \
	'exec "$REAL_INSTALL" "$@"' >"$FAKE_BIN/install"
chmod +x "$FAKE_BIN/install"
if PATH="$FAKE_BIN:$PATH" bash "$SCRIPT" stage "$BROKEN_LIVE" "$BROKEN_STAGE" "$BROKEN_VERSION"; then
	echo "stage unexpectedly succeeded after an injected top-level install failure" >&2
	exit 1
fi
test ! -e "$BROKEN_LIVE/xpanel-$BROKEN_VERSION-linux-amd64.tar.gz"
if find "$BROKEN_LIVE" -maxdepth 1 -name ".xpanel-$BROKEN_VERSION-linux-amd64.tar.gz.new.*" -print -quit | grep -q .; then
	echo "partial top-level compatibility asset was not cleaned" >&2
	exit 1
fi

UNSAFE_LIVE="$TMP_ROOT/unsafe-live"
OUTSIDE_RELEASES="$TMP_ROOT/outside-releases"
UNSAFE_STAGE="$UNSAFE_LIVE/.staging/run-symlink-parent"
mkdir -p "$UNSAFE_LIVE/.staging" "$OUTSIDE_RELEASES"
cp -a "$BROKEN_STAGE" "$UNSAFE_STAGE"
ln -s "$OUTSIDE_RELEASES" "$UNSAFE_LIVE/releases"
printf '{"version":"v0.7.83"}\n' >"$OUTSIDE_RELEASES/latest.json"
printf '{"version":"v0.7.83"}\n' >"$UNSAFE_LIVE/version.json"
printf 'old installer\n' >"$UNSAFE_LIVE/install-online.sh"
printf 'old xpctl installer\n' >"$UNSAFE_LIVE/install-xpctl.sh"
if bash "$SCRIPT" stage "$UNSAFE_LIVE" "$UNSAFE_STAGE" "$BROKEN_VERSION"; then
	echo "stage unexpectedly accepted a symlinked releases directory" >&2
	exit 1
fi
test ! -e "$OUTSIDE_RELEASES/$BROKEN_VERSION"

echo "update server release transaction passed"
