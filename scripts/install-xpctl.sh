#!/usr/bin/env bash
set -euo pipefail

DEFAULT_UPDATE_URL="https://xpanel.qm.mk"
UPDATE_URL="${XPCTL_UPDATE_URL:-$DEFAULT_UPDATE_URL}"
TARGET="${XPCTL_BIN_PATH:-/usr/local/bin/xpctl}"
VERSION=""

log() { printf '[install-xpctl] %s\n' "$*"; }
fail() { printf '[install-xpctl][ERROR] %s\n' "$*" >&2; exit 1; }

usage() {
	cat <<EOF
X-Panel xpctl 独立安装脚本

用法:
  install-xpctl.sh [--version <tag>] [--update-url <url>]

选项:
  --version <tag>       安装指定版本，例如 v0.7.73
  --update-url <url>    更新服务器地址，默认 $DEFAULT_UPDATE_URL
  --help                显示帮助
EOF
}

need_cmd() {
	command -v "$1" >/dev/null 2>&1 || fail "缺少命令: $1"
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--version)
			[ "$#" -ge 2 ] || fail "--version 需要版本号"
			VERSION="$2"
			shift 2
			;;
		--update-url)
			[ "$#" -ge 2 ] || fail "--update-url 需要地址"
			UPDATE_URL="$2"
			shift 2
			;;
		--help|-h)
			usage
			exit 0
			;;
		*)
			fail "未知参数: $1"
			;;
	esac
done

[ "$(id -u)" -eq 0 ] || fail "请使用 root 用户运行此脚本"
[ "$(uname -s)" = "Linux" ] || fail "仅支持 Linux 系统"

case "$(uname -m)" in
	x86_64|amd64) ARCH="amd64" ;;
	aarch64|arm64) ARCH="arm64" ;;
	*) fail "不支持的系统架构: $(uname -m)" ;;
esac

case "$TARGET" in
	/*) ;;
	*) fail "安装目标必须是绝对路径: $TARGET" ;;
esac

for cmd in curl tar sha256sum cmp install mktemp; do
	need_cmd "$cmd"
done

UPDATE_URL="${UPDATE_URL%/}"
[ -n "$UPDATE_URL" ] || fail "更新服务器地址不能为空"

if [ -z "$VERSION" ]; then
	latest_json="$(curl -fsSL "$UPDATE_URL/releases/latest.json")" || fail "无法获取最新版本信息"
	VERSION="$(printf '%s\n' "$latest_json" | sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([A-Za-z0-9._-]*\)".*/\1/p' | head -n 1)"
	[ -n "$VERSION" ] || fail "最新版本信息中缺少 version"
fi

case "$VERSION" in
	*[!A-Za-z0-9._-]*|"") fail "非法版本号: $VERSION" ;;
esac

package="xpanel-$VERSION-linux-$ARCH.tar.gz"
package_url="$UPDATE_URL/releases/$VERSION/$package"
checksum_url="$package_url.sha256"
tmp_dir="$(mktemp -d)"

cleanup() {
	rm -rf "$tmp_dir"
}
trap cleanup EXIT

archive="$tmp_dir/$package"
checksum="$archive.sha256"
extract_dir="$tmp_dir/extract"

log "下载 xpctl: $VERSION ($ARCH)"
curl -fsSL "$package_url" -o "$archive" || fail "下载安装包失败"
curl -fsSL "$checksum_url" -o "$checksum" || fail "下载校验文件失败"

(
	cd "$tmp_dir"
	sha256sum -c "$(basename "$checksum")" >/dev/null
) || fail "安装包 SHA-256 校验失败"

mkdir -p "$extract_dir"
tar -xzf "$archive" -C "$extract_dir" xpctl || fail "安装包中未找到 xpctl"
candidate="$extract_dir/xpctl"
[ -f "$candidate" ] && [ ! -L "$candidate" ] && [ -x "$candidate" ] || fail "解压出的 xpctl 无效"

if [ -e "$TARGET" ] || [ -L "$TARGET" ]; then
	[ -f "$TARGET" ] && [ ! -L "$TARGET" ] || fail "现有目标不是常规文件: $TARGET"
	if cmp -s "$candidate" "$TARGET"; then
		log "xpctl 已是目标版本，未修改: $TARGET"
		exit 0
	fi
fi

mkdir -p "$(dirname "$TARGET")"
install -m 0755 "$candidate" "$TARGET"
log "xpctl 已安装: $TARGET ($VERSION)"
