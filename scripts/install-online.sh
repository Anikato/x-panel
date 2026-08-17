#!/bin/bash
#
# X-Panel 一键安装脚本
#
# 用法:
#   curl -sSL https://xpanel.qm.mk/install-online.sh | bash
#
# 自定义端口和安全入口:
#   curl -sSL ... | bash -s -- --port 8443 --entrance mySecret123
#
# 自定义安装路径:
#   curl -sSL ... | bash -s -- --path /usr/local/xpanel
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
umask 077

# ==================== 配置 ====================
GITHUB_REPO="Anikato/x-panel"
DEFAULT_UPDATE_URL="https://xpanel.qm.mk"
UPDATE_URL="$DEFAULT_UPDATE_URL"
DEFAULT_INSTALL_DIR="/opt/xpanel"
INSTALL_DIR="$DEFAULT_INSTALL_DIR"
SERVICE_NAME="xpanel"
DEFAULT_PORT="7777"

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

random_hex() {
    local byte_count="$1"
    if command -v openssl &>/dev/null; then
        openssl rand -hex "$byte_count"
    else
        od -An -N "$byte_count" -tx1 /dev/urandom | tr -d ' \n'
    fi
}

# ==================== Bundled Nezha Agent helpers ====================
# Agent always lives under /opt/xpanel/nezha-agent (even if X-Panel INSTALL_DIR is customized).
# When NEZHA_AGENT_DIR is unset, helpers use ${INSTALL_DIR}/nezha-agent (fixture tests).
# Public functions below are self-contained (fixture harness evals them by name).

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
            # Combined -r/-s/-p flags; secret must never be accepted via CLI flags.
            read -rsp "" NEZHA_AGENT_SECRET || true
            printf '\n' >&2
        fi
        if [ -z "${NEZHA_AGENT_SECRET:-}" ]; then
            log_error "提供 --nezha-dashboard / XPANEL_NEZHA_DASHBOARD_URL 时必须同时提供 AgentSecret（环境变量 XPANEL_NEZHA_AGENT_SECRET 或交互输入）"
            return 1
        fi
    fi

    if [ -z "${NEZHA_DASHBOARD_URL:-}" ] && [ -n "${NEZHA_AGENT_SECRET:-}" ]; then
        log_error "提供 AgentSecret 时必须同时提供 Dashboard HTTPS 地址（--nezha-dashboard 或 XPANEL_NEZHA_DASHBOARD_URL）"
        NEZHA_AGENT_SECRET=""
        return 1
    fi

    if ! nezha_dashboard_is_https_origin "${NEZHA_DASHBOARD_URL}"; then
        log_error "Nezha Dashboard 地址必须是无路径、查询参数、片段或凭据的 HTTPS origin，端口范围为 1-65535"
        NEZHA_AGENT_SECRET=""
        return 1
    fi

    # Reject empty / control characters that cannot be safely represented as a
    # single-line YAML scalar. Legal punctuation is allowed and single-quoted on write.
    if [ -z "${NEZHA_AGENT_SECRET}" ]; then
        log_error "AgentSecret 不能为空"
        return 1
    fi
    case "${NEZHA_AGENT_SECRET}" in
        *$'\n'*|*$'\r'*)
            log_error "AgentSecret 含有换行或控制字符，已拒绝"
            NEZHA_AGENT_SECRET=""
            return 1
            ;;
    esac
    if printf '%s' "${NEZHA_AGENT_SECRET}" | grep -q '[[:cntrl:]]'; then
        log_error "AgentSecret 含有换行或控制字符，已拒绝"
        NEZHA_AGENT_SECRET=""
        return 1
    fi

    NEZHA_SERVER="$(normalize_nezha_dashboard_server "${NEZHA_DASHBOARD_URL}")"
    if [ -z "${NEZHA_SERVER}" ]; then
        log_error "无法规范化 Nezha Dashboard 地址"
        NEZHA_AGENT_SECRET=""
        return 1
    fi

    NEZHA_CONFIGURE=true
    return 0
}

detect_external_nezha_agent() {
    NEZHA_EXTERNAL_CONFLICT=false
    local bundled agent_dir
    if [ -n "${NEZHA_AGENT_DIR:-}" ]; then
        agent_dir="${NEZHA_AGENT_DIR}"
    else
        agent_dir="${INSTALL_DIR}/nezha-agent"
    fi
    bundled="${agent_dir}/nezha-agent"

    # Parse list-unit-files / list-units for official/default and instantiated units.
    # Match: nezha-agent.service, nezha-agent@*.service
    # Never match our bundled unit: xpanel-nezha-agent.service
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
                        log_warn "检测到外部 systemd unit: ${unit}（不会停止或覆盖）"
                        ;;
                esac
                ;;
        esac
    done <<< "${list_out}"

    # Direct cat as an additional signal for the plain official unit.
    if systemctl cat nezha-agent.service >/dev/null 2>&1; then
        case " ${seen_units} " in
            *" nezha-agent.service "*) ;;
            *)
                NEZHA_EXTERNAL_CONFLICT=true
                log_warn "检测到外部 systemd unit: nezha-agent.service（不会停止或覆盖）"
                ;;
        esac
    fi

    # Common external install directories (overridable in tests via NEZHA_EXTERNAL_DIRS).
    local dirs=()
    if [ "${NEZHA_EXTERNAL_DIRS+set}" = "set" ] && [ "${#NEZHA_EXTERNAL_DIRS[@]}" -gt 0 ]; then
        dirs=("${NEZHA_EXTERNAL_DIRS[@]}")
    else
        dirs=("/opt/nezha/agent" "/opt/nezha" "/usr/local/nezha/agent")
    fi
    local d
    for d in "${dirs[@]}"; do
        if [ -e "${d}/nezha-agent" ] || [ -e "${d}/agent" ]; then
            NEZHA_EXTERNAL_CONFLICT=true
            log_warn "检测到外部 Nezha Agent 目录: ${d}（不会停止或覆盖）"
        fi
    done

    # Running processes whose executable is not the bundled path.
    if command -v pgrep >/dev/null 2>&1; then
        local pids pid exe
        pids="$(pgrep -x nezha-agent 2>/dev/null || true)"
        if [ -n "$pids" ]; then
            for pid in $pids; do
                exe=""
                if [ -L "/proc/${pid}/exe" ]; then
                    exe="$(readlink -f "/proc/${pid}/exe" 2>/dev/null || true)"
                fi
                if [ -n "$exe" ] && [ "$exe" != "$bundled" ]; then
                    NEZHA_EXTERNAL_CONFLICT=true
                    log_warn "检测到外部 Nezha Agent 进程 pid=${pid} exe=${exe}（不会停止或覆盖）"
                elif [ -z "$exe" ]; then
                    # Best-effort: process name match without resolvable path still warns.
                    NEZHA_EXTERNAL_CONFLICT=true
                    log_warn "检测到名为 nezha-agent 的进程 pid=${pid}（不会停止或覆盖）"
                fi
            done
        fi
    fi

    if [ "${NEZHA_EXTERNAL_CONFLICT}" = true ]; then
        log_warn "外部 Nezha Agent 冲突：X-Panel 安装继续，但不会启动捆绑的 xpanel-nezha-agent"
    fi
    return 0
}

# Full destination precheck before any systemctl stop or copy.
# Rejects symlinks and wrong types; never follows links; never partial-replaces.
precheck_nezha_agent_targets() {
    local agent_dir systemd_dir agent_dst unit_dst config_path
    if [ -n "${NEZHA_AGENT_DIR:-}" ]; then
        agent_dir="${NEZHA_AGENT_DIR}"
    else
        agent_dir="${INSTALL_DIR}/nezha-agent"
    fi
    if [ -n "${SYSTEMD_DIR:-}" ]; then
        systemd_dir="${SYSTEMD_DIR}"
    else
        systemd_dir="/etc/systemd/system"
    fi
    agent_dst="${agent_dir}/nezha-agent"
    unit_dst="${systemd_dir}/xpanel-nezha-agent.service"
    config_path="${agent_dir}/config.yml"

    if [ -L "${agent_dir}" ]; then
        log_error "Nezha Agent 目标目录是符号链接，拒绝安装: ${agent_dir}"
        return 1
    fi
    if [ -e "${agent_dir}" ] && [ ! -d "${agent_dir}" ]; then
        log_error "Nezha Agent 目标路径已存在且不是目录: ${agent_dir}"
        return 1
    fi

    if [ -L "${agent_dst}" ]; then
        log_error "Nezha Agent 目标二进制是符号链接，拒绝安装: ${agent_dst}"
        return 1
    fi
    if [ -e "${agent_dst}" ] && [ ! -f "${agent_dst}" ]; then
        log_error "Nezha Agent 目标二进制已存在且不是普通文件: ${agent_dst}"
        return 1
    fi

    if [ -L "${systemd_dir}" ]; then
        log_error "systemd 目录是符号链接，拒绝安装: ${systemd_dir}"
        return 1
    fi
    if [ -e "${systemd_dir}" ] && [ ! -d "${systemd_dir}" ]; then
        log_error "systemd 路径已存在且不是目录: ${systemd_dir}"
        return 1
    fi

    if [ -L "${unit_dst}" ]; then
        log_error "目标 unit 是符号链接，拒绝安装: ${unit_dst}"
        return 1
    fi
    if [ -e "${unit_dst}" ] && [ ! -f "${unit_dst}" ]; then
        log_error "目标 unit 已存在且不是普通文件: ${unit_dst}"
        return 1
    fi

    if [ -L "${config_path}" ]; then
        log_error "已有 config.yml 是符号链接，中止 Nezha Agent 安装"
        return 1
    fi
    if [ -e "${config_path}" ] && [ ! -f "${config_path}" ]; then
        log_error "已有 config.yml 不是普通文件，中止 Nezha Agent 安装"
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
        log_step "停止捆绑 Nezha Agent 服务以进行升级..."
        systemctl stop xpanel-nezha-agent 2>/dev/null || true
    fi
    return 0
}

require_nezha_agent_package() {
    local extract_dir="$1"
    local agent_bin="${extract_dir}/nezha-agent/nezha-agent"
    local unit_src="${extract_dir}/xpanel-nezha-agent.service"

    if [ ! -d "${extract_dir}/nezha-agent" ]; then
        log_error "安装包缺少 nezha-agent 目录"
        return 1
    fi
    if [ -L "${agent_bin}" ]; then
        log_error "安装包中的 nezha-agent 不能是符号链接"
        return 1
    fi
    if [ ! -f "${agent_bin}" ]; then
        log_error "安装包缺少 nezha-agent/nezha-agent 可执行文件"
        return 1
    fi
    if [ ! -x "${agent_bin}" ]; then
        log_error "安装包中的 nezha-agent/nezha-agent 不可执行"
        return 1
    fi
    if [ -L "${unit_src}" ]; then
        log_error "安装包中的 xpanel-nezha-agent.service 不能是符号链接"
        return 1
    fi
    if [ ! -f "${unit_src}" ]; then
        log_error "安装包缺少 xpanel-nezha-agent.service"
        return 1
    fi
    return 0
}

write_initial_nezha_config() {
    local agent_dir config_path secret_yaml
    if [ -n "${NEZHA_AGENT_DIR:-}" ]; then
        agent_dir="${NEZHA_AGENT_DIR}"
    else
        agent_dir="${INSTALL_DIR}/nezha-agent"
    fi
    config_path="${agent_dir}/config.yml"

    if [ -L "${config_path}" ]; then
        log_error "已有 config.yml 是符号链接，拒绝写入"
        return 1
    fi
    if [ -e "${config_path}" ]; then
        if [ ! -f "${config_path}" ]; then
            log_error "已有 config.yml 不是普通文件，拒绝写入"
            return 1
        fi
        # Existing file: merge only node_role. Never replace UUID/secret/unknown fields.
        local tmp
        tmp="$(mktemp "${agent_dir}/.config.yml.XXXXXX")"
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
        log_error "缺少写入初始 config.yml 所需的 server/secret"
        return 1
    fi
    case "${NEZHA_AGENT_SECRET}" in
        *$'\n'*|*$'\r'*)
            log_error "AgentSecret 含有换行或控制字符，拒绝写入 config.yml"
            NEZHA_AGENT_SECRET=""
            return 1
            ;;
    esac
    if printf '%s' "${NEZHA_AGENT_SECRET}" | grep -q '[[:cntrl:]]'; then
        log_error "AgentSecret 含有换行或控制字符，拒绝写入 config.yml"
        NEZHA_AGENT_SECRET=""
        return 1
    fi

    mkdir -p "${agent_dir}"
    chmod 0700 "${agent_dir}"

    local tmp
    tmp="$(mktemp "${agent_dir}/.config.yml.XXXXXX")"
    chmod 0600 "${tmp}"
    # Single-quoted YAML scalar; escape embedded single quotes as ''.
    # (bash ${var//\'/\'\'} is unreliable for this replacement)
    secret_yaml="$(printf '%s' "${NEZHA_AGENT_SECRET}" | sed "s/'/''/g")"
    if ! {
        printf 'server: %s\n' "${NEZHA_SERVER}"
        printf "client_secret: '%s'\n" "${secret_yaml}"
        printf 'tls: true\n'
        printf 'insecure_tls: false\n'
        printf 'disable_auto_update: true\n'
        printf 'disable_force_update: true\n'
        printf 'disable_command_execute: false\n'
        printf 'node_role: xpanel\n'
    } >"${tmp}"; then
        rm -f "${tmp}"
        log_error "写入临时 config.yml 失败"
        return 1
    fi
    chmod 0600 "${tmp}"
    mv -f "${tmp}" "${config_path}"
    chmod 0600 "${config_path}"
    # Drop secret from shell memory as soon as it is persisted.
    NEZHA_AGENT_SECRET=""
    return 0
}

install_bundled_nezha_agent() {
    local agent_dir package_dir unit_src unit_dst agent_src agent_dst systemd_dir
    if [ -n "${NEZHA_AGENT_DIR:-}" ]; then
        agent_dir="${NEZHA_AGENT_DIR}"
    else
        agent_dir="${INSTALL_DIR}/nezha-agent"
    fi
    if [ -n "${SYSTEMD_DIR:-}" ]; then
        systemd_dir="${SYSTEMD_DIR}"
    else
        systemd_dir="/etc/systemd/system"
    fi
    package_dir="${NEZHA_PACKAGE_DIR:-}"
    unit_src="${NEZHA_UNIT_SRC:-}"
    unit_dst="${systemd_dir}/xpanel-nezha-agent.service"
    agent_src="${package_dir}/nezha-agent/nezha-agent"
    agent_dst="${agent_dir}/nezha-agent"

    if [ -z "${package_dir}" ] || [ ! -d "${package_dir}" ]; then
        log_error "NEZHA_PACKAGE_DIR 未设置或无效"
        return 1
    fi
    if [ -z "${unit_src}" ]; then
        unit_src="${package_dir}/xpanel-nezha-agent.service"
    fi

    # Full target precheck before any mkdir/copy (fixture harness loads this function alone).
    # Mirrors precheck_nezha_agent_targets: reject symlinks/wrong types; never follow links.
    if [ -L "${agent_dir}" ]; then
        log_error "Nezha Agent 目标目录是符号链接，拒绝安装: ${agent_dir}"
        return 1
    fi
    if [ -e "${agent_dir}" ] && [ ! -d "${agent_dir}" ]; then
        log_error "Nezha Agent 目标路径已存在且不是目录: ${agent_dir}"
        return 1
    fi
    if [ -L "${agent_dst}" ]; then
        log_error "Nezha Agent 目标二进制是符号链接，拒绝安装: ${agent_dst}"
        return 1
    fi
    if [ -e "${agent_dst}" ] && [ ! -f "${agent_dst}" ]; then
        log_error "Nezha Agent 目标二进制已存在且不是普通文件: ${agent_dst}"
        return 1
    fi
    if [ -L "${systemd_dir}" ]; then
        log_error "systemd 目录是符号链接，拒绝安装: ${systemd_dir}"
        return 1
    fi
    if [ -e "${systemd_dir}" ] && [ ! -d "${systemd_dir}" ]; then
        log_error "systemd 路径已存在且不是目录: ${systemd_dir}"
        return 1
    fi
    if [ -L "${unit_dst}" ]; then
        log_error "目标 unit 是符号链接，拒绝安装: ${unit_dst}"
        return 1
    fi
    if [ -e "${unit_dst}" ] && [ ! -f "${unit_dst}" ]; then
        log_error "目标 unit 已存在且不是普通文件: ${unit_dst}"
        return 1
    fi
    if [ -L "${agent_dir}/config.yml" ]; then
        log_error "已有 config.yml 是符号链接，中止 Nezha Agent 安装"
        return 1
    fi
    if [ -e "${agent_dir}/config.yml" ] && [ ! -f "${agent_dir}/config.yml" ]; then
        log_error "已有 config.yml 不是普通文件，中止 Nezha Agent 安装"
        return 1
    fi

    log_step "安装捆绑 Nezha Agent 到 ${agent_dir}..."

    if [ -L "${agent_src}" ] || [ ! -f "${agent_src}" ] || [ ! -x "${agent_src}" ]; then
        log_error "包内 nezha-agent 无效"
        return 1
    fi
    if [ -L "${unit_src}" ] || [ ! -f "${unit_src}" ]; then
        log_error "包内 xpanel-nezha-agent.service 无效"
        return 1
    fi

    # Create destinations only after precheck; never mkdir/cp through a link.
    mkdir -p "${agent_dir}"
    chmod 0700 "${agent_dir}"
    cp -f "${agent_src}" "${agent_dst}"
    chmod 0755 "${agent_dst}"

    mkdir -p "${systemd_dir}"
    cp -f "${unit_src}" "${unit_dst}"
    chmod 0644 "${unit_dst}"

    if ! write_initial_nezha_config; then
        return 1
    fi

    # Ensure secret is never left in the environment after install attempt.
    NEZHA_AGENT_SECRET=""
    return 0
}

finalize_nezha_agent_service() {
    local agent_dir config_path
    if [ -n "${NEZHA_AGENT_DIR:-}" ]; then
        agent_dir="${NEZHA_AGENT_DIR}"
    else
        agent_dir="${INSTALL_DIR}/nezha-agent"
    fi
    config_path="${agent_dir}/config.yml"

    systemctl daemon-reload 2>/dev/null || true

    if [ "${NEZHA_EXTERNAL_CONFLICT}" = true ]; then
        log_warn "跳过捆绑 Nezha Agent 的 enable/start（存在外部 Agent 冲突）"
        NEZHA_AGENT_SECRET=""
        return 0
    fi

    # X-Panel start failure must not start the bundled Agent.
    if [ "${XPANEL_START_OK:-true}" != true ]; then
        log_warn "X-Panel 未成功启动，跳过捆绑 Nezha Agent 的 enable/start"
        systemctl disable xpanel-nezha-agent 2>/dev/null || true
        NEZHA_AGENT_SECRET=""
        return 0
    fi

    if [ "${IS_UPGRADE}" = true ]; then
        # Historical upgrade without config: install assets only, keep inactive.
        if [ ! -f "${config_path}" ]; then
            systemctl disable xpanel-nezha-agent 2>/dev/null || true
            NEZHA_AGENT_SECRET=""
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
        NEZHA_AGENT_SECRET=""
        return 0
    fi

    # Fresh install: only enable/start when a complete credential pair configured config.
    if [ "${NEZHA_CONFIGURE}" = true ] && [ -f "${config_path}" ]; then
        log_step "启用并启动捆绑 Nezha Agent..."
        systemctl enable xpanel-nezha-agent >/dev/null 2>&1 || true
        systemctl start xpanel-nezha-agent 2>/dev/null || true
    else
        systemctl disable xpanel-nezha-agent 2>/dev/null || true
    fi
    NEZHA_AGENT_SECRET=""
    return 0
}

# 交互式读取（支持 curl | bash 管道模式）
read_input() {
    local prompt="$1"
    local var_name="$2"
    local default_val="$3"
    if [ -t 0 ]; then
        read -p "$prompt" "$var_name"
    else
        read -p "$prompt" "$var_name" < /dev/tty
    fi
    # 如果为空，使用默认值
    if [ -z "${!var_name}" ] && [ -n "$default_val" ]; then
        eval "$var_name='$default_val'"
    fi
}

# ==================== 参数解析 ====================
VERSION=""
UNINSTALL=false
GITHUB_TOKEN=""
YES=false
CUSTOM_PORT=""
CUSTOM_PATH=""
ENTRANCE=""
ENABLE_SSL=true
AGENT_TOKEN=""
INIT_USERNAME=""
INIT_PASSWORD=""
ADMIN_PASSWORD_GENERATED=false
LOCAL_FILE=""   # --file 指定本地路径或任意 URL，跳过 GitHub 下载

# Nezha Agent credentials: secret is accepted only via env (or later interactive read).
# Copy then immediately clear the original environment variable.
NEZHA_DASHBOARD_URL="${XPANEL_NEZHA_DASHBOARD_URL:-}"
NEZHA_AGENT_SECRET="${XPANEL_NEZHA_AGENT_SECRET:-}"
unset XPANEL_NEZHA_AGENT_SECRET
unset XPANEL_NEZHA_DASHBOARD_URL
NEZHA_SERVER=""
NEZHA_CONFIGURE=false
NEZHA_EXTERNAL_CONFLICT=false
NEZHA_WAS_ACTIVE=false
NEZHA_WAS_ENABLED=false
# Fixed Agent root regardless of custom X-Panel --path.
NEZHA_AGENT_DIR="/opt/xpanel/nezha-agent"
SYSTEMD_DIR="${SYSTEMD_DIR:-/etc/systemd/system}"
NEZHA_PACKAGE_DIR=""
NEZHA_UNIT_SRC=""

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
        --path)
            CUSTOM_PATH="$2"
            shift 2
            ;;
        --entrance|-e)
            ENTRANCE="$2"
            shift 2
            ;;
        --agent-token)
            AGENT_TOKEN="$2"
            shift 2
            ;;
        --username|-u)
            INIT_USERNAME="$2"
            shift 2
            ;;
        --password|-P)
            INIT_PASSWORD="$2"
            shift 2
            ;;
        --nezha-dashboard)
            NEZHA_DASHBOARD_URL="$2"
            shift 2
            ;;
        --file|-f)
            LOCAL_FILE="$2"
            shift 2
            ;;
        --update-url)
            UPDATE_URL="${2%/}"
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
            echo "  --port, -p <端口>     自定义面板端口 (默认: $DEFAULT_PORT)"
            echo "  --path <路径>         自定义安装路径 (默认: $DEFAULT_INSTALL_DIR)"
            echo "  --entrance, -e <路径> 安全入口路径 (如 mySecret123)"
            echo "  --ssl                 启用 HTTPS 自签证书 (默认)"
            echo "  --no-ssl              禁用 HTTPS，使用 HTTP"
            echo "  --version, -v <版本>  安装指定版本 (如 v1.0.0)"
            echo "  --update-url <URL>    自定义更新服务器 (默认: $DEFAULT_UPDATE_URL)"
            echo "  --file, -f <路径|URL> 指定本地包路径或任意 URL（跳过 GitHub 下载）"
            echo "                        本地: --file /tmp/xpanel-v1.0.0-linux-amd64.tar.gz"
            echo "                        URL:  --file https://mirror.example.com/xpanel.tar.gz"
            echo "  --agent-token <TOKEN> 设置 Agent Token（用于被主面板管理）"
            echo "  --username, -u <用户名> 预设管理员用户名（跳过初始化向导）"
            echo "  --password, -P <密码>   预设管理员密码（跳过初始化向导）"
            echo "  --nezha-dashboard <URL>"
            echo "                        Nezha Dashboard HTTPS origin（仅地址，不含密钥）"
            echo "  XPANEL_NEZHA_DASHBOARD_URL 环境变量"
            echo "                        同上（可与 --nezha-dashboard 二选一）"
            echo "  XPANEL_NEZHA_AGENT_SECRET 环境变量"
            echo "                        Nezha AgentSecret（不接受命令行参数；读取后立即清除）"
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

# 应用自定义安装路径（不影响固定的 Nezha Agent 目录 /opt/xpanel/nezha-agent）
if [ -n "$CUSTOM_PATH" ]; then
    INSTALL_DIR="$CUSTOM_PATH"
fi
CONFIG_FILE="$INSTALL_DIR/config.yaml"

# Nezha preflight must run before any service stop / file replacement.
if ! validate_nezha_install_inputs; then
    exit 1
fi

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
        read_input "确定要卸载 X-Panel 吗？数据目录将被保留。(y/N): " confirm ""
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
log_info "安装路径: ${BOLD}$INSTALL_DIR${NC}"

# 检查必要工具
for cmd in curl tar sha256sum; do
    if ! command -v $cmd &>/dev/null; then
        log_error "缺少必要工具: $cmd"
        log_info "请先安装: apt install -y $cmd 或 yum install -y $cmd"
        exit 1
    fi
done

if ! command -v sqlite3 &>/dev/null; then
    log_step "安装 SQLite 工具（用于一致性备份和恢复校验）..."
    if command -v apt-get &>/dev/null; then
        apt-get update -y >/dev/null 2>&1
        apt-get install -y sqlite3 >/dev/null 2>&1
    elif command -v dnf &>/dev/null; then
        dnf install -y sqlite >/dev/null 2>&1
    elif command -v yum &>/dev/null; then
        yum install -y sqlite >/dev/null 2>&1
    elif command -v apk &>/dev/null; then
        apk add --no-cache sqlite >/dev/null 2>&1
    elif command -v pacman &>/dev/null; then
        pacman -S --noconfirm sqlite >/dev/null 2>&1
    else
        log_error "未检测到 sqlite3，且无法识别可用的包管理器"
        exit 1
    fi
    command -v sqlite3 &>/dev/null || {
        log_error "sqlite3 安装失败；xpctl 备份和恢复功能依赖该工具"
        exit 1
    }
fi

# 检查是否已安装
IS_UPGRADE=false
if [ -f "$INSTALL_DIR/xpanel" ]; then
    CURRENT_VERSION=$("$INSTALL_DIR/xpanel" --version 2>/dev/null || echo "unknown")
    log_warn "检测到已安装 X-Panel (${CURRENT_VERSION})"
    IS_UPGRADE=true
fi

if [ "$IS_UPGRADE" = false ]; then
    if { [ -n "$INIT_USERNAME" ] && [ -z "$INIT_PASSWORD" ]; } ||
       { [ -z "$INIT_USERNAME" ] && [ -n "$INIT_PASSWORD" ]; }; then
        log_error "全新安装必须同时提供 --username 和 --password"
        exit 1
    fi
    if [ -z "$INIT_USERNAME" ] && [ -z "$INIT_PASSWORD" ]; then
        INIT_USERNAME="admin"
        INIT_PASSWORD=$(random_hex 16)
        ADMIN_PASSWORD_GENERATED=true
        log_info "未提供管理员凭据，已生成高熵初始密码并将在服务启动前写入"
    fi
fi

TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

if [ -n "$LOCAL_FILE" ]; then
    # ==================== 本地文件 / 自定义 URL 模式 ====================
    log_step "使用指定安装包（跳过 GitHub 下载）..."
    if echo "$LOCAL_FILE" | grep -qE '^https?://'; then
        log_info "从 URL 下载: $LOCAL_FILE"
        HTTP_CODE=$(curl -L -w "%{http_code}" -o "$TMP_DIR/xpanel.tar.gz" "$LOCAL_FILE" 2>/dev/null)
        if [ "$HTTP_CODE" != "200" ]; then
            log_error "下载失败 (HTTP $HTTP_CODE): $LOCAL_FILE"
            exit 1
        fi
        log_info "下载完成 ($(du -h "$TMP_DIR/xpanel.tar.gz" | cut -f1))"
    else
        if [ ! -f "$LOCAL_FILE" ]; then
            log_error "文件不存在: $LOCAL_FILE"
            exit 1
        fi
        cp "$LOCAL_FILE" "$TMP_DIR/xpanel.tar.gz"
        log_info "使用本地文件: $LOCAL_FILE ($(du -h "$LOCAL_FILE" | cut -f1))"
    fi
    # 从文件名提取版本号（格式: xpanel-vX.X.X-linux-amd64.tar.gz）
    BASENAME=$(basename "$LOCAL_FILE")
    VERSION=$(echo "$BASENAME" | sed -n 's/xpanel-\(v[0-9][^-]*\)-.*/\1/p')
    [ -z "$VERSION" ] && VERSION="local"
    log_info "版本: ${BOLD}$VERSION${NC}"

else
    if [ -z "$GITHUB_TOKEN" ]; then
        # ==================== 获取版本信息（自建更新服务器模式）====================
        log_step "获取版本信息..."

        if [ -z "$VERSION" ]; then
            MANIFEST_URL="$UPDATE_URL/releases/latest.json"
            HTTP_CODE=$(curl -sL -w "%{http_code}" -o "$TMP_DIR/latest.json" "$MANIFEST_URL" 2>/dev/null)
            if [ "$HTTP_CODE" != "200" ]; then
                log_error "无法获取更新清单 (HTTP $HTTP_CODE): $MANIFEST_URL"
                log_info "请确认更新服务器已发布 releases/latest.json"
                exit 1
            fi
            VERSION=$(grep '"version"' "$TMP_DIR/latest.json" | head -1 | sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
            if [ -z "$VERSION" ]; then
                log_error "无法从更新清单解析版本号"
                exit 1
            fi
        fi

        log_info "目标版本: ${BOLD}$VERSION${NC}"
        PKG_NAME="xpanel-${VERSION}-linux-${ARCH}"
        DOWNLOAD_URL="$UPDATE_URL/releases/$VERSION/${PKG_NAME}.tar.gz"
        CHECKSUM_URL="${DOWNLOAD_URL}.sha256"

        log_step "下载安装包..."
        echo "  URL: $DOWNLOAD_URL"
        HTTP_CODE=$(curl -sL -w "%{http_code}" -o "$TMP_DIR/xpanel.tar.gz" "$DOWNLOAD_URL" 2>/dev/null)
        if [ "$HTTP_CODE" != "200" ]; then
            log_error "下载失败 (HTTP $HTTP_CODE)"
            log_info "请检查更新服务器文件是否存在: $DOWNLOAD_URL"
            exit 1
        fi

        log_info "下载完成 ($(du -h "$TMP_DIR/xpanel.tar.gz" | cut -f1))"

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
    else
    # ==================== 获取版本信息（GitHub 模式）====================
    log_step "获取版本信息..."

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
        log_info "国内网络受限时，可先下载安装包再用 --file 参数离线安装"
        exit 1
    fi

    if echo "$RELEASE_INFO" | grep -q '"message"'; then
        API_MSG=$(echo "$RELEASE_INFO" | grep '"message"' | head -1 | sed 's/.*"message": *"\([^"]*\)".*/\1/')
        log_error "GitHub API 错误: $API_MSG"
        [ -z "$GITHUB_TOKEN" ] && log_info "如果是私有仓库，请使用 --token 参数提供 GitHub Token"
        exit 1
    fi

    VERSION=$(echo "$RELEASE_INFO" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
    if [ -z "$VERSION" ]; then
        log_error "无法获取版本号，请确认仓库已发布 Release"
        log_info "仓库地址: https://github.com/$GITHUB_REPO/releases"
        exit 1
    fi

    log_info "目标版本: ${BOLD}$VERSION${NC}"

    # ==================== 下载（GitHub 模式）====================
    PKG_NAME="xpanel-${VERSION}-linux-${ARCH}"

    if [ -n "$GITHUB_TOKEN" ]; then
        # 私有仓库：通过 GitHub API 资产端点下载
        log_step "解析 Release 资产..."
        DOWNLOAD_API_URL=$(echo "$RELEASE_INFO" | grep -B5 "\"name\": \"${PKG_NAME}.tar.gz\"" | grep '"url\":.*api.github.com.*/assets/' | head -1 | sed 's/.*"url": *"\([^"]*\)".*/\1/')
        CHECKSUM_API_URL=$(echo "$RELEASE_INFO" | grep -B5 "\"name\": \"${PKG_NAME}.tar.gz.sha256\"" | grep '"url\":.*api.github.com.*/assets/' | head -1 | sed 's/.*"url": *"\([^"]*\)".*/\1/')

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

        log_info "下载完成 ($(du -h "$TMP_DIR/xpanel.tar.gz" | cut -f1))"

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
        # 公开仓库：直接下载
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
            log_info "国内网络受限时，可先下载安装包再用 --file 参数离线安装"
            exit 1
        fi

        log_info "下载完成 ($(du -h "$TMP_DIR/xpanel.tar.gz" | cut -f1))"

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

# Require bundled Nezha assets before stopping services or replacing live files.
if ! require_nezha_agent_package "$TMP_DIR/extract"; then
    exit 1
fi
NEZHA_PACKAGE_DIR="$TMP_DIR/extract"
NEZHA_UNIT_SRC="$TMP_DIR/extract/xpanel-nezha-agent.service"
# Fixed path even when X-Panel uses a custom --path.
NEZHA_AGENT_DIR="/opt/xpanel/nezha-agent"
SYSTEMD_DIR="${SYSTEMD_DIR:-/etc/systemd/system}"

detect_external_nezha_agent
# Full destination precheck must pass before capture/stop or any live replacement.
if ! precheck_nezha_agent_targets; then
    exit 1
fi
capture_nezha_agent_systemd_state
stop_nezha_agent_if_active

# ==================== 安装 ====================
if [ "$IS_UPGRADE" = true ]; then
    log_step "升级模式：停止现有服务..."
    systemctl stop $SERVICE_NAME 2>/dev/null || true
    # 备份当前版本
    cp -f "$INSTALL_DIR/xpanel" "$INSTALL_DIR/xpanel.bak" 2>/dev/null || true
fi

# 创建目录结构
log_step "创建安装目录: $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/data/db"
mkdir -p "$INSTALL_DIR/data/log"
mkdir -p "$INSTALL_DIR/secrets"
mkdir -p "$INSTALL_DIR/backups"
chmod 0700 "$INSTALL_DIR/data" "$INSTALL_DIR/data/db" "$INSTALL_DIR/data/log" "$INSTALL_DIR/secrets" "$INSTALL_DIR/backups"

# 安装二进制
log_step "安装主程序..."
cp -f "$TMP_DIR/extract/xpanel" "$INSTALL_DIR/xpanel"
chmod 0755 "$INSTALL_DIR/xpanel"

# Install bundled Nezha Agent (binary + unit; config only when credential pair present).
if ! install_bundled_nezha_agent; then
    log_error "捆绑 Nezha Agent 安装失败"
    exit 1
fi

if [ -f "$TMP_DIR/extract/xpctl" ]; then
    log_step "安装 xpctl 控制工具..."
    cp -f "$TMP_DIR/extract/xpctl" /usr/local/bin/xpctl
    chmod 0755 /usr/local/bin/xpctl
fi

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
                chmod 0600 "$SSL_CERT_PATH" "$SSL_KEY_PATH"
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
    JWT_SECRET=$(random_hex 32)

    cat > "$CONFIG_FILE" << YAML
system:
  port: "${PANEL_PORT}"
  mode: "release"
  data_dir: "${INSTALL_DIR}/data"
  db_path: "db/xpanel.db"
  credential_key_path: "${INSTALL_DIR}/secrets/credential-keyring.json"
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
  build_repo: "Anikato/nginx-build"
YAML
    chmod 0600 "$CONFIG_FILE"

    log_info "配置文件已生成: $CONFIG_FILE"
else
    chmod 0600 "$CONFIG_FILE"
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
UMask=0077
LimitNOFILE=65535
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable $SERVICE_NAME >/dev/null 2>&1

# ==================== 首次启动前安全初始化 ====================
if [ "$IS_UPGRADE" = false ]; then
    if [ -n "$INIT_USERNAME" ] && [ -n "$INIT_PASSWORD" ]; then
        log_step "配置管理员账户..."
        if SETUP_LOG=$(cd "$INSTALL_DIR" && ./xpanel setup --username "$INIT_USERNAME" --password "$INIT_PASSWORD" 2>&1); then
            log_info "管理员账户已设置: ${INIT_USERNAME}"
        else
            log_error "管理员账户设置失败: $SETUP_LOG"
            exit 1
        fi
    fi

    if [ -n "$ENTRANCE" ] || [ -n "$AGENT_TOKEN" ]; then
        log_step "写入安全引导配置..."
        if ! (cd "$INSTALL_DIR" && XPANEL_SECURITY_ENTRANCE="$ENTRANCE" XPANEL_AGENT_TOKEN="$AGENT_TOKEN" ./xpanel bootstrap-config); then
            log_error "安全入口或 Agent Token 写入失败，服务未启动"
            exit 1
        fi
        log_info "安全引导配置已写入本地数据库"
    fi
fi

# ==================== 启动服务 ====================
log_step "启动 X-Panel..."
XPANEL_START_OK=false
if systemctl start $SERVICE_NAME; then
    XPANEL_START_OK=true
else
    log_warn "X-Panel 启动失败，将不会启动捆绑 Nezha Agent"
fi

# 等待启动
sleep 2
if systemctl is-active --quiet $SERVICE_NAME; then
    log_info "X-Panel 启动成功 ✓"
    XPANEL_START_OK=true
else
    if [ "${XPANEL_START_OK}" = true ]; then
        log_warn "X-Panel 可能还在启动中..."
        log_info "请稍后检查: systemctl status $SERVICE_NAME"
    else
        log_warn "X-Panel 未处于 active 状态"
        log_info "请稍后检查: systemctl status $SERVICE_NAME"
    fi
fi

# Nezha Agent enable/start only after X-Panel start path; skipped if start failed.
finalize_nezha_agent_service

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
if [ -n "$AGENT_TOKEN" ]; then
echo ""
echo -e "  ${BOLD}Agent 模式:${NC} ${GREEN}已启用${NC}"
echo -e "  ${BOLD}Agent Token:${NC} ${AGENT_TOKEN}"
echo -e "  ${YELLOW}  在主面板中添加节点时，填写本机地址 ${SERVER_IP}:${PORT} 和上述 Token${NC}"
fi
echo ""
echo -e "  ${BOLD}常用命令:${NC}"
echo "    systemctl start $SERVICE_NAME     # 启动"
echo "    systemctl stop $SERVICE_NAME      # 停止"
echo "    systemctl restart $SERVICE_NAME   # 重启"
echo "    systemctl status $SERVICE_NAME    # 查看状态"
echo "    journalctl -u $SERVICE_NAME -f    # 查看日志"
echo "    xpctl doctor                      # 诊断"
echo "    xpctl backup db                   # 备份面板数据库"
echo "    xpctl recover migrate --apply     # 服务停止后执行迁移"
echo ""
if [ "$IS_UPGRADE" = false ]; then
    echo -e "  ${GREEN}${BOLD}✓ 管理员账户已在服务启动前设置${NC}"
    echo -e "  ${BOLD}用户名:${NC} ${INIT_USERNAME}"
    if [ "$ADMIN_PASSWORD_GENERATED" = true ]; then
        echo -e "  ${BOLD}初始密码:${NC} ${INIT_PASSWORD}"
        echo -e "  ${YELLOW}请立即安全保存，并在首次登录后修改密码${NC}"
    fi
    echo ""
fi
echo -e "  ${BOLD}卸载命令:${NC}"
UNINSTALL_CMD="curl -sSL $DEFAULT_UPDATE_URL/install-online.sh | bash -s -- --uninstall --yes"
if [ "$INSTALL_DIR" != "$DEFAULT_INSTALL_DIR" ]; then
    UNINSTALL_CMD="$UNINSTALL_CMD --path $INSTALL_DIR"
fi
echo "    $UNINSTALL_CMD"
echo ""
