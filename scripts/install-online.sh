#!/bin/bash
#
# X-Panel 一键安装脚本
#
# 用法:
#   curl -sSL https://raw.githubusercontent.com/Anikato/x-panel/main/scripts/install-online.sh | bash
#
# 自定义端口和安全入口:
#   curl -sSL ... | bash -s -- --port 8443 --entrance mySecret123
#
# 禁用 HTTPS（默认启用自签证书）:
#   curl -sSL ... | bash -s -- --no-ssl
#
# 安装指定版本:
#   curl -sSL ... | bash -s -- --version v1.0.0
#
# 卸载:
#   curl -sSL ... | bash -s -- --uninstall --yes
#

set -e

# ==================== 配置 ====================
GITHUB_REPO="Anikato/x-panel"
INSTALL_DIR="/opt/xpanel"
SERVICE_NAME="xpanel"
CONFIG_FILE="$INSTALL_DIR/config.yaml"
DEFAULT_PORT="9999"

# ==================== 颜色 ====================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }
log_step()  { echo -e "${BLUE}>>>${NC} $*"; }

# ==================== 参数解析 ====================
VERSION=""
UNINSTALL=false
GITHUB_TOKEN=""
YES=false
CUSTOM_PORT=""
ENTRANCE=""
ENABLE_SSL=true

while [[ $# -gt 0 ]]; do
    case $1 in
        --version|-v)
            VERSION="$2"
            shift 2
            ;;
        --token|-t)
            GITHUB_TOKEN="$2"
            shift 2
            ;;
        --port|-p)
            CUSTOM_PORT="$2"
            shift 2
            ;;
        --entrance|-e)
            ENTRANCE="$2"
            shift 2
            ;;
        --ssl)
            ENABLE_SSL=true
            shift
            ;;
        --no-ssl)
            ENABLE_SSL=false
            shift
            ;;
        --uninstall)
            UNINSTALL=true
            shift
            ;;
        --yes|-y)
            YES=true
            shift
            ;;
        --help|-h)
            echo "X-Panel 安装脚本"
            echo ""
            echo "用法:"
            echo "  bash install-online.sh [选项]"
            echo ""
            echo "选项:"
            echo "  --port, -p <端口>     自定义面板端口 (默认: 9999)"
            echo "  --entrance, -e <路径> 安全入口路径 (如 mySecret123)"
            echo "  --ssl                 启用 HTTPS 自签证书 (默认)"
            echo "  --no-ssl              禁用 HTTPS，使用 HTTP"
            echo "  --version, -v <版本>  安装指定版本 (如 v1.0.0)"
            echo "  --token, -t <TOKEN>   GitHub Token（私有仓库）"
            echo "  --uninstall           卸载 X-Panel"
            echo "  --yes, -y             跳过确认提示"
            echo "  --help, -h            显示帮助"
            exit 0
            ;;
        *)
            log_error "未知参数: $1"
            exit 1
            ;;
    esac
done

# 构建 curl 认证头
AUTH_HEADER=""
if [ -n "$GITHUB_TOKEN" ]; then
    AUTH_HEADER="Authorization: token $GITHUB_TOKEN"
    log_info "已配置 GitHub Token 认证"
fi

# 封装带认证的 curl 请求
github_curl() {
    if [ -n "$AUTH_HEADER" ]; then
        curl -sL -H "$AUTH_HEADER" "$@"
    else
        curl -sL "$@"
    fi
}

github_curl_with_code() {
    if [ -n "$AUTH_HEADER" ]; then
        curl -sL -H "$AUTH_HEADER" "$@"
    else
        curl -sL "$@"
    fi
}

# ==================== 卸载 ====================
do_uninstall() {
    echo ""
    echo -e "${RED}${BOLD}=============================="
    echo "  X-Panel 卸载"
    echo -e "==============================${NC}"
    echo ""

    if [ "$YES" != true ]; then
        # 从 /dev/tty 读取，支持 curl | bash 管道模式
        if [ -t 0 ]; then
            read -p "确定要卸载 X-Panel 吗？数据目录将被保留。(y/N): " confirm
        else
            read -p "确定要卸载 X-Panel 吗？数据目录将被保留。(y/N): " confirm < /dev/tty
        fi
        if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
            log_info "取消卸载"
            exit 0
        fi
    else
        log_info "跳过确认（--yes）"
    fi

    log_step "停止服务..."
    systemctl stop $SERVICE_NAME 2>/dev/null || true
    systemctl disable $SERVICE_NAME 2>/dev/null || true

    log_step "移除 systemd 服务..."
    rm -f /etc/systemd/system/${SERVICE_NAME}.service
    systemctl daemon-reload 2>/dev/null || true

    log_step "移除程序文件..."
    rm -f "$INSTALL_DIR/xpanel"
    rm -f "$INSTALL_DIR/xpanel.bak"

    echo ""
    log_info "卸载完成！"
    log_info "数据目录已保留: $INSTALL_DIR/data"
    log_info "配置文件已保留: $CONFIG_FILE"
    echo ""
    log_info "如需完全删除所有数据，请手动执行:"
    echo "  rm -rf $INSTALL_DIR"
    echo ""
    exit 0
}

if [ "$UNINSTALL" = true ]; then
    do_uninstall
fi

# ==================== 环境检查 ====================
echo ""
echo -e "${CYAN}${BOLD}╔══════════════════════════════════════╗"
echo "║                                      ║"
echo "║       X-Panel 一键安装脚本            ║"
echo "║       https://github.com/$GITHUB_REPO ║"
echo "║                                      ║"
echo -e "╚══════════════════════════════════════╝${NC}"
echo ""

# 检查 root 权限
if [ "$(id -u)" -ne 0 ]; then
    log_error "请使用 root 用户运行此脚本"
    log_info "提示: sudo bash install-online.sh"
    exit 1
fi

# 检查操作系统
if [[ "$(uname -s)" != "Linux" ]]; then
    log_error "仅支持 Linux 系统"
    exit 1
fi

# 检测系统架构
ARCH=$(uname -m)
case $ARCH in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        log_error "不支持的系统架构: $ARCH"
        log_info "支持的架构: x86_64 (amd64), aarch64 (arm64)"
        exit 1
        ;;
esac
log_info "系统架构: ${BOLD}$ARCH${NC}"

# 检查必要工具
for cmd in curl tar sha256sum; do
    if ! command -v $cmd &>/dev/null; then
        log_error "缺少必要工具: $cmd"
        log_info "请先安装: apt install -y $cmd 或 yum install -y $cmd"
        exit 1
    fi
done

# 检查是否已安装
IS_UPGRADE=false
if [ -f "$INSTALL_DIR/xpanel" ]; then
    CURRENT_VERSION=$("$INSTALL_DIR/xpanel" --version 2>/dev/null || echo "unknown")
    log_warn "检测到已安装 X-Panel (${CURRENT_VERSION})"
    IS_UPGRADE=true
fi

# ==================== 获取版本信息 ====================
log_step "获取版本信息..."

# 始终从 API 获取 Release 信息（私有仓库需要解析资产 API URL）
if [ -z "$VERSION" ]; then
    RELEASE_API_URL="https://api.github.com/repos/$GITHUB_REPO/releases/latest"
else
    RELEASE_API_URL="https://api.github.com/repos/$GITHUB_REPO/releases/tags/$VERSION"
fi

RELEASE_INFO=$(github_curl "$RELEASE_API_URL" 2>/dev/null)
if [ $? -ne 0 ] || [ -z "$RELEASE_INFO" ]; then
    log_error "无法连接到 GitHub，请检查网络连接"
    if [ -z "$GITHUB_TOKEN" ]; then
        log_info "如果是私有仓库，请使用 --token 参数提供 GitHub Token"
    fi
    exit 1
fi

# 检查 API 响应是否有效
if echo "$RELEASE_INFO" | grep -q '"message"'; then
    API_MSG=$(echo "$RELEASE_INFO" | grep '"message"' | head -1 | sed 's/.*"message": *"\([^"]*\)".*/\1/')
    log_error "GitHub API 错误: $API_MSG"
    if [ -z "$GITHUB_TOKEN" ]; then
        log_info "如果是私有仓库，请使用 --token 参数提供 GitHub Token"
    fi
    exit 1
fi

VERSION=$(echo "$RELEASE_INFO" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
if [ -z "$VERSION" ]; then
    log_error "无法获取版本号，请确认仓库已发布 Release"
    log_info "仓库地址: https://github.com/$GITHUB_REPO/releases"
    exit 1
fi

log_info "目标版本: ${BOLD}$VERSION${NC}"

# ==================== 下载 ====================
PKG_NAME="xpanel-${VERSION}-linux-${ARCH}"

TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

if [ -n "$GITHUB_TOKEN" ]; then
    # ===== 私有仓库：通过 GitHub API 资产端点下载 =====
    # 私有仓库的 browser_download_url 不支持 Token 头认证（302 到 S3 会丢弃 Auth 头）
    # 必须用 API 资产 URL + Accept: application/octet-stream 下载
    log_step "解析 Release 资产..."

    # 从 Release JSON 中提取资产的 API URL
    # JSON 结构: "url": "https://api.github.com/.../assets/ID", ... "name": "xpanel-xxx.tar.gz"
    DOWNLOAD_API_URL=$(echo "$RELEASE_INFO" | grep -B5 "\"name\": \"${PKG_NAME}.tar.gz\"" | grep '"url":.*api.github.com.*/assets/' | head -1 | sed 's/.*"url": *"\([^"]*\)".*/\1/')
    CHECKSUM_API_URL=$(echo "$RELEASE_INFO" | grep -B5 "\"name\": \"${PKG_NAME}.tar.gz.sha256\"" | grep '"url":.*api.github.com.*/assets/' | head -1 | sed 's/.*"url": *"\([^"]*\)".*/\1/')

    if [ -z "$DOWNLOAD_API_URL" ]; then
        log_error "未在 Release 中找到 ${PKG_NAME}.tar.gz 资产"
        log_info "可能原因: CI 构建尚未完成或构建失败"
        log_info "请检查: https://github.com/$GITHUB_REPO/actions"
        exit 1
    fi

    log_step "下载安装包 (通过 GitHub API)..."
    echo "  资产: ${PKG_NAME}.tar.gz"

    HTTP_CODE=$(curl -sL \
        -H "Authorization: token $GITHUB_TOKEN" \
        -H "Accept: application/octet-stream" \
        -w "%{http_code}" \
        -o "$TMP_DIR/xpanel.tar.gz" \
        "$DOWNLOAD_API_URL" 2>/dev/null)

    if [ "$HTTP_CODE" != "200" ]; then
        log_error "下载失败 (HTTP $HTTP_CODE)"
        log_info "请检查 Token 权限是否包含 repo 范围"
        exit 1
    fi

    DOWNLOAD_SIZE=$(du -h "$TMP_DIR/xpanel.tar.gz" | cut -f1)
    log_info "下载完成 (${DOWNLOAD_SIZE})"

    # 下载并校验 checksum
    log_step "校验文件完整性..."
    if [ -n "$CHECKSUM_API_URL" ]; then
        curl -sL \
            -H "Authorization: token $GITHUB_TOKEN" \
            -H "Accept: application/octet-stream" \
            -o "$TMP_DIR/checksum.sha256" \
            "$CHECKSUM_API_URL" 2>/dev/null

        if [ -f "$TMP_DIR/checksum.sha256" ] && [ -s "$TMP_DIR/checksum.sha256" ]; then
            EXPECTED_HASH=$(awk '{print $1}' "$TMP_DIR/checksum.sha256")
            ACTUAL_HASH=$(sha256sum "$TMP_DIR/xpanel.tar.gz" | awk '{print $1}')
            if [ "$EXPECTED_HASH" = "$ACTUAL_HASH" ]; then
                log_info "SHA256 校验通过 ✓"
            else
                log_error "SHA256 校验失败！"
                log_error "  期望: $EXPECTED_HASH"
                log_error "  实际: $ACTUAL_HASH"
                exit 1
            fi
        else
            log_warn "校验文件下载失败，跳过校验"
        fi
    else
        log_warn "未找到校验文件资产，跳过校验"
    fi

else
    # ===== 公开仓库：直接下载 =====
    DOWNLOAD_URL="https://github.com/$GITHUB_REPO/releases/download/$VERSION/${PKG_NAME}.tar.gz"
    CHECKSUM_URL="https://github.com/$GITHUB_REPO/releases/download/$VERSION/${PKG_NAME}.tar.gz.sha256"

    log_step "下载安装包..."
    echo "  URL: $DOWNLOAD_URL"

    HTTP_CODE=$(curl -sL -w "%{http_code}" -o "$TMP_DIR/xpanel.tar.gz" "$DOWNLOAD_URL" 2>/dev/null)
    if [ "$HTTP_CODE" != "200" ]; then
        log_error "下载失败 (HTTP $HTTP_CODE)"
        if [ "$HTTP_CODE" = "404" ]; then
            log_info "如果是私有仓库，请使用 --token 参数提供 GitHub Token"
        fi
        log_info "请检查版本号是否正确: $VERSION"
        exit 1
    fi

    DOWNLOAD_SIZE=$(du -h "$TMP_DIR/xpanel.tar.gz" | cut -f1)
    log_info "下载完成 (${DOWNLOAD_SIZE})"

    # 校验
    log_step "校验文件完整性..."
    if curl -sL -o "$TMP_DIR/checksum.sha256" "$CHECKSUM_URL" 2>/dev/null; then
        EXPECTED_HASH=$(awk '{print $1}' "$TMP_DIR/checksum.sha256")
        ACTUAL_HASH=$(sha256sum "$TMP_DIR/xpanel.tar.gz" | awk '{print $1}')
        if [ "$EXPECTED_HASH" = "$ACTUAL_HASH" ]; then
            log_info "SHA256 校验通过 ✓"
        else
            log_error "SHA256 校验失败！"
            log_error "  期望: $EXPECTED_HASH"
            log_error "  实际: $ACTUAL_HASH"
            exit 1
        fi
    else
        log_warn "未找到校验文件，跳过 SHA256 校验"
    fi
fi

# ==================== 解压 ====================
log_step "解压安装包..."
mkdir -p "$TMP_DIR/extract"
tar -xzf "$TMP_DIR/xpanel.tar.gz" -C "$TMP_DIR/extract"

if [ ! -f "$TMP_DIR/extract/xpanel" ]; then
    log_error "安装包格式异常：未找到 xpanel 二进制文件"
    exit 1
fi

# ==================== 安装 ====================
if [ "$IS_UPGRADE" = true ]; then
    log_step "升级模式：停止现有服务..."
    systemctl stop $SERVICE_NAME 2>/dev/null || true
    # 备份当前版本
    cp -f "$INSTALL_DIR/xpanel" "$INSTALL_DIR/xpanel.bak" 2>/dev/null || true
fi

# 创建目录结构
log_step "创建安装目录..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/data/db"
mkdir -p "$INSTALL_DIR/data/log"

# 安装二进制
log_step "安装主程序..."
cp -f "$TMP_DIR/extract/xpanel" "$INSTALL_DIR/xpanel"
chmod +x "$INSTALL_DIR/xpanel"

# 保存安装脚本副本（方便后续卸载/升级）
if [ -f "$0" ] && [ "$0" != "bash" ] && [ "$0" != "/dev/stdin" ]; then
    cp -f "$0" "$INSTALL_DIR/install.sh" 2>/dev/null || true
fi

# 确定端口
PANEL_PORT="${CUSTOM_PORT:-$DEFAULT_PORT}"

# ==================== SSL 证书 ====================
SSL_CERT_PATH="$INSTALL_DIR/data/ssl/server.crt"
SSL_KEY_PATH="$INSTALL_DIR/data/ssl/server.key"
SSL_ENABLED=false

if [ "$ENABLE_SSL" = true ]; then
    if command -v openssl &>/dev/null; then
        if [ ! -f "$SSL_CERT_PATH" ] || [ ! -f "$SSL_KEY_PATH" ]; then
            log_step "生成自签名 SSL 证书..."
            mkdir -p "$INSTALL_DIR/data/ssl"
            openssl req -x509 -nodes -newkey rsa:2048 \
                -keyout "$SSL_KEY_PATH" \
                -out "$SSL_CERT_PATH" \
                -days 3650 \
                -subj "/C=CN/ST=Server/L=Server/O=X-Panel/CN=xpanel.local" \
                2>/dev/null
            if [ $? -eq 0 ]; then
                log_info "自签名证书已生成（有效期 10 年）"
                SSL_ENABLED=true
            else
                log_warn "证书生成失败，将使用 HTTP"
            fi
        else
            log_info "SSL 证书已存在，跳过生成"
            SSL_ENABLED=true
        fi
    else
        log_warn "未找到 openssl，将使用 HTTP"
    fi
fi

# 首次安装：创建配置文件
if [ ! -f "$CONFIG_FILE" ]; then
    log_step "创建配置文件..."
    JWT_SECRET=$(openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | xxd -p | head -c 64)

    cat > "$CONFIG_FILE" << YAML
system:
  port: "${PANEL_PORT}"
  mode: "release"
  data_dir: "${INSTALL_DIR}/data"
  db_path: "db/xpanel.db"
  jwt_secret: "${JWT_SECRET}"
  session_timeout: 86400
  ssl:
    enable: ${SSL_ENABLED}
    cert_path: "${SSL_CERT_PATH}"
    key_path: "${SSL_KEY_PATH}"

log:
  level: "info"
  path: "log"
  max_size: 100
  max_age: 30
  compress: true

nginx:
  install_dir: "${INSTALL_DIR}/nginx"
  version: ""
YAML

    log_info "配置文件已生成: $CONFIG_FILE"
else
    log_info "配置文件已存在，跳过生成"
    # 升级时如果指定了新端口，更新配置
    if [ -n "$CUSTOM_PORT" ]; then
        sed -i "s/port: \"[0-9]*\"/port: \"${CUSTOM_PORT}\"/" "$CONFIG_FILE"
        log_info "已更新端口为 ${CUSTOM_PORT}"
    fi
fi

# 安装 systemd 服务
log_step "配置 systemd 服务..."
cat > /etc/systemd/system/${SERVICE_NAME}.service << EOF
[Unit]
Description=X-Panel Server Management Panel
Documentation=https://github.com/$GITHUB_REPO
After=network.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/xpanel
WorkingDirectory=$INSTALL_DIR
Restart=always
RestartSec=5
LimitNOFILE=65535
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable $SERVICE_NAME >/dev/null 2>&1

# ==================== 启动服务 ====================
log_step "启动 X-Panel..."
systemctl start $SERVICE_NAME

# 等待启动
sleep 2
if systemctl is-active --quiet $SERVICE_NAME; then
    log_info "X-Panel 启动成功 ✓"
else
    log_warn "X-Panel 可能还在启动中..."
    log_info "请稍后检查: systemctl status $SERVICE_NAME"
fi

# ==================== 安全入口 ====================
if [ -n "$ENTRANCE" ] && [ "$IS_UPGRADE" = false ]; then
    log_step "配置安全入口: /${ENTRANCE}"
    # 安全入口存储在数据库中，首次启动后通过 migration 初始化
    # 这里在启动后通过 SQLite 直接写入
fi

# ==================== 获取访问信息 ====================
SERVER_IP=$(curl -s4 --connect-timeout 3 https://ifconfig.me 2>/dev/null \
    || curl -s4 --connect-timeout 3 https://api.ipify.org 2>/dev/null \
    || hostname -I 2>/dev/null | awk '{print $1}' \
    || echo "<服务器IP>")

PORT=$PANEL_PORT
if [ "$SSL_ENABLED" = true ]; then
    PROTOCOL="https"
else
    PROTOCOL="http"
fi

# ==================== 完成 ====================
echo ""
echo -e "${GREEN}${BOLD}╔══════════════════════════════════════╗"
echo "║                                      ║"
echo "║    ✅ X-Panel 安装完成！               ║"
echo "║                                      ║"
echo -e "╚══════════════════════════════════════╝${NC}"
echo ""
echo -e "  ${BOLD}版本:${NC}     $VERSION"
echo -e "  ${BOLD}安装目录:${NC} $INSTALL_DIR"
echo -e "  ${BOLD}配置文件:${NC} $CONFIG_FILE"
if [ "$SSL_ENABLED" = true ]; then
echo -e "  ${BOLD}SSL:${NC}      ${GREEN}已启用（自签名证书）${NC}"
else
echo -e "  ${BOLD}SSL:${NC}      ${YELLOW}未启用${NC}"
fi
echo ""
if [ -n "$ENTRANCE" ]; then
echo -e "  ${BOLD}访问面板:${NC} ${CYAN}${PROTOCOL}://${SERVER_IP}:${PORT}/${ENTRANCE}${NC}"
echo -e "  ${YELLOW}  ⚠ 必须通过安全入口路径访问，直接访问根路径会返回 404${NC}"
else
echo -e "  ${BOLD}访问面板:${NC} ${CYAN}${PROTOCOL}://${SERVER_IP}:${PORT}${NC}"
fi
if [ "$SSL_ENABLED" = true ]; then
echo -e "  ${YELLOW}  (自签名证书，浏览器会提示不安全，点击继续访问即可)${NC}"
fi
echo ""
echo -e "  ${BOLD}常用命令:${NC}"
echo "    systemctl start $SERVICE_NAME     # 启动"
echo "    systemctl stop $SERVICE_NAME      # 停止"
echo "    systemctl restart $SERVICE_NAME   # 重启"
echo "    systemctl status $SERVICE_NAME    # 查看状态"
echo "    journalctl -u $SERVICE_NAME -f    # 查看日志"
echo ""
if [ "$IS_UPGRADE" = false ]; then
    echo -e "  ${YELLOW}${BOLD}⚠ 首次安装需要初始化管理员账户${NC}"
    echo -e "  ${YELLOW}请打开面板地址完成初始化设置${NC}"
    echo ""
fi
echo -e "  ${BOLD}卸载命令:${NC}"
echo "    curl -sSL https://raw.githubusercontent.com/$GITHUB_REPO/main/scripts/install-online.sh | bash -s -- --uninstall --yes"
echo ""

# ==================== 安全入口写入数据库 ====================
# 等服务启动后写入安全入口（SQLite 数据库在服务首次启动时创建）
if [ -n "$ENTRANCE" ] && [ "$IS_UPGRADE" = false ]; then
    sleep 3  # 等待服务初始化数据库
    DB_PATH="$INSTALL_DIR/data/db/xpanel.db"
    if command -v sqlite3 &>/dev/null && [ -f "$DB_PATH" ]; then
        sqlite3 "$DB_PATH" "UPDATE settings SET value='${ENTRANCE}' WHERE key='SecurityEntrance';" 2>/dev/null
        if [ $? -eq 0 ]; then
            log_info "安全入口已配置: /${ENTRANCE}"
        else
            log_warn "安全入口写入失败，请在面板设置中手动配置"
        fi
    else
        log_warn "sqlite3 未安装，安全入口需在面板设置中手动配置"
        log_info "安装 sqlite3: apt install -y sqlite3 或 yum install -y sqlite"
    fi
fi
