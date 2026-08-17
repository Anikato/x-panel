#!/bin/bash
set -e
umask 077

# X-Panel 离线安装脚本（当前目录需包含发布包内容）
INSTALL_DIR="/opt/xpanel"
SERVICE_NAME="xpanel"
CONFIG_FILE="$INSTALL_DIR/config.yaml"
SYSTEMD_DIR="${SYSTEMD_DIR:-/etc/systemd/system}"
# Bundled Nezha Agent is always installed to the fixed path.
NEZHA_AGENT_DIR="/opt/xpanel/nezha-agent"
NEZHA_DASHBOARD_URL="${XPANEL_NEZHA_DASHBOARD_URL:-}"
NEZHA_AGENT_SECRET="${XPANEL_NEZHA_AGENT_SECRET:-}"
unset XPANEL_NEZHA_AGENT_SECRET
unset XPANEL_NEZHA_DASHBOARD_URL
NEZHA_SERVER=""
NEZHA_CONFIGURE=false
NEZHA_EXTERNAL_CONFLICT=false
NEZHA_WAS_ACTIVE=false
NEZHA_WAS_ENABLED=false
XPANEL_START_OK=true
IS_UPGRADE=false

while [ $# -gt 0 ]; do
    case "$1" in
        --nezha-dashboard)
            NEZHA_DASHBOARD_URL="$2"
            shift 2
            ;;
        --help|-h)
            echo "用法: bash install.sh [--nezha-dashboard https://dashboard.example.com]"
            echo "  XPANEL_NEZHA_DASHBOARD_URL / XPANEL_NEZHA_AGENT_SECRET 环境变量"
            echo "  AgentSecret 不接受命令行参数"
            exit 0
            ;;
        *)
            echo "错误：未知参数 $1" >&2
            exit 1
            ;;
    esac
done

echo "=============================="
echo "  X-Panel 安装脚本"
echo "=============================="

# 检查 root 权限
if [ "$(id -u)" -ne 0 ]; then
    echo "错误：请使用 root 用户运行此脚本"
    exit 1
fi

if [ -f "$INSTALL_DIR/xpanel" ]; then
    IS_UPGRADE=true
fi

# ---- Nezha helpers (aligned with install-online.sh) ----
nezha_dashboard_is_https_origin() {
    local value="$1"
    local port=""
    if [[ "$value" =~ ^https://(\[[0-9A-Fa-f:.%]+\]|[A-Za-z0-9][A-Za-z0-9.-]*)(:([0-9]{1,5}))?$ ]]; then
        port="${BASH_REMATCH[3]}"
        if [ -z "$port" ] || ((10#$port >= 1 && 10#$port <= 65535)); then
            return 0
        fi
    fi
    return 1
}

normalize_nezha_dashboard_server() {
    local value="$1"
    local host=""
    local port=""
    if [[ "$value" =~ ^https://(\[[0-9A-Fa-f:.%]+\]|[A-Za-z0-9][A-Za-z0-9.-]*)(:([0-9]{1,5}))?$ ]]; then
        host="${BASH_REMATCH[1]}"
        port="${BASH_REMATCH[3]}"
        host=$(printf '%s' "$host" | tr 'A-Z' 'a-z')
        if [ -z "$port" ]; then
            port="443"
        fi
        printf '%s:%s\n' "$host" "$port"
        return 0
    fi
    return 1
}

validate_nezha_install_inputs() {
    NEZHA_CONFIGURE=false
    NEZHA_SERVER=""
    if [ -z "${NEZHA_DASHBOARD_URL:-}" ] && [ -z "${NEZHA_AGENT_SECRET:-}" ]; then
        return 0
    fi
    if [ -n "${NEZHA_DASHBOARD_URL:-}" ] && [ -z "${NEZHA_AGENT_SECRET:-}" ]; then
        if [ -t 0 ]; then
            printf 'Nezha Agent Secret (input hidden): ' >&2
            read -rsp "" NEZHA_AGENT_SECRET || true
            printf '\n' >&2
        fi
        if [ -z "${NEZHA_AGENT_SECRET:-}" ]; then
            echo "错误：提供 --nezha-dashboard 时必须同时提供 XPANEL_NEZHA_AGENT_SECRET 或交互输入" >&2
            return 1
        fi
    fi
    if [ -z "${NEZHA_DASHBOARD_URL:-}" ] && [ -n "${NEZHA_AGENT_SECRET:-}" ]; then
        echo "错误：提供 AgentSecret 时必须同时提供 Dashboard 地址" >&2
        NEZHA_AGENT_SECRET=""
        return 1
    fi
    if ! nezha_dashboard_is_https_origin "${NEZHA_DASHBOARD_URL}"; then
        echo "错误：Nezha Dashboard 必须是合法 HTTPS origin" >&2
        NEZHA_AGENT_SECRET=""
        return 1
    fi
    if [ -z "${NEZHA_AGENT_SECRET}" ]; then
        echo "错误：AgentSecret 不能为空" >&2
        return 1
    fi
    # Only reject control input that cannot be a safe single-line YAML scalar.
    case "${NEZHA_AGENT_SECRET}" in
        *$'\n'*|*$'\r'*)
            echo "错误：AgentSecret 含有换行或控制字符" >&2
            NEZHA_AGENT_SECRET=""
            return 1
            ;;
    esac
    if printf '%s' "${NEZHA_AGENT_SECRET}" | grep -q '[[:cntrl:]]'; then
        echo "错误：AgentSecret 含有换行或控制字符" >&2
        NEZHA_AGENT_SECRET=""
        return 1
    fi
    NEZHA_SERVER="$(normalize_nezha_dashboard_server "${NEZHA_DASHBOARD_URL}")"
    NEZHA_CONFIGURE=true
    return 0
}

detect_external_nezha_agent() {
    NEZHA_EXTERNAL_CONFLICT=false
    local list_out line unit seen_units=""
    list_out="$(
        {
            systemctl list-unit-files --type=service --no-legend --no-pager 2>/dev/null || true
            systemctl list-units --type=service --all --no-legend --no-pager 2>/dev/null || true
        } | tr '\t' ' '
    )"
    while IFS= read -r line; do
        [ -z "${line}" ] && continue
        unit="${line%% *}"
        case "${unit}" in
            nezha-agent.service|nezha-agent@*.service)
                case " ${seen_units} " in
                    *" ${unit} "*) ;;
                    *)
                        seen_units="${seen_units} ${unit}"
                        NEZHA_EXTERNAL_CONFLICT=true
                        echo "警告：检测到外部 systemd unit ${unit}，不会启动捆绑 Agent"
                        ;;
                esac
                ;;
        esac
    done <<< "${list_out}"

    if systemctl cat nezha-agent.service >/dev/null 2>&1; then
        case " ${seen_units} " in
            *" nezha-agent.service "*) ;;
            *)
                NEZHA_EXTERNAL_CONFLICT=true
                echo "警告：检测到外部 nezha-agent.service，不会启动捆绑 Agent"
                ;;
        esac
    fi
    local d
    for d in /opt/nezha/agent /opt/nezha /usr/local/nezha/agent; do
        if [ -e "${d}/nezha-agent" ] || [ -e "${d}/agent" ]; then
            NEZHA_EXTERNAL_CONFLICT=true
            echo "警告：检测到外部 Nezha 目录 ${d}，不会启动捆绑 Agent"
        fi
    done
    return 0
}

# Full destination precheck before any systemctl stop or copy.
precheck_nezha_agent_targets() {
    local agent_dir="${NEZHA_AGENT_DIR}"
    local systemd_dir="${SYSTEMD_DIR}"
    local agent_dst="${agent_dir}/nezha-agent"
    local unit_dst="${systemd_dir}/xpanel-nezha-agent.service"
    local config_path="${agent_dir}/config.yml"

    if [ -L "${agent_dir}" ]; then
        echo "错误：Nezha Agent 目标目录是符号链接: ${agent_dir}" >&2
        return 1
    fi
    if [ -e "${agent_dir}" ] && [ ! -d "${agent_dir}" ]; then
        echo "错误：Nezha Agent 目标路径已存在且不是目录: ${agent_dir}" >&2
        return 1
    fi
    if [ -L "${agent_dst}" ]; then
        echo "错误：Nezha Agent 目标二进制是符号链接: ${agent_dst}" >&2
        return 1
    fi
    if [ -e "${agent_dst}" ] && [ ! -f "${agent_dst}" ]; then
        echo "错误：Nezha Agent 目标二进制已存在且不是普通文件: ${agent_dst}" >&2
        return 1
    fi
    if [ -L "${systemd_dir}" ]; then
        echo "错误：systemd 目录是符号链接: ${systemd_dir}" >&2
        return 1
    fi
    if [ -e "${systemd_dir}" ] && [ ! -d "${systemd_dir}" ]; then
        echo "错误：systemd 路径已存在且不是目录: ${systemd_dir}" >&2
        return 1
    fi
    if [ -L "${unit_dst}" ]; then
        echo "错误：目标 unit 是符号链接: ${unit_dst}" >&2
        return 1
    fi
    if [ -e "${unit_dst}" ] && [ ! -f "${unit_dst}" ]; then
        echo "错误：目标 unit 已存在且不是普通文件: ${unit_dst}" >&2
        return 1
    fi
    if [ -L "${config_path}" ]; then
        echo "错误：已有 config.yml 是符号链接" >&2
        return 1
    fi
    if [ -e "${config_path}" ] && [ ! -f "${config_path}" ]; then
        echo "错误：已有 config.yml 不是普通文件" >&2
        return 1
    fi
    return 0
}

capture_nezha_agent_systemd_state() {
    NEZHA_WAS_ACTIVE=false
    NEZHA_WAS_ENABLED=false
    if systemctl is-active --quiet xpanel-nezha-agent 2>/dev/null; then
        NEZHA_WAS_ACTIVE=true
    fi
    if systemctl is-enabled --quiet xpanel-nezha-agent 2>/dev/null; then
        NEZHA_WAS_ENABLED=true
    fi
    return 0
}

stop_nezha_agent_if_active() {
    if [ "${NEZHA_WAS_ACTIVE}" = true ]; then
        systemctl stop xpanel-nezha-agent 2>/dev/null || true
    fi
    return 0
}

require_nezha_agent_package() {
    local root="${1:-.}"
    local agent_bin="${root}/nezha-agent/nezha-agent"
    local unit_src="${root}/xpanel-nezha-agent.service"
    if [ ! -d "${root}/nezha-agent" ]; then
        echo "错误：安装包缺少 nezha-agent 目录" >&2
        return 1
    fi
    if [ -L "${agent_bin}" ] || [ ! -f "${agent_bin}" ] || [ ! -x "${agent_bin}" ]; then
        echo "错误：缺少可执行的 nezha-agent/nezha-agent 普通文件" >&2
        return 1
    fi
    if [ -L "${unit_src}" ] || [ ! -f "${unit_src}" ]; then
        echo "错误：缺少普通文件 xpanel-nezha-agent.service" >&2
        return 1
    fi
    return 0
}

write_initial_nezha_config() {
    local config_path="${NEZHA_AGENT_DIR}/config.yml"
    local secret_yaml
    if [ -L "${config_path}" ] || { [ -e "${config_path}" ] && [ ! -f "${config_path}" ]; }; then
        echo "错误：已有 config.yml 不是普通文件" >&2
        return 1
    fi
    if [ -f "${config_path}" ]; then
        local tmp
        tmp="$(mktemp "${NEZHA_AGENT_DIR}/.config.yml.XXXXXX")"
        chmod 0600 "${tmp}"
        if grep -q '^node_role:' "${config_path}"; then
            sed 's/^node_role:.*/node_role: xpanel/' "${config_path}" >"${tmp}" || { rm -f "${tmp}"; return 0; }
        else
            cat "${config_path}" >"${tmp}" || { rm -f "${tmp}"; return 0; }
            printf 'node_role: xpanel\n' >>"${tmp}" || { rm -f "${tmp}"; return 0; }
        fi
        chmod 0600 "${tmp}"
        mv -f "${tmp}" "${config_path}"
        chmod 0600 "${config_path}"
        return 0
    fi
    if [ "${NEZHA_CONFIGURE}" != true ]; then
        return 0
    fi
    if [ -z "${NEZHA_SERVER:-}" ] || [ -z "${NEZHA_AGENT_SECRET:-}" ]; then
        echo "错误：缺少写入初始 config.yml 所需的 server/secret" >&2
        return 1
    fi
    case "${NEZHA_AGENT_SECRET}" in
        *$'\n'*|*$'\r'*)
            echo "错误：AgentSecret 含有换行或控制字符" >&2
            NEZHA_AGENT_SECRET=""
            return 1
            ;;
    esac
    if printf '%s' "${NEZHA_AGENT_SECRET}" | grep -q '[[:cntrl:]]'; then
        echo "错误：AgentSecret 含有换行或控制字符" >&2
        NEZHA_AGENT_SECRET=""
        return 1
    fi
    mkdir -p "${NEZHA_AGENT_DIR}"
    chmod 0700 "${NEZHA_AGENT_DIR}"
    local tmp
    tmp="$(mktemp "${NEZHA_AGENT_DIR}/.config.yml.XXXXXX")"
    chmod 0600 "${tmp}"
    # Single-quoted YAML scalar; escape embedded single quotes as ''.
    # (bash ${var//\'/\'\'} is unreliable for this replacement)
    secret_yaml="$(printf '%s' "${NEZHA_AGENT_SECRET}" | sed "s/'/''/g")"
    {
        printf 'server: %s\n' "${NEZHA_SERVER}"
        printf "client_secret: '%s'\n" "${secret_yaml}"
        printf 'tls: true\n'
        printf 'insecure_tls: false\n'
        printf 'disable_auto_update: true\n'
        printf 'disable_force_update: true\n'
        printf 'disable_command_execute: false\n'
        printf 'node_role: xpanel\n'
    } >"${tmp}"
    chmod 0600 "${tmp}"
    mv -f "${tmp}" "${config_path}"
    chmod 0600 "${config_path}"
    NEZHA_AGENT_SECRET=""
    return 0
}

install_bundled_nezha_agent() {
    local package_dir="${NEZHA_PACKAGE_DIR:-.}"
    local unit_src="${NEZHA_UNIT_SRC:-${package_dir}/xpanel-nezha-agent.service}"
    local unit_dst="${SYSTEMD_DIR}/xpanel-nezha-agent.service"
    local agent_src="${package_dir}/nezha-agent/nezha-agent"
    local agent_dst="${NEZHA_AGENT_DIR}/nezha-agent"

    # Full target precheck before any mkdir/copy; never follow links or partial-replace.
    if [ -L "${NEZHA_AGENT_DIR}" ]; then
        echo "错误：Nezha Agent 目标目录是符号链接: ${NEZHA_AGENT_DIR}" >&2
        return 1
    fi
    if [ -e "${NEZHA_AGENT_DIR}" ] && [ ! -d "${NEZHA_AGENT_DIR}" ]; then
        echo "错误：Nezha Agent 目标路径已存在且不是目录: ${NEZHA_AGENT_DIR}" >&2
        return 1
    fi
    if [ -L "${agent_dst}" ]; then
        echo "错误：Nezha Agent 目标二进制是符号链接: ${agent_dst}" >&2
        return 1
    fi
    if [ -e "${agent_dst}" ] && [ ! -f "${agent_dst}" ]; then
        echo "错误：Nezha Agent 目标二进制已存在且不是普通文件: ${agent_dst}" >&2
        return 1
    fi
    if [ -L "${SYSTEMD_DIR}" ]; then
        echo "错误：systemd 目录是符号链接: ${SYSTEMD_DIR}" >&2
        return 1
    fi
    if [ -e "${SYSTEMD_DIR}" ] && [ ! -d "${SYSTEMD_DIR}" ]; then
        echo "错误：systemd 路径已存在且不是目录: ${SYSTEMD_DIR}" >&2
        return 1
    fi
    if [ -L "${unit_dst}" ]; then
        echo "错误：目标 unit 是符号链接: ${unit_dst}" >&2
        return 1
    fi
    if [ -e "${unit_dst}" ] && [ ! -f "${unit_dst}" ]; then
        echo "错误：目标 unit 已存在且不是普通文件: ${unit_dst}" >&2
        return 1
    fi
    if [ -L "${NEZHA_AGENT_DIR}/config.yml" ]; then
        echo "错误：已有 config.yml 是符号链接" >&2
        return 1
    fi
    if [ -e "${NEZHA_AGENT_DIR}/config.yml" ] && [ ! -f "${NEZHA_AGENT_DIR}/config.yml" ]; then
        echo "错误：已有 config.yml 不是普通文件" >&2
        return 1
    fi

    if [ -L "${agent_src}" ] || [ ! -f "${agent_src}" ] || [ ! -x "${agent_src}" ]; then
        echo "错误：包内 nezha-agent 无效" >&2
        return 1
    fi
    if [ -L "${unit_src}" ] || [ ! -f "${unit_src}" ]; then
        echo "错误：包内 xpanel-nezha-agent.service 无效" >&2
        return 1
    fi

    echo ">>> 安装捆绑 Nezha Agent: ${NEZHA_AGENT_DIR}"
    mkdir -p "${NEZHA_AGENT_DIR}"
    chmod 0700 "${NEZHA_AGENT_DIR}"
    cp -f "${agent_src}" "${agent_dst}"
    chmod 0755 "${agent_dst}"
    mkdir -p "${SYSTEMD_DIR}"
    cp -f "${unit_src}" "${unit_dst}"
    chmod 0644 "${unit_dst}"
    if ! write_initial_nezha_config; then
        return 1
    fi
    NEZHA_AGENT_SECRET=""
    return 0
}

finalize_nezha_agent_service() {
    systemctl daemon-reload 2>/dev/null || true
    if [ "${NEZHA_EXTERNAL_CONFLICT}" = true ]; then
        echo "警告：跳过捆绑 Nezha Agent 启动（外部冲突）"
        NEZHA_AGENT_SECRET=""
        return 0
    fi
    if [ "${XPANEL_START_OK:-true}" != true ]; then
        echo "警告：X-Panel 未成功启动，跳过捆绑 Nezha Agent 启动"
        systemctl disable xpanel-nezha-agent 2>/dev/null || true
        NEZHA_AGENT_SECRET=""
        return 0
    fi
    if [ "${IS_UPGRADE}" = true ]; then
        if [ ! -f "${NEZHA_AGENT_DIR}/config.yml" ]; then
            systemctl disable xpanel-nezha-agent 2>/dev/null || true
            return 0
        fi
        if [ "${NEZHA_WAS_ENABLED}" = true ]; then
            systemctl enable xpanel-nezha-agent >/dev/null 2>&1 || true
        else
            systemctl disable xpanel-nezha-agent 2>/dev/null || true
        fi
        if [ "${NEZHA_WAS_ACTIVE}" = true ]; then
            systemctl start xpanel-nezha-agent 2>/dev/null || true
        fi
        return 0
    fi
    if [ "${NEZHA_CONFIGURE}" = true ] && [ -f "${NEZHA_AGENT_DIR}/config.yml" ]; then
        systemctl enable xpanel-nezha-agent >/dev/null 2>&1 || true
        systemctl start xpanel-nezha-agent 2>/dev/null || true
    else
        systemctl disable xpanel-nezha-agent 2>/dev/null || true
    fi
    NEZHA_AGENT_SECRET=""
    return 0
}

# Preflight before any stop/copy.
if ! validate_nezha_install_inputs; then
    exit 1
fi
if ! require_nezha_agent_package "."; then
    exit 1
fi
NEZHA_PACKAGE_DIR="."
NEZHA_UNIT_SRC="./xpanel-nezha-agent.service"
detect_external_nezha_agent
# Full destination precheck must pass before capture/stop or any live replacement.
if ! precheck_nezha_agent_targets; then
    exit 1
fi
capture_nezha_agent_systemd_state
stop_nezha_agent_if_active

if [ "$IS_UPGRADE" = true ]; then
    systemctl stop $SERVICE_NAME 2>/dev/null || true
fi

# 创建安装目录
echo ">>> 创建安装目录: $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/data/db"
mkdir -p "$INSTALL_DIR/data/log"

# 复制二进制
if [ -f "./xpanel" ]; then
    echo ">>> 安装主程序..."
    cp -f ./xpanel "$INSTALL_DIR/xpanel"
    chmod +x "$INSTALL_DIR/xpanel"
else
    echo "错误：当前目录下未找到 xpanel 二进制文件"
    exit 1
fi

# 安装控制/救援命令
if [ -f "./xpctl" ]; then
    echo ">>> 安装 xpctl 控制工具..."
    cp -f ./xpctl /usr/local/bin/xpctl
    chmod +x /usr/local/bin/xpctl
fi

# Bundled Nezha Agent (fixed /opt/xpanel/nezha-agent)
if ! install_bundled_nezha_agent; then
    echo "错误：捆绑 Nezha Agent 安装失败" >&2
    exit 1
fi

# 复制配置文件（不覆盖已有配置）
if [ ! -f "$CONFIG_FILE" ]; then
    if [ -f "./config.yaml.example" ]; then
        echo ">>> 创建默认配置文件..."
        cp ./config.yaml.example "$CONFIG_FILE"
        # 生成随机 JWT Secret
        JWT_SECRET=$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | xxd -p)
        sed -i "s/dev-secret-change-in-production/$JWT_SECRET/" "$CONFIG_FILE"
        # 设置生产模式
        sed -i 's/mode: "debug"/mode: "release"/' "$CONFIG_FILE"
        # 设置数据目录为绝对路径
        sed -i "s|data_dir: \"./data\"|data_dir: \"$INSTALL_DIR/data\"|" "$CONFIG_FILE"
    fi
else
    echo ">>> 配置文件已存在，跳过"
fi

# 安装 systemd 服务
if [ -f "./xpanel.service" ]; then
    echo ">>> 安装 systemd 服务..."
    cp -f ./xpanel.service /etc/systemd/system/xpanel.service
    systemctl daemon-reload
    systemctl enable xpanel
fi

# X-Panel start must happen before Agent finalize; both before completion summary.
# X-Panel start failure must not start the bundled Agent. No state changes after summary.
XPANEL_START_OK=false
if systemctl start xpanel 2>/dev/null; then
    XPANEL_START_OK=true
else
    echo "警告：X-Panel 启动失败，不会启动捆绑 Nezha Agent" >&2
fi
finalize_nezha_agent_service

echo ""
echo "=============================="
echo "  X-Panel 安装完成!"
echo "=============================="
echo ""
echo "安装目录: $INSTALL_DIR"
echo "配置文件: $CONFIG_FILE"
echo "Nezha Agent: $NEZHA_AGENT_DIR"
echo ""
echo "启动命令:"
echo "  systemctl start xpanel"
echo ""
echo "查看状态:"
echo "  systemctl status xpanel"
echo ""
echo "控制工具:"
echo "  xpctl doctor"
echo "  xpctl backup db"
echo "  xpctl recover migrate --apply"
echo ""
echo "访问面板:"
echo "  http://<服务器IP>:7777"
echo ""
