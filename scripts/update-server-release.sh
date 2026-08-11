#!/usr/bin/env bash
set -euo pipefail

die() {
	echo "update-server-release: $*" >&2
	exit 1
}

atomic_install() {
	local src="$1" dst="$2" mode="${3:-0644}" parent base tmp
	[[ "$mode" =~ ^[0-7]{3,4}$ ]] || return 1
	parent="$(dirname "$dst")"
	base="$(basename "$dst")"
	[ -d "$parent" ] || return 1
	[ ! -d "$dst" ] && [ ! -L "$dst" ] || return 1
	tmp="$(mktemp "$parent/.${base}.new.XXXXXX")" || return 1
	if install -m "$mode" "$src" "$tmp" && mv -f -- "$tmp" "$dst"; then
		return 0
	fi
	rm -f -- "$tmp"
	return 1
}

file_mode() {
	stat -c '%a' "$1" 2>/dev/null || stat -f '%Lp' "$1"
}

backup_file() {
	local rel="$1" src="$ROOT/$rel" dst="$ROLLBACK/$rel"
	mkdir -p "$(dirname "$dst")"
	if [ -L "$src" ] || [ -d "$src" ]; then
		die "control path must be a regular file or absent: $rel"
	elif [ -f "$src" ]; then
		cp -p "$src" "$dst"
	elif [ -e "$src" ]; then
		die "unsupported control path: $rel"
	else
		: >"$dst.missing"
	fi
}

restore_file() {
	local rel="$1" saved="$ROLLBACK/$rel" live="$ROOT/$rel"
	if [ -f "$saved" ]; then
		mkdir -p "$(dirname "$live")"
		atomic_install "$saved" "$live" "$(file_mode "$saved")"
	elif [ -f "$saved.missing" ]; then
		rm -f "$live"
	fi
}

[ "$#" -eq 4 ] || die "usage: $0 stage|activate|rollback|finalize ROOT STAGE VERSION"
ACTION="$1"
ROOT_INPUT="$2"
STAGE_INPUT="$3"
VERSION="$4"

case "$ACTION" in
stage | activate | rollback | finalize) ;;
*) die "invalid action: $ACTION" ;;
esac
case "$ROOT_INPUT" in
/*) ;;
*) die "ROOT must be absolute" ;;
esac
case "$STAGE_INPUT" in
/*) ;;
*) die "STAGE must be absolute" ;;
esac
[[ "$ROOT_INPUT" =~ ^/[0-9A-Za-z._/-]+$ ]] || die "ROOT contains invalid characters"
[[ "$STAGE_INPUT" =~ ^/[0-9A-Za-z._/-]+$ ]] || die "STAGE contains invalid characters"
case "$ROOT_INPUT" in
*/./* | */../* | */. | */..) die "ROOT must not contain dot path segments" ;;
esac
case "$STAGE_INPUT" in
*/./* | */../* | */. | */..) die "STAGE must not contain dot path segments" ;;
esac
[[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]] || die "invalid VERSION"
[ -d "$ROOT_INPUT" ] || die "ROOT does not exist"
[ -d "$STAGE_INPUT" ] || die "STAGE does not exist"

ROOT="$(cd "$ROOT_INPUT" && pwd -P)"
STAGE="$(cd "$STAGE_INPUT" && pwd -P)"
[ "$ROOT" != "/" ] || die "ROOT must not be /"
case "$STAGE" in
"$ROOT"/.staging/*) ;;
*) die "STAGE must be below ROOT/.staging" ;;
esac
[ ! -L "$ROOT/releases" ] || die "ROOT/releases must not be a symlink"
if [ -e "$ROOT/releases" ] && [ ! -d "$ROOT/releases" ]; then
	die "ROOT/releases must be a directory or absent"
fi
[ ! -L "$STAGE/releases" ] || die "STAGE/releases must not be a symlink"

ROLLBACK="$STAGE/.rollback"
PREPARED="$STAGE/.prepared"
CREATED_ASSETS="$STAGE/.created-assets"
CONTROL_FILES=("releases/latest.json" "version.json" "install-online.sh" "install-xpctl.sh")

restore_control_files() {
	local rel
	for rel in "${CONTROL_FILES[@]}"; do
		restore_file "$rel"
	done
}

case "$ACTION" in
stage)
	source_release="$STAGE/releases/$VERSION"
	target_release="$ROOT/releases/$VERSION"
	[ ! -e "$PREPARED" ] || die "release is already prepared"
	[ -d "$source_release" ] && [ ! -L "$source_release" ] || die "staged version directory missing or invalid"
	for rel in "${CONTROL_FILES[@]}"; do
		[ -f "$STAGE/$rel" ] && [ ! -L "$STAGE/$rel" ] || die "staged control file missing or invalid: $rel"
	done
	(cd "$source_release" && sha256sum -c ./*.sha256)
	TOP_LEVEL_ASSETS=()
	for arch in amd64 arm64; do
		for suffix in .tar.gz .tar.gz.sha256; do
			file="$STAGE/xpanel-$VERSION-linux-$arch$suffix"
			[ -f "$file" ] && [ ! -L "$file" ] || die "top-level compatibility asset missing or invalid: $(basename "$file")"
			TOP_LEVEL_ASSETS+=("$file")
		done
		(cd "$STAGE" && sha256sum -c "xpanel-$VERSION-linux-$arch.tar.gz.sha256")
	done
	for file in "${TOP_LEVEL_ASSETS[@]}"; do
		destination="$ROOT/$(basename "$file")"
		if [ -L "$destination" ] || [ -d "$destination" ]; then
			die "top-level compatibility destination is unsafe: $(basename "$file")"
		elif [ -f "$destination" ]; then
			cmp -s "$file" "$destination" || die "top-level compatibility asset differs from staged bytes: $(basename "$file")"
		elif [ -e "$destination" ]; then
			die "top-level compatibility destination is invalid: $(basename "$file")"
		fi
	done
	mkdir -p "$ROOT/releases" "$ROLLBACK"
	for rel in "${CONTROL_FILES[@]}"; do
		backup_file "$rel"
	done
	if [ -L "$target_release" ]; then
		die "published version destination must not be a symlink"
	elif [ -d "$target_release" ]; then
		diff -qr "$source_release" "$target_release" >/dev/null || die "published version differs from staged bytes"
	elif [ -e "$target_release" ]; then
		die "published version destination is invalid"
	else
		release_tmp="$ROOT/releases/.$VERSION.new-$(basename "$STAGE")"
		[ ! -e "$release_tmp" ] && [ ! -L "$release_tmp" ] || die "temporary version destination already exists"
		if ! cp -a "$source_release" "$release_tmp"; then
			rm -rf "$release_tmp"
			die "failed to copy staged version directory"
		fi
		if ! mv "$release_tmp" "$target_release"; then
			rm -rf "$release_tmp"
			die "failed to activate immutable version directory"
		fi
	fi
	: >"$CREATED_ASSETS"
	for file in "${TOP_LEVEL_ASSETS[@]}"; do
		destination="$ROOT/$(basename "$file")"
		if [ ! -e "$destination" ]; then
			atomic_install "$file" "$destination" || die "failed to install top-level compatibility asset: $(basename "$file")"
			printf '%s\n' "$(basename "$file")" >>"$CREATED_ASSETS"
		fi
	done
	: >"$PREPARED"
	;;
activate)
	[ -f "$PREPARED" ] || die "release is not prepared"
	if atomic_install "$STAGE/install-online.sh" "$ROOT/install-online.sh" "$(file_mode "$STAGE/install-online.sh")" &&
		atomic_install "$STAGE/install-xpctl.sh" "$ROOT/install-xpctl.sh" "$(file_mode "$STAGE/install-xpctl.sh")" &&
		atomic_install "$STAGE/version.json" "$ROOT/version.json" &&
		atomic_install "$STAGE/releases/latest.json" "$ROOT/releases/latest.json"; then
		:
	else
		restore_control_files
		die "activation failed; previous control files restored"
	fi
	;;
rollback)
	[ -d "$ROLLBACK" ] || die "rollback state missing"
	restore_control_files
	for arch in amd64 arm64; do
		for suffix in .tar.gz .tar.gz.sha256; do
			name="xpanel-$VERSION-linux-$arch$suffix"
			if [ -f "$CREATED_ASSETS" ] && grep -Fxq "$name" "$CREATED_ASSETS"; then
				rm -f "$ROOT/$name"
			fi
		done
	done
	rm -f "$PREPARED" "$CREATED_ASSETS"
	;;
finalize)
	[ -f "$PREPARED" ] || die "release is not prepared"
	rm -rf "$STAGE"
	;;
esac
