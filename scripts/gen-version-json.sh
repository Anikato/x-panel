#!/bin/bash
# 生成自建更新服务器清单。
# 用法:
#   ./scripts/gen-version-json.sh v1.0.0 "更新说明" https://xpanel.qm.mk build publish

set -e

VERSION=${1:-"dev"}
RELEASE_NOTE=${2:-""}
BASE_URL=${3:-"https://xpanel.qm.mk"}
ARTIFACT_DIR=${4:-"build"}
OUT_DIR=${5:-"publish"}
PUBLISH_DATE=$(date -u '+%Y-%m-%d')

BASE_URL=${BASE_URL%/}
RELEASE_DIR="$OUT_DIR/releases/$VERSION"
mkdir -p "$RELEASE_DIR"

for file in "$ARTIFACT_DIR"/xpanel-"$VERSION"-linux-*.tar.gz "$ARTIFACT_DIR"/xpanel-"$VERSION"-linux-*.tar.gz.sha256; do
    if [ -f "$file" ]; then
        cp "$file" "$RELEASE_DIR/"
        cp "$file" "$OUT_DIR/"
    fi
done

python3 << PY
import glob
import json
import os
from pathlib import Path

version = ${VERSION@Q}
release_note = ${RELEASE_NOTE@Q}
base_url = ${BASE_URL@Q}
publish_date = ${PUBLISH_DATE@Q}
out_dir = Path(${OUT_DIR@Q})
release_dir = out_dir / "releases" / version

assets = {}
for package in sorted(glob.glob(str(release_dir / f"xpanel-{version}-linux-*.tar.gz"))):
    package_path = Path(package)
    checksum_path = Path(str(package_path) + ".sha256")
    sha256 = checksum_path.read_text(encoding="utf-8").split()[0] if checksum_path.exists() else ""
    arch = package_path.name.removesuffix(".tar.gz").split("-linux-", 1)[1]
    url = f"{base_url}/releases/{version}/{package_path.name}"
    assets[f"linux-{arch}"] = {
        "url": url,
        "checksumUrl": f"{url}.sha256",
        "sha256": sha256,
        "size": package_path.stat().st_size,
    }

manifest = {
    "version": version,
    "releaseNote": release_note,
    "publishDate": publish_date,
    "assets": assets,
}
(out_dir / "releases").mkdir(parents=True, exist_ok=True)
(out_dir / "releases" / "latest.json").write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
(out_dir / "version.json").write_text(json.dumps({
    "version": version,
    "releaseNote": release_note,
    "publishDate": publish_date,
}, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY

echo "Generated update manifest under $OUT_DIR"
