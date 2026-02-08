#!/bin/bash
# 生成 version.json，用于放在更新服务器上供面板检查更新
# 用法: ./gen-version-json.sh v1.0.0 "这是更新说明"

VERSION=${1:-"dev"}
RELEASE_NOTE=${2:-""}
PUBLISH_DATE=$(date -u '+%Y-%m-%d')

cat > version.json << EOF
{
  "version": "${VERSION}",
  "releaseNote": "${RELEASE_NOTE}",
  "publishDate": "${PUBLISH_DATE}"
}
EOF

echo "Generated version.json:"
cat version.json
