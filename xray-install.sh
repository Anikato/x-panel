#!/usr/bin/env bash

# The files installed by the script conform to the Filesystem Hierarchy Standard:
# https://wiki.linuxfoundation.org/lsb/fhs

# The URL of the script project is:
# https://github.com/XTLS/Xray-install

# The URL of the script is:
# https://github.com/XTLS/Xray-install/raw/main/install-release.sh

# If the script executes incorrectly, go to:
# https://github.com/XTLS/Xray-install/issues

# ==========================================
# 交互式 Xray 安装脚本
# ==========================================
# 
# 使用方法：
# 1. 交互式模式（推荐）：
#    bash xray-install.sh
# 
# 2. 命令行模式：
#    bash xray-install.sh install
#    bash xray-install.sh remove
#    bash xray-install.sh check
#    bash xray-install.sh help
#
# 所有文件将安装到 /data/xray 目录下：
# - /data/xray/bin/xray     (可执行文件)
# - /data/xray/etc/         (配置文件)
# - /data/xray/share/       (数据文件)
# - /data/xray/log/         (日志文件)
# ==========================================

# You can set this variable whatever you want in shell session right before running this script by issuing:
# export DAT_PATH='/data/xray/share'
DAT_PATH=${DAT_PATH:-/data/xray/share}

# You can set this variable whatever you want in shell session right before running this script by issuing:
# export JSON_PATH='/data/xray/etc'
JSON_PATH=${JSON_PATH:-/data/xray/etc}

# Set this variable only if you are starting xray with multiple configuration files:
# export JSONS_PATH='/data/xray/etc'

# Set this variable only if you want this script to check all the systemd unit file:
# export check_all_service_files='yes'

# Gobal verbals

if [[ -f '/etc/systemd/system/xray.service' ]] && [[ -f '/data/xray/bin/xray' ]]; then
  XRAY_IS_INSTALLED_BEFORE_RUNNING_SCRIPT=1
else
  XRAY_IS_INSTALLED_BEFORE_RUNNING_SCRIPT=0
fi

# Xray current version
CURRENT_VERSION=''

# Xray latest release version
RELEASE_LATEST=''

# Xray latest prerelease/release version
PRE_RELEASE_LATEST=''

# Xray version will be installed
INSTALL_VERSION=''

# install
INSTALL='0'

# install-geodata
INSTALL_GEODATA='0'

# remove
REMOVE='0'

# help
HELP='0'

# check
CHECK='0'

# --force
FORCE='0'

# --beta
BETA='0'

# --install-user ?
INSTALL_USER=''

# --without-geodata
NO_GEODATA='0'

# --without-logfiles
NO_LOGFILES='0'

# --logrotate
LOGROTATE='0'

# --no-update-service
N_UP_SERVICE='0'

# --reinstall
REINSTALL='0'

# --version ?
SPECIFIED_VERSION=''

# --local ?
LOCAL_FILE=''

# --proxy ?
PROXY=''

# --purge
PURGE='0'

curl() {
  $(type -P curl) -L -q --retry 5 --retry-delay 10 --retry-max-time 60 "$@"
}

systemd_cat_config() {
  if systemd-analyze --help | grep -qw 'cat-config'; then
    systemd-analyze --no-pager cat-config "$@"
    echo
  else
    echo "${aoi}~~~~~~~~~~~~~~~~"
    cat "$@" "$1".d/*
    echo "${aoi}~~~~~~~~~~~~~~~~"
    echo "${red}warning: ${green}The systemd version on the current operating system is too low."
    echo "${red}warning: ${green}Please consider to upgrade the systemd or the operating system.${reset}"
    echo
  fi
}

check_if_running_as_root() {
  if [[ "$(id -u)" -eq 0 ]]; then
    return 0
  else
    echo "error: You must run this script as root!"
    return 1
  fi
}

identify_the_operating_system_and_architecture() {
  if [[ "$(uname)" != 'Linux' ]]; then
    echo "error: This operating system is not supported."
    return 1
  fi
  case "$(uname -m)" in
  'i386' | 'i686')
    MACHINE='32'
    ;;
  'amd64' | 'x86_64')
    MACHINE='64'
    ;;
  'armv5tel')
    MACHINE='arm32-v5'
    ;;
  'armv6l')
    MACHINE='arm32-v6'
    grep Features /proc/cpuinfo | grep -qw 'vfp' || MACHINE='arm32-v5'
    ;;
  'armv7' | 'armv7l')
    MACHINE='arm32-v7a'
    grep Features /proc/cpuinfo | grep -qw 'vfp' || MACHINE='arm32-v5'
    ;;
  'armv8' | 'aarch64')
    MACHINE='arm64-v8a'
    ;;
  'mips')
    MACHINE='mips32'
    ;;
  'mipsle')
    MACHINE='mips32le'
    ;;
  'mips64')
    MACHINE='mips64'
    lscpu | grep -q "Little Endian" && MACHINE='mips64le'
    ;;
  'mips64le')
    MACHINE='mips64le'
    ;;
  'ppc64')
    MACHINE='ppc64'
    ;;
  'ppc64le')
    MACHINE='ppc64le'
    ;;
  'riscv64')
    MACHINE='riscv64'
    ;;
  's390x')
    MACHINE='s390x'
    ;;
  *)
    echo "error: The architecture is not supported."
    return 1
    ;;
  esac
  if [[ ! -f '/etc/os-release' ]]; then
    echo "error: Don't use outdated Linux distributions."
    return 1
  fi
  # Do not combine this judgment condition with the following judgment condition.
  ## Be aware of Linux distribution like Gentoo, which kernel supports switch between Systemd and OpenRC.
  if [[ -f /.dockerenv ]] || grep -q 'docker\|lxc' /proc/1/cgroup && [[ "$(type -P systemctl)" ]]; then
    true
  elif [[ -d /run/systemd/system ]] || grep -q systemd <(ls -l /sbin/init); then
    true
  else
    echo "error: Only Linux distributions using systemd are supported."
    return 1
  fi
  if [[ "$(type -P apt)" ]]; then
    PACKAGE_MANAGEMENT_INSTALL='apt -y --no-install-recommends install'
    PACKAGE_MANAGEMENT_REMOVE='apt purge'
    package_provide_tput='ncurses-bin'
  elif [[ "$(type -P dnf)" ]]; then
    PACKAGE_MANAGEMENT_INSTALL='dnf -y install'
    PACKAGE_MANAGEMENT_REMOVE='dnf remove'
    package_provide_tput='ncurses'
  elif [[ "$(type -P yum)" ]]; then
    PACKAGE_MANAGEMENT_INSTALL='yum -y install'
    PACKAGE_MANAGEMENT_REMOVE='yum remove'
    package_provide_tput='ncurses'
  elif [[ "$(type -P zypper)" ]]; then
    PACKAGE_MANAGEMENT_INSTALL='zypper install -y --no-recommends'
    PACKAGE_MANAGEMENT_REMOVE='zypper remove'
    package_provide_tput='ncurses-utils'
  elif [[ "$(type -P pacman)" ]]; then
    PACKAGE_MANAGEMENT_INSTALL='pacman -Syy --noconfirm'
    PACKAGE_MANAGEMENT_REMOVE='pacman -Rsn'
    package_provide_tput='ncurses'
  elif [[ "$(type -P emerge)" ]]; then
    PACKAGE_MANAGEMENT_INSTALL='emerge -qv'
    PACKAGE_MANAGEMENT_REMOVE='emerge -Cv'
    package_provide_tput='ncurses'
  else
    echo "error: The script does not support the package manager in this operating system."
    return 1
  fi
}

## Demo function for processing parameters
judgment_parameters() {
  local local_install='0'
  local temp_version='0'
  while [[ "$#" -gt '0' ]]; do
    case "$1" in
    'install')
      INSTALL='1'
      ;;
    'install-geodata')
      INSTALL_GEODATA='1'
      ;;
    'remove')
      REMOVE='1'
      ;;
    'help')
      HELP='1'
      ;;
    'check')
      CHECK='1'
      ;;
    '--without-geodata')
      NO_GEODATA='1'
      ;;
    '--without-logfiles')
      NO_LOGFILES='1'
      ;;
    '--purge')
      PURGE='1'
      ;;
    '--version')
      if [[ -z "$2" ]]; then
        echo "error: Please specify the correct version."
        return 1
      fi
      temp_version='1'
      SPECIFIED_VERSION="$2"
      shift
      ;;
    '-f' | '--force')
      FORCE='1'
      ;;
    '--beta')
      BETA='1'
      ;;
    '-l' | '--local')
      local_install='1'
      if [[ -z "$2" ]]; then
        echo "error: Please specify the correct local file."
        return 1
      fi
      LOCAL_FILE="$2"
      shift
      ;;
    '-p' | '--proxy')
      if [[ -z "$2" ]]; then
        echo "error: Please specify the proxy server address."
        return 1
      fi
      PROXY="$2"
      shift
      ;;
    '-u' | '--install-user')
      if [[ -z "$2" ]]; then
        echo "error: Please specify the install user.}"
        return 1
      fi
      INSTALL_USER="$2"
      shift
      ;;
    '--reinstall')
      REINSTALL='1'
      ;;
    '--no-update-service')
      N_UP_SERVICE='1'
      ;;
    '--logrotate')
      if ! grep -qE '\b([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]\b' <<<"$2"; then
        echo "error: Wrong format of time, it should be in the format of 12:34:56, under 10:00:00 should be start with 0, e.g. 01:23:45."
        exit 1
      fi
      LOGROTATE='1'
      LOGROTATE_TIME="$2"
      shift
      ;;
    *)
      echo "$0: unknown option -- -"
      return 1
      ;;
    esac
    shift
  done
  if ((INSTALL + INSTALL_GEODATA + HELP + CHECK + REMOVE == 0)); then
    INSTALL='1'
  elif ((INSTALL + INSTALL_GEODATA + HELP + CHECK + REMOVE > 1)); then
    echo 'You can only choose one action.'
    return 1
  fi
  if [[ "$INSTALL" -eq '1' ]] && ((temp_version + local_install + REINSTALL + BETA > 1)); then
    echo "--version,--reinstall,--beta and --local can't be used together."
    return 1
  fi
}

check_install_user() {
  if [[ -z "$INSTALL_USER" ]]; then
    if [[ -f '/data/xray/bin/xray' ]]; then
      INSTALL_USER="$(grep '^[ '$'\t]*User[ '$'\t]*=' /etc/systemd/system/xray.service | tail -n 1 | awk -F = '{print $2}' | awk '{print $1}')"
      if [[ -z "$INSTALL_USER" ]]; then
        INSTALL_USER='root'
      fi
    else
      INSTALL_USER='nobody'
    fi
  fi
  if ! id "$INSTALL_USER" >/dev/null 2>&1; then
    echo "the user '$INSTALL_USER' is not effective"
    exit 1
  fi
  INSTALL_USER_UID="$(id -u "$INSTALL_USER")"
  INSTALL_USER_GID="$(id -g "$INSTALL_USER")"
}

install_software() {
  package_name="$1"
  file_to_detect="$2"
  type -P "$file_to_detect" >/dev/null 2>&1 && return
  if ${PACKAGE_MANAGEMENT_INSTALL} "$package_name" >/dev/null 2>&1; then
    echo "info: $package_name is installed."
  else
    echo "error: Installation of $package_name failed, please check your network."
    exit 1
  fi
}

get_current_version() {
  # Get the CURRENT_VERSION
  if [[ -f '/data/xray/bin/xray' ]]; then
    CURRENT_VERSION="$(/data/xray/bin/xray -version | awk 'NR==1 {print $2}')"
    CURRENT_VERSION="v${CURRENT_VERSION#v}"
  else
    CURRENT_VERSION=""
  fi
}

get_latest_version() {
  # Get Xray latest release version number
  local tmp_file
  tmp_file="$(mktemp)"
  local url='https://api.github.com/repos/XTLS/Xray-core/releases/latest'
  if curl -x "${PROXY}" -sSfLo "$tmp_file" -H "Accept: application/vnd.github.v3+json" "$url"; then
    echo "get release list success"
  else
    "rm" "$tmp_file"
    echo 'error: Failed to get release list, please check your network.'
    exit 1
  fi
  RELEASE_LATEST="$(sed 'y/,/\n/' "$tmp_file" | grep 'tag_name' | awk -F '"' '{print $4}')"
  if [[ -z "$RELEASE_LATEST" ]]; then
    if grep -q "API rate limit exceeded" "$tmp_file"; then
      echo "error: github API rate limit exceeded"
    else
      echo "error: Failed to get the latest release version."
      echo "Welcome bug report:https://github.com/XTLS/Xray-install/issues"
    fi
    "rm" "$tmp_file"
    exit 1
  fi
  "rm" "$tmp_file"
  RELEASE_LATEST="v${RELEASE_LATEST#v}"
  url='https://api.github.com/repos/XTLS/Xray-core/releases'
  if curl -x "${PROXY}" -sSfLo "$tmp_file" -H "Accept: application/vnd.github.v3+json" "$url"; then
    echo "get release list success"
  else
    "rm" "$tmp_file"
    echo 'error: Failed to get release list, please check your network.'
    exit 1
  fi
  local releases_list
  readarray -t releases_list < <(sed 'y/,/\n/' "$tmp_file" | grep 'tag_name' | awk -F '"' '{print $4}')
  if [[ "${#releases_list[@]}" -eq 0 ]]; then
    if grep -q "API rate limit exceeded" "$tmp_file"; then
      echo "error: github API rate limit exceeded"
    else
      echo "error: Failed to get the latest release version."
      echo "Welcome bug report:https://github.com/XTLS/Xray-install/issues"
    fi
    "rm" "$tmp_file"
    exit 1
  fi
  local i url_zip
  for i in "${!releases_list[@]}"; do
    releases_list["$i"]="v${releases_list[$i]#v}"
    url_zip="https://github.com/XTLS/Xray-core/releases/download/${releases_list[$i]}/Xray-linux-$MACHINE.zip"
    if grep -q "$url_zip" "$tmp_file"; then
      PRE_RELEASE_LATEST="${releases_list[$i]}"
      break
    fi
  done
  "rm" "$tmp_file"
}

version_gt() {
  test "$(echo -e "$1\\n$2" | sort -V | head -n 1)" != "$1"
}

download_xray() {
  local DOWNLOAD_LINK="https://github.com/XTLS/Xray-core/releases/download/${INSTALL_VERSION}/Xray-linux-${MACHINE}.zip"
  echo "Downloading Xray archive: $DOWNLOAD_LINK"
  if curl -f -x "${PROXY}" -R -H 'Cache-Control: no-cache' -o "$ZIP_FILE" "$DOWNLOAD_LINK"; then
    echo "ok."
  else
    echo 'error: Download failed! Please check your network or try again.'
    return 1
  fi
  echo "Downloading verification file for Xray archive: ${DOWNLOAD_LINK}.dgst"
  if curl -f -x "${PROXY}" -sSR -H 'Cache-Control: no-cache' -o "${ZIP_FILE}.dgst" "${DOWNLOAD_LINK}.dgst"; then
    echo "ok."
  else
    echo 'error: Download failed! Please check your network or try again.'
    return 1
  fi
  if grep 'Not Found' "${ZIP_FILE}.dgst"; then
    echo 'error: This version does not support verification. Please replace with another version.'
    return 1
  fi

  # Verification of Xray archive
  CHECKSUM=$(awk -F '= ' '/256=/ {print $2}' "${ZIP_FILE}.dgst")
  LOCALSUM=$(sha256sum "$ZIP_FILE" | awk '{printf $1}')
  if [[ "$CHECKSUM" != "$LOCALSUM" ]]; then
    echo 'error: SHA256 check failed! Please check your network or try again.'
    return 1
  fi
}

decompression() {
  if ! unzip -q "$1" -d "$TMP_DIRECTORY"; then
    echo 'error: Xray decompression failed.'
    "rm" -r "$TMP_DIRECTORY"
    echo "removed: $TMP_DIRECTORY"
    exit 1
  fi
  echo "info: Extract the Xray package to $TMP_DIRECTORY and prepare it for installation."
}

install_file() {
  NAME="$1"
  if [[ "$NAME" == 'xray' ]]; then
    # Ensure the target directory exists
    if [[ ! -d "/data/xray/bin" ]]; then
      echo "info: 创建目录 /data/xray/bin"
      install -d /data/xray/bin || {
        echo "error: 无法创建目录 /data/xray/bin"
        exit 1
      }
    fi
    
    echo "info: 安装 Xray 二进制文件到 /data/xray/bin/$NAME"
    install -m 755 "${TMP_DIRECTORY}/$NAME" "/data/xray/bin/$NAME" || {
      echo "error: 无法安装 Xray 二进制文件"
      exit 1
    }
    echo "info: Xray 二进制文件安装成功"
    
  elif [[ "$NAME" == 'geoip.dat' ]] || [[ "$NAME" == 'geosite.dat' ]]; then
    echo "info: 安装 $NAME 到 ${DAT_PATH}/$NAME"
    install -m 644 "${TMP_DIRECTORY}/$NAME" "${DAT_PATH}/$NAME" || {
      echo "error: 无法安装 $NAME"
      exit 1
    }
    # Xray 默认在可执行文件目录查找数据文件，所以也需要安装到 /data/xray/bin/
    echo "info: 安装 $NAME 到 /data/xray/bin/$NAME (Xray 默认查找位置)"
    install -m 644 "${TMP_DIRECTORY}/$NAME" "/data/xray/bin/$NAME" || {
      echo "error: 无法安装 $NAME 到 /data/xray/bin/"
      exit 1
    }
    echo "info: $NAME 安装成功"
  fi
}

# Pre-installation checks
pre_install_checks() {
  echo "info: 执行预安装检查..."
  
  # Check if running as root
  if [[ "$(id -u)" -ne 0 ]]; then
    echo "error: 此脚本需要root权限运行"
    exit 1
  fi
  
  # Check if /data directory exists and is writable
  if [[ ! -d "/data" ]]; then
    echo "info: 创建 /data 目录"
    install -d /data || {
      echo "error: 无法创建 /data 目录"
      exit 1
    }
  fi
  
  if [[ ! -w "/data" ]]; then
    echo "error: /data 目录不可写"
    exit 1
  fi
  
  # Check available disk space (at least 100MB)
  local available_space=$(df /data | awk 'NR==2 {print $4}')
  if [[ "$available_space" -lt 102400 ]]; then
    echo "warning: /data 目录可用空间不足 100MB，可能影响安装"
  fi
  
  echo "info: 预安装检查完成"
}

# Function to create directories safely
create_xray_directories() {
  echo "info: 检查并创建必要的目录..."
  
  local dirs=("/data/xray/bin" "/data/xray/etc" "/data/xray/share" "/data/xray/log")
  
  for dir in "${dirs[@]}"; do
    if [[ ! -d "$dir" ]]; then
      echo "info: 创建目录 $dir"
      install -d "$dir" || {
        echo "error: 无法创建目录 $dir"
        exit 1
      }
    else
      echo "info: 目录 $dir 已存在"
    fi
  done
  
  # Set proper permissions for log directory
  chown 0:0 /data/xray/log/ 2>/dev/null || true
  chmod 755 /data/xray/log/ 2>/dev/null || true
}

install_xray() {
  # Run pre-installation checks
  pre_install_checks
  
  # Create necessary directories first
  create_xray_directories
  
  # Install Xray binary to /data/xray/bin/ and $DAT_PATH
  install_file xray
  # If the file exists, geoip.dat and geosite.dat will not be installed or updated
  if [[ "$NO_GEODATA" -eq '0' ]] && [[ ! -f "${DAT_PATH}/.undat" ]]; then
    install -d "$DAT_PATH"
    install_file geoip.dat
    install_file geosite.dat
    GEODATA='1'
  fi

  # Install Xray configuration file to $JSON_PATH
  # shellcheck disable=SC2153
  if [[ -z "$JSONS_PATH" ]]; then
    if [[ ! -f "${JSON_PATH}/config.json" ]]; then
      echo "info: 创建默认配置文件 ${JSON_PATH}/config.json"
      echo "{}" >"${JSON_PATH}/config.json"
      CONFIG_NEW='1'
    else
      echo "info: 配置文件 ${JSON_PATH}/config.json 已存在"
    fi
  fi

  # Install Xray configuration file to $JSONS_PATH
  if [[ -n "$JSONS_PATH" ]] && [[ ! -d "$JSONS_PATH" ]]; then
    install -d "$JSONS_PATH"
    for BASE in 00_log 01_api 02_dns 03_routing 04_policy 05_inbounds 06_outbounds 07_transport 08_stats 09_reverse; do
      echo '{}' >"${JSONS_PATH}/${BASE}.json"
    done
    CONFDIR='1'
  fi

  # Used to store Xray log files
  if [[ "$NO_LOGFILES" -eq '0' ]]; then
    if [[ ! -d '/data/xray/log/' ]]; then
      install -d -m 755 -o 0 -g 0 /data/xray/log/
      install -m 600 -o "$INSTALL_USER_UID" -g "$INSTALL_USER_GID" /dev/null /data/xray/log/access.log
      install -m 600 -o "$INSTALL_USER_UID" -g "$INSTALL_USER_GID" /dev/null /data/xray/log/error.log
      LOG='1'
    else
      chown 0:0 /data/xray/log/
      chmod 755 /data/xray/log/
      chown "$INSTALL_USER_UID:$INSTALL_USER_GID" /data/xray/log/*.log
      chmod 600 /data/xray/log/*.log
    fi
  fi
}

install_startup_service_file() {
  mkdir -p '/etc/systemd/system/xray.service.d'
  mkdir -p '/etc/systemd/system/xray@.service.d/'
  local temp_CapabilityBoundingSet="CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE"
  local temp_AmbientCapabilities="AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE"
  local temp_NoNewPrivileges="NoNewPrivileges=true"
  if [[ "$INSTALL_USER_UID" -eq '0' ]]; then
    temp_CapabilityBoundingSet="#${temp_CapabilityBoundingSet}"
    temp_AmbientCapabilities="#${temp_AmbientCapabilities}"
    temp_NoNewPrivileges="#${temp_NoNewPrivileges}"
  fi
  cat >/etc/systemd/system/xray.service <<EOF
[Unit]
Description=Xray Service
Documentation=https://github.com/xtls
After=network.target nss-lookup.target

[Service]
User=$INSTALL_USER
${temp_CapabilityBoundingSet}
${temp_AmbientCapabilities}
${temp_NoNewPrivileges}
ExecStart=/data/xray/bin/xray run -config /data/xray/etc/config.json
Restart=on-failure
RestartPreventExitStatus=23
LimitNPROC=10000
LimitNOFILE=1000000

[Install]
WantedBy=multi-user.target
EOF
  cat >/etc/systemd/system/xray@.service <<EOF
[Unit]
Description=Xray Service
Documentation=https://github.com/xtls
After=network.target nss-lookup.target

[Service]
User=$INSTALL_USER
${temp_CapabilityBoundingSet}
${temp_AmbientCapabilities}
${temp_NoNewPrivileges}
ExecStart=/data/xray/bin/xray run -config /data/xray/etc/%i.json
Restart=on-failure
RestartPreventExitStatus=23
LimitNPROC=10000
LimitNOFILE=1000000

[Install]
WantedBy=multi-user.target
EOF
  chmod 644 /etc/systemd/system/xray.service /etc/systemd/system/xray@.service
  if [[ -n "$JSONS_PATH" ]]; then
    "rm" '/etc/systemd/system/xray.service.d/10-donot_touch_single_conf.conf' \
      '/etc/systemd/system/xray@.service.d/10-donot_touch_single_conf.conf'
    echo "# In case you have a good reason to do so, duplicate this file in the same directory and make your customizes there.
# Or all changes you made will be lost!  # Refer: https://www.freedesktop.org/software/systemd/man/systemd.unit.html
[Service]
ExecStart=
ExecStart=/data/xray/bin/xray run -confdir $JSONS_PATH" |
      tee '/etc/systemd/system/xray.service.d/10-donot_touch_multi_conf.conf' > \
        '/etc/systemd/system/xray@.service.d/10-donot_touch_multi_conf.conf'
  else
    "rm" '/etc/systemd/system/xray.service.d/10-donot_touch_multi_conf.conf' \
      '/etc/systemd/system/xray@.service.d/10-donot_touch_multi_conf.conf'
    echo "# In case you have a good reason to do so, duplicate this file in the same directory and make your customizes there.
# Or all changes you made will be lost!  # Refer: https://www.freedesktop.org/software/systemd/man/systemd.unit.html
[Service]
ExecStart=
ExecStart=/data/xray/bin/xray run -config ${JSON_PATH}/config.json" > \
      '/etc/systemd/system/xray.service.d/10-donot_touch_single_conf.conf'
    echo "# In case you have a good reason to do so, duplicate this file in the same directory and make your customizes there.
# Or all changes you made will be lost!  # Refer: https://www.freedesktop.org/software/systemd/man/systemd.unit.html
[Service]
ExecStart=
ExecStart=/data/xray/bin/xray run -config ${JSON_PATH}/%i.json" > \
      '/etc/systemd/system/xray@.service.d/10-donot_touch_single_conf.conf'
  fi
  echo "info: Systemd service files have been installed successfully!"
  echo "${red}warning: ${green}The following are the actual parameters for the xray service startup."
  echo "${red}warning: ${green}Please make sure the configuration file path is correctly set.${reset}"
  systemd_cat_config /etc/systemd/system/xray.service
  # shellcheck disable=SC2154
  if [[ "${check_all_service_files:0:1}" = 'y' ]]; then
    echo
    echo
    systemd_cat_config /etc/systemd/system/xray@.service
  fi
  systemctl daemon-reload
  SYSTEMD='1'
}

start_xray() {
  if [[ -f '/etc/systemd/system/xray.service' ]]; then
    systemctl start "${XRAY_CUSTOMIZE:-xray}"
    sleep 1s
    if systemctl -q is-active "${XRAY_CUSTOMIZE:-xray}"; then
      echo 'info: Start the Xray service.'
    else
      echo 'error: Failed to start Xray service.'
      exit 1
    fi
  fi
}

stop_xray() {
  XRAY_CUSTOMIZE="$(systemctl list-units | grep 'xray@' | awk -F ' ' '{print $1}')"
  if [[ -z "$XRAY_CUSTOMIZE" ]]; then
    local xray_daemon_to_stop='xray.service'
  else
    local xray_daemon_to_stop="$XRAY_CUSTOMIZE"
  fi
  if ! systemctl stop "$xray_daemon_to_stop"; then
    echo 'error: Stopping the Xray service failed.'
    exit 1
  fi
  echo 'info: Stop the Xray service.'
}

install_with_logrotate() {
  install_software 'logrotate' 'logrotate'
  if [[ -z "$LOGROTATE_TIME" ]]; then
    LOGROTATE_TIME="00:00:00"
  fi
  cat <<EOF >/etc/systemd/system/logrotate@.service
[Unit]
Description=Rotate log files
Documentation=man:logrotate(8)

[Service]
Type=oneshot
ExecStart=/usr/sbin/logrotate /etc/logrotate.d/%i
EOF
  cat <<EOF >/etc/systemd/system/logrotate@.timer
[Unit]
Description=Run logrotate for %i logs

[Timer]
OnCalendar=*-*-* $LOGROTATE_TIME
Persistent=true

[Install]
WantedBy=timers.target
EOF
  if [[ ! -d '/etc/logrotate.d/' ]]; then
    install -d -m 700 -o "$INSTALL_USER_UID" -g "$INSTALL_USER_GID" /etc/logrotate.d/
    LOGROTATE_DIR='1'
  fi
  cat <<EOF >/etc/logrotate.d/xray
/data/xray/log/*.log {
    daily
    missingok
    rotate 7
    compress
    delaycompress
    notifempty
    create 0600 $INSTALL_USER_UID $INSTALL_USER_GID
}
EOF
  LOGROTATE_FIN='1'
}

install_geodata() {
  download_geodata() {
    if ! curl -x "${PROXY}" -R -H 'Cache-Control: no-cache' -o "${dir_tmp}/${2}" "${1}"; then
      echo 'error: Download failed! Please check your network or try again.'
      exit 1
    fi
    if ! curl -x "${PROXY}" -R -H 'Cache-Control: no-cache' -o "${dir_tmp}/${2}.sha256sum" "${1}.sha256sum"; then
      echo 'error: Download failed! Please check your network or try again.'
      exit 1
    fi
  }
  local download_link_geoip="https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat"
  local download_link_geosite="https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat"
  local file_ip='geoip.dat'
  local file_dlc='geosite.dat'
  local file_site='geosite.dat'
  local dir_tmp
  dir_tmp="$(mktemp -d)"
  [[ "$XRAY_IS_INSTALLED_BEFORE_RUNNING_SCRIPT" -eq '0' ]] && echo "warning: Xray was not installed"
  download_geodata $download_link_geoip $file_ip
  download_geodata $download_link_geosite $file_dlc
  cd "${dir_tmp}" || exit
  for i in "${dir_tmp}"/*.sha256sum; do
    if ! sha256sum -c "${i}"; then
      echo 'error: Check failed! Please check your network or try again.'
      exit 1
    fi
  done
  cd - >/dev/null || exit 1
  install -d "$DAT_PATH"
  install -m 644 "${dir_tmp}"/${file_dlc} "${DAT_PATH}"/${file_site}
  install -m 644 "${dir_tmp}"/${file_ip} "${DAT_PATH}"/${file_ip}
  # Xray 默认在可执行文件目录查找数据文件，所以也需要安装到 /data/xray/bin/
  install -m 644 "${dir_tmp}"/${file_dlc} "/data/xray/bin/${file_site}"
  install -m 644 "${dir_tmp}"/${file_ip} "/data/xray/bin/${file_ip}"
  rm -r "${dir_tmp}"
  exit 0
}

check_update() {
  if [[ "$XRAY_IS_INSTALLED_BEFORE_RUNNING_SCRIPT" -eq '1' ]]; then
    get_current_version
    echo "info: The current version of Xray is ${CURRENT_VERSION}."
  else
    echo 'warning: Xray is not installed.'
  fi
  get_latest_version
  echo "info: The latest release version of Xray is ${RELEASE_LATEST}."
  echo "info: The latest pre-release/release version of Xray is ${PRE_RELEASE_LATEST}."
  exit 0
}

remove_xray() {
  if systemctl list-unit-files | grep -qw 'xray'; then
    if [[ -n "$(pidof xray)" ]]; then
      stop_xray
    fi
    # Define files and directories to delete
    local delete_files=()
    local delete_dirs=()
    
    # Always delete these files
    delete_files+=('/data/xray/bin/xray')
    delete_files+=('/etc/systemd/system/xray.service')
    delete_files+=('/etc/systemd/system/xray@.service')
    delete_files+=('/etc/logrotate.d/xray')
    
    # Always delete these directories
    delete_dirs+=('/etc/systemd/system/xray.service.d')
    delete_dirs+=('/etc/systemd/system/xray@.service.d')
    
    if [[ "$PURGE" -eq '1' ]]; then
      # For complete removal, delete the entire /data/xray directory
      delete_dirs+=('/data/xray')
      delete_files+=('/etc/systemd/system/logrotate@.service')
      delete_files+=('/etc/systemd/system/logrotate@.timer')
    else
      # For partial removal, delete specific subdirectories
      [[ -d "$DAT_PATH" ]] && delete_dirs+=("$DAT_PATH")
      [[ -d '/data/xray/log' ]] && delete_dirs+=('/data/xray/log')
      if [[ -z "$JSONS_PATH" ]]; then
        [[ -d "$JSON_PATH" ]] && delete_dirs+=("$JSON_PATH")
      else
        [[ -d "$JSONS_PATH" ]] && delete_dirs+=("$JSONS_PATH")
      fi
    fi
    systemctl disable xray
    if [[ -f '/etc/systemd/system/logrotate@.timer' ]]; then
      if ! systemctl stop logrotate@xray.timer && systemctl disable logrotate@xray.timer; then
        echo 'error: Stopping and disabling the logrotate service failed.'
        exit 1
      fi
      echo 'info: Stop and disable the logrotate service.'
    fi
    # Delete files first
    for file in "${delete_files[@]}"; do
      if [[ -f "$file" ]]; then
        if rm -f "$file"; then
          echo "removed: $file"
        else
          echo "warning: Failed to remove file $file"
        fi
      fi
    done
    
    # Delete directories
    for dir in "${delete_dirs[@]}"; do
      if [[ -d "$dir" ]]; then
        if rm -rf "$dir"; then
          echo "removed: $dir"
        else
          echo "warning: Failed to remove directory $dir"
        fi
      fi
    done
    
    systemctl daemon-reload
    echo "You may need to execute a command to remove dependent software: $PACKAGE_MANAGEMENT_REMOVE curl unzip"
    echo 'info: Xray has been removed.'
    if [[ "$PURGE" -eq '0' ]]; then
      echo 'info: If necessary, manually delete the configuration and log files.'
      if [[ -n "$JSONS_PATH" ]]; then
        echo "info: e.g., $JSONS_PATH and /data/xray/log/ ..."
      else
        echo "info: e.g., $JSON_PATH and /data/xray/log/ ..."
      fi
    fi
    exit 0
  else
    echo 'error: Xray is not installed.'
    exit 1
  fi
}

# Interactive menu function
show_interactive_menu() {
  echo "=========================================="
  echo "    Xray 安装脚本 - 交互式菜单"
  echo "=========================================="
  echo
  echo "请选择要执行的操作："
  echo
  echo "1) 快速安装 Xray (使用默认设置)"
  echo "2) 自定义安装 Xray"
  echo "3) 仅安装/更新地理数据 (geoip.dat, geosite.dat)"
  echo "4) 卸载 Xray"
  echo "5) 检查更新"
  echo "6) 显示帮助信息"
  echo "7) 退出"
  echo
  echo -n "请输入选项 (1-7): "
}

# Interactive parameter selection
get_interactive_parameters() {
  show_interactive_menu
  read -r choice
  
  case "$choice" in
    1)
      INSTALL='1'
      echo "将使用默认设置快速安装最新稳定版 Xray"
      ;;
    2)
      INSTALL='1'
      echo
      echo "安装选项："
      echo "1) 安装最新稳定版"
      echo "2) 安装指定版本"
      echo "3) 安装预发布版"
      echo "4) 从本地文件安装"
      echo "5) 重新安装当前版本"
      echo
      echo -n "请选择安装方式 (1-5): "
      read -r install_choice
      
      case "$install_choice" in
        1)
          echo "将安装最新稳定版"
          ;;
        2)
          echo -n "请输入版本号 (例如: v1.8.0): "
          read -r version
          SPECIFIED_VERSION="$version"
          ;;
        3)
          BETA='1'
          echo "将安装预发布版"
          ;;
        4)
          echo -n "请输入本地文件路径: "
          read -r local_file
          LOCAL_FILE="$local_file"
          ;;
        5)
          REINSTALL='1'
          echo "将重新安装当前版本"
          ;;
        *)
          echo "无效选择，将安装最新稳定版"
          ;;
      esac
      
      echo
      echo "其他选项："
      echo "1) 不安装地理数据"
      echo "2) 不创建日志文件"
      echo "3) 不更新服务文件"
      echo "4) 强制安装"
      echo "5) 跳过所有选项"
      echo
      echo -n "请选择额外选项 (1-5): "
      read -r extra_choice
      
      case "$extra_choice" in
        1) NO_GEODATA='1' ;;
        2) NO_LOGFILES='1' ;;
        3) N_UP_SERVICE='1' ;;
        4) FORCE='1' ;;
        5) ;;
        *) ;;
      esac
      
      echo
      echo "高级选项："
      echo "1) 设置代理服务器"
      echo "2) 指定运行用户"
      echo "3) 跳过高级选项"
      echo
      echo -n "请选择高级选项 (1-3): "
      read -r advanced_choice
      
      case "$advanced_choice" in
        1)
          echo -n "请输入代理服务器地址 (例如: http://127.0.0.1:8080 或 socks5://127.0.0.1:1080): "
          read -r proxy
          PROXY="$proxy"
          ;;
        2)
          echo -n "请输入运行用户 (默认: nobody): "
          read -r user
          if [[ -n "$user" ]]; then
            INSTALL_USER="$user"
          fi
          ;;
        3) ;;
        *) ;;
      esac
      ;;
    3)
      INSTALL_GEODATA='1'
      echo "将仅安装/更新地理数据"
      ;;
    4)
      REMOVE='1'
      echo
      echo "卸载选项："
      echo "1) 仅卸载程序文件"
      echo "2) 完全卸载 (包括配置和日志)"
      echo
      echo -n "请选择卸载方式 (1-2): "
      read -r remove_choice
      
      case "$remove_choice" in
        2) PURGE='1' ;;
        *) ;;
      esac
      ;;
    5)
      CHECK='1'
      echo "将检查更新"
      ;;
    6)
      HELP='1'
      echo "将显示帮助信息"
      ;;
    7)
      echo "退出脚本"
      exit 0
      ;;
    *)
      echo "无效选择，将执行默认安装"
      INSTALL='1'
      ;;
  esac
  
  # Show confirmation
  echo
  echo "=========================================="
  echo "确认操作："
  echo "=========================================="
  
  if [[ "$INSTALL" -eq '1' ]]; then
    echo "操作: 安装/更新 Xray"
    if [[ -n "$SPECIFIED_VERSION" ]]; then
      echo "版本: $SPECIFIED_VERSION"
    elif [[ "$BETA" -eq '1' ]]; then
      echo "版本: 预发布版"
    elif [[ "$REINSTALL" -eq '1' ]]; then
      echo "版本: 重新安装当前版本"
    else
      echo "版本: 最新稳定版"
    fi
    
    if [[ -n "$LOCAL_FILE" ]]; then
      echo "本地文件: $LOCAL_FILE"
    fi
    
    if [[ "$NO_GEODATA" -eq '1' ]]; then
      echo "选项: 不安装地理数据"
    fi
    if [[ "$NO_LOGFILES" -eq '1' ]]; then
      echo "选项: 不创建日志文件"
    fi
    if [[ "$FORCE" -eq '1' ]]; then
      echo "选项: 强制安装"
    fi
    if [[ -n "$PROXY" ]]; then
      echo "代理: $PROXY"
    fi
    if [[ -n "$INSTALL_USER" ]]; then
      echo "运行用户: $INSTALL_USER"
    fi
    
  elif [[ "$INSTALL_GEODATA" -eq '1' ]]; then
    echo "操作: 仅安装/更新地理数据"
  elif [[ "$REMOVE" -eq '1' ]]; then
    echo "操作: 卸载 Xray"
    if [[ "$PURGE" -eq '1' ]]; then
      echo "选项: 完全卸载 (包括配置和日志)"
    else
      echo "选项: 仅卸载程序文件"
    fi
  elif [[ "$CHECK" -eq '1' ]]; then
    echo "操作: 检查更新"
  elif [[ "$HELP" -eq '1' ]]; then
    echo "操作: 显示帮助信息"
  fi
  
  echo
  echo -n "确认执行以上操作? (y/N): "
  read -r confirm
  
  if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "操作已取消"
    exit 0
  fi
  
  echo
  echo "开始执行操作..."
  echo
}

# Explanation of parameters in the script
show_help() {
  echo "usage: $0 ACTION [OPTION]..."
  echo
  echo 'ACTION:'
  echo '  install                   Install/Update Xray'
  echo '  install-geodata           Install/Update geoip.dat and geosite.dat only'
  echo '  remove                    Remove Xray'
  echo '  help                      Show help'
  echo '  check                     Check if Xray can be updated'
  echo 'If no action is specified, then install will be selected'
  echo
  echo 'OPTION:'
  echo '  install:'
  echo '    --version                 Install the specified version of Xray, e.g., --version v1.0.0'
  echo '    -f, --force               Force install even though the versions are same'
  echo '    --beta                    Install the pre-release version if it is exist'
  echo '    -l, --local               Install Xray from a local file'
  echo '    -p, --proxy               Download through a proxy server, e.g., -p http://127.0.0.1:8118 or -p socks5://127.0.0.1:1080'
  echo '    -u, --install-user        Install Xray in specified user, e.g, -u root'
  echo '    --reinstall               Reinstall current Xray version'
  echo "    --no-update-service       Don't change service files if they are exist"
  echo "    --without-geodata         Don't install/update geoip.dat and geosite.dat"
  echo "    --without-logfiles        Don't install /data/xray/log"
  echo "    --logrotate [time]        Install with logrotate."
  echo "                              [time] need be in the format of 12:34:56, under 10:00:00 should be start with 0, e.g. 01:23:45."
  echo '  install-geodata:'
  echo '    -p, --proxy               Download through a proxy server'
  echo '  remove:'
  echo '    --purge                   Remove all the Xray files, include logs, configs, etc'
  echo '  check:'
  echo '    -p, --proxy               Check new version through a proxy server'
  exit 0
}

main() {
  check_if_running_as_root || return 1
  identify_the_operating_system_and_architecture || return 1
  
  # Check if running in interactive mode (no command line arguments)
  if [[ $# -eq 0 ]]; then
    echo "检测到交互式模式，启动菜单..."
    echo
    get_interactive_parameters
  else
    # Use command line parameters
    judgment_parameters "$@" || return 1
  fi

  install_software "$package_provide_tput" 'tput'
  red=$(tput setaf 1)
  green=$(tput setaf 2)
  aoi=$(tput setaf 6)
  reset=$(tput sgr0)

  # Parameter information
  [[ "$HELP" -eq '1' ]] && show_help
  [[ "$CHECK" -eq '1' ]] && check_update
  [[ "$REMOVE" -eq '1' ]] && remove_xray
  [[ "$INSTALL_GEODATA" -eq '1' ]] && install_geodata

  # Check if the user is effective
  check_install_user

  # Check Logrotate after Check User
  [[ "$LOGROTATE" -eq '1' ]] && install_with_logrotate

  # Two very important variables
  TMP_DIRECTORY="$(mktemp -d)"
  ZIP_FILE="${TMP_DIRECTORY}/Xray-linux-$MACHINE.zip"

  # Install Xray from a local file, but still need to make sure the network is available
  if [[ -n "$LOCAL_FILE" ]]; then
    echo 'warn: Install Xray from a local file, but still need to make sure the network is available.'
    echo -n 'warn: Please make sure the file is valid because we cannot confirm it. (Press any key) ...'
    read -r
    install_software 'unzip' 'unzip'
    decompression "$LOCAL_FILE"
  else
    get_current_version
    if [[ "$REINSTALL" -eq '1' ]]; then
      if [[ -z "$CURRENT_VERSION" ]]; then
        echo "error: Xray is not installed"
        exit 1
      fi
      INSTALL_VERSION="$CURRENT_VERSION"
      echo "info: Reinstalling Xray $CURRENT_VERSION"
    elif [[ -n "$SPECIFIED_VERSION" ]]; then
      SPECIFIED_VERSION="v${SPECIFIED_VERSION#v}"
      if [[ "$CURRENT_VERSION" == "$SPECIFIED_VERSION" ]] && [[ "$FORCE" -eq '0' ]]; then
        echo "info: The current version is same as the specified version. The version is ${CURRENT_VERSION}."
        exit 0
      fi
      INSTALL_VERSION="$SPECIFIED_VERSION"
      echo "info: Installing specified Xray version $INSTALL_VERSION for $(uname -m)"
    else
      install_software 'curl' 'curl'
      get_latest_version
      if [[ "$BETA" -eq '0' ]]; then
        INSTALL_VERSION="$RELEASE_LATEST"
      else
        INSTALL_VERSION="$PRE_RELEASE_LATEST"
      fi
      if ! version_gt "$INSTALL_VERSION" "$CURRENT_VERSION" && [[ "$FORCE" -eq '0' ]]; then
        echo "info: No new version. The current version of Xray is ${CURRENT_VERSION}."
        exit 0
      fi
      echo "info: Installing Xray $INSTALL_VERSION for $(uname -m)"
    fi
    install_software 'curl' 'curl'
    install_software 'unzip' 'unzip'
    if ! download_xray; then
      "rm" -r "$TMP_DIRECTORY"
      echo "removed: $TMP_DIRECTORY"
      exit 1
    fi
    decompression "$ZIP_FILE"
  fi

  # Determine if Xray is running
  if systemctl list-unit-files | grep -qw 'xray'; then
    if [[ -n "$(pidof xray)" ]]; then
      stop_xray
      XRAY_RUNNING='1'
    fi
  fi
  install_xray
  [[ "$N_UP_SERVICE" -eq '1' && -f '/etc/systemd/system/xray.service' ]] || install_startup_service_file
  echo 'installed: /data/xray/bin/xray'
  # If the file exists, the content output of installing or updating geoip.dat and geosite.dat will not be displayed
  if [[ "$GEODATA" -eq '1' ]]; then
    echo "installed: ${DAT_PATH}/geoip.dat"
    echo "installed: ${DAT_PATH}/geosite.dat"
  fi
  if [[ "$CONFIG_NEW" -eq '1' ]]; then
    echo "installed: ${JSON_PATH}/config.json"
  fi
  if [[ "$CONFDIR" -eq '1' ]]; then
    echo "installed: ${JSON_PATH}/00_log.json"
    echo "installed: ${JSON_PATH}/01_api.json"
    echo "installed: ${JSON_PATH}/02_dns.json"
    echo "installed: ${JSON_PATH}/03_routing.json"
    echo "installed: ${JSON_PATH}/04_policy.json"
    echo "installed: ${JSON_PATH}/05_inbounds.json"
    echo "installed: ${JSON_PATH}/06_outbounds.json"
    echo "installed: ${JSON_PATH}/07_transport.json"
    echo "installed: ${JSON_PATH}/08_stats.json"
    echo "installed: ${JSON_PATH}/09_reverse.json"
  fi
  if [[ "$LOG" -eq '1' ]]; then
    echo 'installed: /data/xray/log/'
    echo 'installed: /data/xray/log/access.log'
    echo 'installed: /data/xray/log/error.log'
  fi
  if [[ "$LOGROTATE_FIN" -eq '1' ]]; then
    echo 'installed: /etc/systemd/system/logrotate@.service'
    echo 'installed: /etc/systemd/system/logrotate@.timer'
    if [[ "$LOGROTATE_DIR" -eq '1' ]]; then
      echo 'installed: /etc/logrotate.d/'
    fi
    echo 'installed: /etc/logrotate.d/xray'
    systemctl start logrotate@xray.timer
    systemctl enable logrotate@xray.timer
    sleep 1s
    if systemctl -q is-active logrotate@xray.timer; then
      echo "info: Enable and start the logrotate@xray.timer service"
    else
      echo "warning: Failed to enable and start the logrotate@xray.timer service"
    fi
  fi
  if [[ "$SYSTEMD" -eq '1' ]]; then
    echo 'installed: /etc/systemd/system/xray.service'
    echo 'installed: /etc/systemd/system/xray@.service'
  fi
  "rm" -r "$TMP_DIRECTORY"
  echo "removed: $TMP_DIRECTORY"
  get_current_version
  echo "info: Xray $CURRENT_VERSION is installed."
  echo "You may need to execute a command to remove dependent software: $PACKAGE_MANAGEMENT_REMOVE curl unzip"
  if [[ "$XRAY_IS_INSTALLED_BEFORE_RUNNING_SCRIPT" -eq '1' ]] && [[ "$FORCE" -eq '0' ]] && [[ "$REINSTALL" -eq '0' ]]; then
    [[ "$XRAY_RUNNING" -eq '1' ]] && start_xray
  else
    systemctl start xray
    systemctl enable xray
    sleep 1s
    if systemctl -q is-active xray; then
      echo "info: Enable and start the Xray service"
    else
      echo "warning: Failed to enable and start the Xray service"
    fi
  fi
}

main "$@"