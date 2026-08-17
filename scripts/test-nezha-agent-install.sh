#!/usr/bin/env bash
# Offline fixture tests for bundled Nezha Agent install/upgrade behavior.
# No root, no real systemd, no network, no writes to /opt or /etc.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
INSTALLER="${ROOT_DIR}/scripts/install-online.sh"
OFFLINE_INSTALLER="${ROOT_DIR}/scripts/install.sh"
UNIT_FILE="${ROOT_DIR}/scripts/xpanel-nezha-agent.service"
TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/xpanel-nezha-agent-install.XXXXXX")"
PASS=0
FAIL=0
SECRET_FIXTURE='nzh_test_secret_MUST_NOT_LEAK_7f3a'

cleanup() {
	rm -rf "${TMP_ROOT}"
}
trap cleanup EXIT

pass() {
	printf 'PASS: %s\n' "$*"
	PASS=$((PASS + 1))
}

fail() {
	printf 'FAIL: %s\n' "$*" >&2
	FAIL=$((FAIL + 1))
}

file_mode() {
	local path="$1"
	if stat -c '%a' "${path}" >/dev/null 2>&1; then
		stat -c '%a' "${path}"
	else
		stat -f '%Lp' "${path}"
	fi
}

assert_mode() {
	local path="$1"
	local want="$2"
	local got
	got="$(file_mode "${path}")"
	if [ "${got}" = "${want}" ] || [ "${got}" = "0${want}" ] || [ "0${got}" = "${want}" ]; then
		return 0
	fi
	fail "mode of ${path}: got ${got}, want ${want}"
	return 1
}

assert_no_secret() {
	local label="$1"
	shift
	local f
	for f in "$@"; do
		if [ -f "${f}" ] && grep -Fq "${SECRET_FIXTURE}" "${f}" 2>/dev/null; then
			fail "${label}: secret leaked into ${f}"
			return 1
		fi
	done
	return 0
}

# --- RED gate ---
if [ ! -f "${INSTALLER}" ]; then
	echo "RED: ${INSTALLER} missing"
	exit 1
fi

if ! grep -q 'nezha_dashboard_is_https_origin' "${INSTALLER}" \
	|| ! grep -q 'install_bundled_nezha_agent' "${INSTALLER}" \
	|| ! grep -q 'XPANEL_NEZHA_AGENT_SECRET' "${INSTALLER}"; then
	echo "RED: Nezha installer behavior is absent from install-online.sh (expected before implementation)"
	# Still ensure the harness itself is executable and would exercise the installer.
	if grep -qE -- '--nezha-secret|--nezha-agent-secret|--agent-secret' "${INSTALLER}"; then
		echo "unexpected secret CLI flag already present" >&2
		exit 1
	fi
	echo "test harness confirmed missing Nezha install helpers"
	exit 1
fi

# Load production helpers without executing the installer body.
eval "$(sed -n '/^nezha_dashboard_is_https_origin() {$/,/^}$/p' "${INSTALLER}")"
eval "$(sed -n '/^normalize_nezha_dashboard_server() {$/,/^}$/p' "${INSTALLER}")"
eval "$(sed -n '/^validate_nezha_install_inputs() {$/,/^}$/p' "${INSTALLER}")"
eval "$(sed -n '/^detect_external_nezha_agent() {$/,/^}$/p' "${INSTALLER}")"
eval "$(sed -n '/^capture_nezha_agent_systemd_state() {$/,/^}$/p' "${INSTALLER}")"
eval "$(sed -n '/^stop_nezha_agent_if_active() {$/,/^}$/p' "${INSTALLER}")"
eval "$(sed -n '/^require_nezha_agent_package() {$/,/^}$/p' "${INSTALLER}")"
eval "$(sed -n '/^install_bundled_nezha_agent() {$/,/^}$/p' "${INSTALLER}")"
eval "$(sed -n '/^write_initial_nezha_config() {$/,/^}$/p' "${INSTALLER}")"
eval "$(sed -n '/^finalize_nezha_agent_service() {$/,/^}$/p' "${INSTALLER}")"

# Minimal log stubs used by helpers.
log_info()  { printf '[INFO] %s\n' "$*"; }
log_warn()  { printf '[WARN] %s\n' "$*"; }
log_error() { printf '[ERROR] %s\n' "$*" >&2; }
log_step()  { printf '[STEP] %s\n' "$*"; }

# Fake systemctl driven by per-case state files.
setup_fake_systemctl() {
	local bin_dir="$1"
	local state_dir="$2"
	mkdir -p "${bin_dir}" "${state_dir}"
	cat >"${bin_dir}/systemctl" <<EOF
#!/usr/bin/env bash
set -euo pipefail
STATE_DIR="${state_dir}"
cmd="\${1:-}"
shift || true
printf '%s\n' "\$cmd \$*" >>"\${STATE_DIR}/calls"
unit=""
case "\$cmd" in
  is-active)
    quiet=false
    while [ "\$#" -gt 0 ]; do
      case "\$1" in
        --quiet) quiet=true; shift ;;
        *) unit="\$1"; shift ;;
      esac
    done
    status="\$(cat "\${STATE_DIR}/\${unit}.active" 2>/dev/null || echo inactive)"
    if [ "\$status" = "active" ]; then
      [ "\$quiet" = true ] || echo active
      exit 0
    fi
    [ "\$quiet" = true ] || echo inactive
    exit 1
    ;;
  is-enabled)
    quiet=false
    while [ "\$#" -gt 0 ]; do
      case "\$1" in
        --quiet) quiet=true; shift ;;
        *) unit="\$1"; shift ;;
      esac
    done
    status="\$(cat "\${STATE_DIR}/\${unit}.enabled" 2>/dev/null || echo disabled)"
    if [ "\$status" = "enabled" ]; then
      [ "\$quiet" = true ] || echo enabled
      exit 0
    fi
    [ "\$quiet" = true ] || echo disabled
    exit 1
    ;;
  stop|start|restart|enable|disable|daemon-reload)
    if [ "\$cmd" = "daemon-reload" ]; then
      exit 0
    fi
    # support enable --now / disable --now
    now=false
    unit=""
    for a in "\$@"; do
      if [ "\$a" = "--now" ]; then
        now=true
      else
        unit="\$a"
      fi
    done
    case "\$cmd" in
      stop)
        echo inactive >"\${STATE_DIR}/\${unit}.active"
        ;;
      start)
        echo active >"\${STATE_DIR}/\${unit}.active"
        ;;
      enable)
        echo enabled >"\${STATE_DIR}/\${unit}.enabled"
        if [ "\$now" = true ]; then
          echo active >"\${STATE_DIR}/\${unit}.active"
        fi
        ;;
      disable)
        echo disabled >"\${STATE_DIR}/\${unit}.enabled"
        if [ "\$now" = true ]; then
          echo inactive >"\${STATE_DIR}/\${unit}.active"
        fi
        ;;
      restart)
        echo active >"\${STATE_DIR}/\${unit}.active"
        ;;
    esac
    exit 0
    ;;
  cat|list-units|list-unit-files)
    # External unit presence controlled by state markers.
    if [ -f "\${STATE_DIR}/external.nezha-agent.service" ]; then
      if [ "\$cmd" = "cat" ] && [ "\${1:-}" = "nezha-agent.service" ]; then
        echo '# external unit'
        exit 0
      fi
      if [ "\$cmd" = "list-units" ] || [ "\$cmd" = "list-unit-files" ]; then
        echo "nezha-agent.service loaded active running"
        exit 0
      fi
    fi
    if [ -f "\${STATE_DIR}/external.nezha-agent@node.service" ]; then
      if [ "\$cmd" = "cat" ] && [ "\${1:-}" = "nezha-agent@node.service" ]; then
        echo '# external instance unit'
        exit 0
      fi
      if [ "\$cmd" = "list-units" ] || [ "\$cmd" = "list-unit-files" ]; then
        echo "nezha-agent@node.service loaded active running"
        exit 0
      fi
    fi
    if [ "\$cmd" = "cat" ]; then
      exit 1
    fi
    exit 0
    ;;
  *)
    exit 0
    ;;
esac
EOF
	chmod +x "${bin_dir}/systemctl"
}

make_package_extract() {
	local extract_dir="$1"
	local kind="${2:-ok}"
	mkdir -p "${extract_dir}"
	printf '#!/bin/sh\necho fake-xpanel\n' >"${extract_dir}/xpanel"
	chmod 0755 "${extract_dir}/xpanel"
	case "${kind}" in
	ok)
		mkdir -p "${extract_dir}/nezha-agent"
		printf '#!/bin/sh\necho fake-nezha-agent\n' >"${extract_dir}/nezha-agent/nezha-agent"
		chmod 0755 "${extract_dir}/nezha-agent/nezha-agent"
		cp "${UNIT_FILE}" "${extract_dir}/xpanel-nezha-agent.service"
		;;
	missing-agent)
		mkdir -p "${extract_dir}/nezha-agent"
		printf 'readme\n' >"${extract_dir}/nezha-agent/README"
		;;
	no-dir)
		: # no nezha-agent directory
		;;
	esac
}

# ---------------------------------------------------------------------------
# 1) URL normalization and rejection
# ---------------------------------------------------------------------------
for url in \
	'https://dashboard.example.com' \
	'https://dashboard.example.com:8443' \
	'https://[2001:db8::1]' \
	'https://[2001:db8::1]:8443'; do
	if ! nezha_dashboard_is_https_origin "${url}"; then
		fail "rejected valid origin: ${url}"
	fi
done
if nezha_dashboard_is_https_origin 'https://dashboard.example.com'; then
	server="$(normalize_nezha_dashboard_server 'https://dashboard.example.com')"
	if [ "${server}" = 'dashboard.example.com:443' ]; then
		pass "default HTTPS origin normalizes to host:443"
	else
		fail "normalize default port: got ${server}"
	fi
else
	fail "valid default origin rejected"
fi

server="$(normalize_nezha_dashboard_server 'https://Dashboard.Example.COM:8443')"
if [ "${server}" = 'dashboard.example.com:8443' ]; then
	pass "explicit port + lowercase host normalization"
else
	fail "normalize explicit port: got ${server}"
fi

for bad in \
	'http://dashboard.example.com' \
	'https://user:pass@dashboard.example.com' \
	'https://dashboard.example.com/path' \
	'https://dashboard.example.com?q=1' \
	'https://dashboard.example.com#frag' \
	'https://dashboard.example.com:0' \
	'https://dashboard.example.com:65536' \
	'https://' \
	'ftp://dashboard.example.com'; do
	if nezha_dashboard_is_https_origin "${bad}"; then
		fail "accepted invalid origin: ${bad}"
	fi
done
pass "invalid Dashboard origins rejected"

# ---------------------------------------------------------------------------
# 2) Input pairing / interactive / non-interactive preflight
# ---------------------------------------------------------------------------
NEZHA_DASHBOARD_URL=''
NEZHA_AGENT_SECRET=''
NEZHA_SERVER=''
NEZHA_CONFIGURE=false
if validate_nezha_install_inputs; then
	if [ "${NEZHA_CONFIGURE}" = false ] && [ -z "${NEZHA_AGENT_SECRET}" ]; then
		pass "no credentials: configure skipped"
	else
		fail "no credentials should leave configure=false"
	fi
else
	fail "empty credentials should pass preflight"
fi

NEZHA_DASHBOARD_URL='https://dashboard.example.com'
NEZHA_AGENT_SECRET="${SECRET_FIXTURE}"
NEZHA_SERVER=''
NEZHA_CONFIGURE=false
if validate_nezha_install_inputs \
	&& [ "${NEZHA_CONFIGURE}" = true ] \
	&& [ "${NEZHA_SERVER}" = 'dashboard.example.com:443' ]; then
	pass "paired credentials accepted and normalized"
else
	fail "paired credentials rejected"
fi

# Non-interactive half-pair must fail (no tty).
NEZHA_DASHBOARD_URL='https://dashboard.example.com'
NEZHA_AGENT_SECRET=''
NEZHA_CONFIGURE=false
set +e
validate_nezha_install_inputs </dev/null >/dev/null 2>&1
rc=$?
set -e
if [ "${rc}" -ne 0 ]; then
	pass "non-interactive dashboard-only fails preflight"
else
	fail "non-interactive dashboard-only should fail"
fi

NEZHA_DASHBOARD_URL=''
NEZHA_AGENT_SECRET="${SECRET_FIXTURE}"
set +e
validate_nezha_install_inputs </dev/null >/dev/null 2>&1
rc=$?
set -e
if [ "${rc}" -ne 0 ]; then
	pass "secret-only half-pair fails preflight"
else
	fail "secret-only should fail preflight"
fi

# Half-pair failure must happen before any stop/copy: live markers stay intact.
CASE="$(mktemp -d "${TMP_ROOT}/halfpair.XXXXXX")"
mkdir -p "${CASE}/live" "${CASE}/systemd" "${CASE}/bin" "${CASE}/state"
printf 'live-binary\n' >"${CASE}/live/xpanel"
printf 'live-agent\n' >"${CASE}/live/agent-marker"
setup_fake_systemctl "${CASE}/bin" "${CASE}/state"
echo active >"${CASE}/state/xpanel-nezha-agent.active"
echo enabled >"${CASE}/state/xpanel-nezha-agent.enabled"
export PATH="${CASE}/bin:${PATH}"
INSTALL_DIR="${CASE}/live"
SYSTEMD_DIR="${CASE}/systemd"
NEZHA_DASHBOARD_URL='https://dashboard.example.com'
NEZHA_AGENT_SECRET=''
NEZHA_CONFIGURE=false
NEZHA_WAS_ACTIVE=false
NEZHA_WAS_ENABLED=false
set +e
out="$(validate_nezha_install_inputs 2>&1)"
rc=$?
set -e
if [ "${rc}" -ne 0 ] \
	&& [ ! -f "${CASE}/state/calls" ] \
	&& grep -q 'live-binary' "${CASE}/live/xpanel" \
	&& [ ! -f "${CASE}/live/nezha-agent/nezha-agent" ]; then
	pass "half-pair fails before stop/copy"
else
	fail "half-pair preflight disturbed live state or files"
fi
assert_no_secret "half-pair output" <(printf '%s' "${out}")

# ---------------------------------------------------------------------------
# 3) No secret CLI parameters (static)
# ---------------------------------------------------------------------------
if grep -qE -- '--nezha-secret|--nezha-agent-secret|--client-secret|--agent-secret' "${INSTALLER}" \
	|| grep -qE -- '--nezha-secret|--nezha-agent-secret|--client-secret' "${OFFLINE_INSTALLER}"; then
	fail "installer accepts a secret CLI flag"
else
	pass "no secret CLI parameters"
fi

if ! grep -q -- '--nezha-dashboard' "${INSTALLER}"; then
	fail "install-online.sh missing --nezha-dashboard"
else
	pass "--nezha-dashboard accepted as address-only flag"
fi

# Secret env is read then unset.
if grep -q 'XPANEL_NEZHA_AGENT_SECRET' "${INSTALLER}" \
	&& grep -q 'unset XPANEL_NEZHA_AGENT_SECRET' "${INSTALLER}"; then
	pass "secret env is read and unset"
else
	fail "secret env not cleared after read"
fi

# ---------------------------------------------------------------------------
# 4) Package missing agent fails before replacement
# ---------------------------------------------------------------------------
CASE="$(mktemp -d "${TMP_ROOT}/missing.XXXXXX")"
make_package_extract "${CASE}/extract" missing-agent
mkdir -p "${CASE}/live"
printf 'old-xpanel\n' >"${CASE}/live/xpanel"
printf 'old-agent\n' >"${CASE}/live/old-marker"
set +e
require_nezha_agent_package "${CASE}/extract" >/dev/null 2>&1
rc=$?
set -e
if [ "${rc}" -ne 0 ] && grep -q 'old-xpanel' "${CASE}/live/xpanel" && [ -f "${CASE}/live/old-marker" ]; then
	pass "missing nezha-agent rejected before live replacement"
else
	fail "missing agent package check failed"
fi

CASE="$(mktemp -d "${TMP_ROOT}/nodir.XXXXXX")"
make_package_extract "${CASE}/extract" no-dir
set +e
require_nezha_agent_package "${CASE}/extract" >/dev/null 2>&1
rc=$?
set -e
if [ "${rc}" -ne 0 ]; then
	pass "missing nezha-agent directory rejected"
else
	fail "missing nezha-agent directory was accepted"
fi

# Existing destination links must be rejected before any linked target is changed.
CASE="$(mktemp -d "${TMP_ROOT}/dest-link.XXXXXX")"
make_package_extract "${CASE}/extract" ok
mkdir -p "${CASE}/opt/xpanel" "${CASE}/external-agent" "${CASE}/systemd"
printf 'external-agent-must-stay\n' >"${CASE}/external-agent/nezha-agent"
ln -s "${CASE}/external-agent" "${CASE}/opt/xpanel/nezha-agent"
INSTALL_DIR="${CASE}/opt/xpanel"
SYSTEMD_DIR="${CASE}/systemd"
NEZHA_PACKAGE_DIR="${CASE}/extract"
NEZHA_UNIT_SRC="${CASE}/extract/xpanel-nezha-agent.service"
NEZHA_CONFIGURE=false
set +e
install_bundled_nezha_agent >/dev/null 2>&1
rc=$?
set -e
if [ "${rc}" -ne 0 ] && grep -q 'external-agent-must-stay' "${CASE}/external-agent/nezha-agent"; then
	pass "symlinked Agent destination directory rejected before overwrite"
else
	fail "symlinked Agent destination directory was followed or accepted"
fi

CASE="$(mktemp -d "${TMP_ROOT}/unit-link.XXXXXX")"
make_package_extract "${CASE}/extract" ok
mkdir -p "${CASE}/opt/xpanel/nezha-agent" "${CASE}/systemd" "${CASE}/external"
printf 'old-agent-must-stay\n' >"${CASE}/opt/xpanel/nezha-agent/nezha-agent"
printf 'external-unit-must-stay\n' >"${CASE}/external/unit"
ln -s "${CASE}/external/unit" "${CASE}/systemd/xpanel-nezha-agent.service"
INSTALL_DIR="${CASE}/opt/xpanel"
SYSTEMD_DIR="${CASE}/systemd"
NEZHA_PACKAGE_DIR="${CASE}/extract"
NEZHA_UNIT_SRC="${CASE}/extract/xpanel-nezha-agent.service"
NEZHA_CONFIGURE=false
set +e
install_bundled_nezha_agent >/dev/null 2>&1
rc=$?
set -e
if [ "${rc}" -ne 0 ] \
	&& grep -q 'old-agent-must-stay' "${CASE}/opt/xpanel/nezha-agent/nezha-agent" \
	&& grep -q 'external-unit-must-stay' "${CASE}/external/unit"; then
	pass "symlinked unit destination rejected before binary/unit overwrite"
else
	fail "symlinked unit destination caused a partial or linked overwrite"
fi

# ---------------------------------------------------------------------------
# 5) No-credential install: assets/unit, disabled/inactive, no config
# ---------------------------------------------------------------------------
CASE="$(mktemp -d "${TMP_ROOT}/nocred.XXXXXX")"
make_package_extract "${CASE}/extract" ok
mkdir -p "${CASE}/systemd" "${CASE}/bin" "${CASE}/state"
setup_fake_systemctl "${CASE}/bin" "${CASE}/state"
echo inactive >"${CASE}/state/xpanel-nezha-agent.active"
echo disabled >"${CASE}/state/xpanel-nezha-agent.enabled"
export PATH="${CASE}/bin:${PATH}"
INSTALL_DIR="${CASE}/opt/xpanel"
SYSTEMD_DIR="${CASE}/systemd"
NEZHA_UNIT_SRC="${CASE}/extract/xpanel-nezha-agent.service"
NEZHA_PACKAGE_DIR="${CASE}/extract"
NEZHA_CONFIGURE=false
NEZHA_SERVER=''
NEZHA_AGENT_SECRET=''
NEZHA_EXTERNAL_CONFLICT=false
NEZHA_WAS_ACTIVE=false
NEZHA_WAS_ENABLED=false
IS_UPGRADE=false
mkdir -p "${INSTALL_DIR}"
install_bundled_nezha_agent
finalize_nezha_agent_service

agent_bin="${INSTALL_DIR}/nezha-agent/nezha-agent"
unit_dst="${SYSTEMD_DIR}/xpanel-nezha-agent.service"
config_path="${INSTALL_DIR}/nezha-agent/config.yml"
if [ -f "${agent_bin}" ] && [ -f "${unit_dst}" ] && [ ! -e "${config_path}" ]; then
	assert_mode "${INSTALL_DIR}/nezha-agent" "700" || true
	assert_mode "${agent_bin}" "755" || true
	assert_mode "${unit_dst}" "644" || true
	active="$(cat "${CASE}/state/xpanel-nezha-agent.active")"
	enabled="$(cat "${CASE}/state/xpanel-nezha-agent.enabled")"
	if [ "${active}" = "inactive" ] && [ "${enabled}" = "disabled" ]; then
		pass "no-credential install: assets present, disabled/inactive, no config"
	else
		fail "no-credential install wrong systemd state active=${active} enabled=${enabled}"
	fi
else
	fail "no-credential install missing assets or unexpected config"
fi

# ---------------------------------------------------------------------------
# 6) Paired env credentials: initial YAML, modes, atomic config, enable after panel
# ---------------------------------------------------------------------------
CASE="$(mktemp -d "${TMP_ROOT}/paired.XXXXXX")"
make_package_extract "${CASE}/extract" ok
mkdir -p "${CASE}/systemd" "${CASE}/bin" "${CASE}/state"
setup_fake_systemctl "${CASE}/bin" "${CASE}/state"
echo inactive >"${CASE}/state/xpanel-nezha-agent.active"
echo disabled >"${CASE}/state/xpanel-nezha-agent.enabled"
echo inactive >"${CASE}/state/xpanel.active"
echo disabled >"${CASE}/state/xpanel.enabled"
export PATH="${CASE}/bin:${PATH}"
INSTALL_DIR="${CASE}/opt/xpanel"
SYSTEMD_DIR="${CASE}/systemd"
NEZHA_UNIT_SRC="${CASE}/extract/xpanel-nezha-agent.service"
NEZHA_PACKAGE_DIR="${CASE}/extract"
NEZHA_DASHBOARD_URL='https://dashboard.example.com:8443'
NEZHA_AGENT_SECRET="${SECRET_FIXTURE}"
NEZHA_SERVER=''
NEZHA_CONFIGURE=false
NEZHA_EXTERNAL_CONFLICT=false
NEZHA_WAS_ACTIVE=false
NEZHA_WAS_ENABLED=false
IS_UPGRADE=false
validate_nezha_install_inputs
mkdir -p "${INSTALL_DIR}"
# Record that X-Panel start happens before agent enable/start by calling finalize only after a fake panel start.
install_bundled_nezha_agent
if [ -f "${INSTALL_DIR}/nezha-agent/config.yml" ]; then
	# Config may be written at install time; agent must not be started yet.
	active="$(cat "${CASE}/state/xpanel-nezha-agent.active")"
	enabled="$(cat "${CASE}/state/xpanel-nezha-agent.enabled")"
	if [ "${active}" = "inactive" ] && [ "${enabled}" = "disabled" ]; then
		pass "credentials: config prepared but agent not started before X-Panel"
	else
		fail "agent started/enabled before X-Panel finalize"
	fi
else
	fail "paired credentials did not write initial config"
fi

# Simulate X-Panel start, then finalize agent.
echo active >"${CASE}/state/xpanel.active"
finalize_nezha_agent_service
active="$(cat "${CASE}/state/xpanel-nezha-agent.active")"
enabled="$(cat "${CASE}/state/xpanel-nezha-agent.enabled")"
if [ "${active}" = "active" ] && [ "${enabled}" = "enabled" ]; then
	pass "agent enable/start only after X-Panel start path"
else
	fail "after finalize: active=${active} enabled=${enabled}"
fi

cfg="${INSTALL_DIR}/nezha-agent/config.yml"
PAIRED_CASE="${CASE}"
assert_mode "${INSTALL_DIR}/nezha-agent" "700" || true
assert_mode "${INSTALL_DIR}/nezha-agent/nezha-agent" "755" || true
assert_mode "${cfg}" "600" || true
assert_mode "${SYSTEMD_DIR}/xpanel-nezha-agent.service" "644" || true

python3 - "${cfg}" "${SECRET_FIXTURE}" <<'PY'
import sys
from pathlib import Path
text = Path(sys.argv[1]).read_text(encoding="utf-8")
secret = sys.argv[2]
required = [
    "server: dashboard.example.com:8443",
    f"client_secret: '{secret}'",
    "tls: true",
    "insecure_tls: false",
    "disable_auto_update: true",
    "disable_force_update: true",
    "disable_command_execute: false",
    "node_role: xpanel",
]
for line in required:
    if line not in text:
        raise SystemExit(f"missing YAML field/line: {line!r}\n{text}")
if "uuid:" in text:
    raise SystemExit("initial config must not include uuid")
PY
pass "initial YAML fields match design"

# Legal punctuation must be represented safely in YAML rather than rejected.
CASE="$(mktemp -d "${TMP_ROOT}/quoted-secret.XXXXXX")"
make_package_extract "${CASE}/extract" ok
mkdir -p "${CASE}/systemd"
INSTALL_DIR="${CASE}/opt/xpanel"
SYSTEMD_DIR="${CASE}/systemd"
NEZHA_PACKAGE_DIR="${CASE}/extract"
NEZHA_UNIT_SRC="${CASE}/extract/xpanel-nezha-agent.service"
NEZHA_DASHBOARD_URL='https://dashboard.example.com'
NEZHA_AGENT_SECRET="nzh-secret_ABC:123#x'y"
NEZHA_SERVER=''
NEZHA_CONFIGURE=false
if validate_nezha_install_inputs && install_bundled_nezha_agent; then
	if grep -Fq "client_secret: 'nzh-secret_ABC:123#x''y'" "${INSTALL_DIR}/nezha-agent/config.yml"; then
		pass "punctuated AgentSecret is safely single-quoted in YAML"
	else
		fail "punctuated AgentSecret was not safely YAML-quoted"
	fi
else
	fail "legal punctuation in AgentSecret was rejected"
fi

# No leftover temp config files
residue="$(find "${PAIRED_CASE}/opt/xpanel/nezha-agent" "${INSTALL_DIR}/nezha-agent" \
	\( -name '.config.yml*' -o -name 'config.yml.*' \) 2>/dev/null | head -n 5 || true)"
if [ -z "${residue}" ]; then
	pass "no leftover config temp files"
else
	fail "leftover temp files: ${residue}"
fi

# Secret must not appear in systemctl argv capture or helper stdout from finalize path.
assert_no_secret "systemctl calls" "${PAIRED_CASE}/state/calls"
if grep -Fq "${SECRET_FIXTURE}" "${PAIRED_CASE}/state/calls" 2>/dev/null; then
	fail "secret in systemctl argv"
else
	pass "secret absent from captured systemctl argv"
fi

# ---------------------------------------------------------------------------
# 7) Upgrade never overwrites existing config; state preservation
# ---------------------------------------------------------------------------
run_upgrade_state_case() {
	local label="$1"
	local was_active="$2"
	local was_enabled="$3"
	local expect_active="$4"
	local expect_enabled="$5"

	local CASE BIN STATE
	CASE="$(mktemp -d "${TMP_ROOT}/up-${label}.XXXXXX")"
	make_package_extract "${CASE}/extract" ok
	# New package agent content differs
	printf '#!/bin/sh\necho new-agent-%s\n' "${label}" >"${CASE}/extract/nezha-agent/nezha-agent"
	chmod 0755 "${CASE}/extract/nezha-agent/nezha-agent"

	mkdir -p "${CASE}/opt/xpanel/nezha-agent" "${CASE}/systemd" "${CASE}/bin" "${CASE}/state"
	printf '#!/bin/sh\necho old-agent\n' >"${CASE}/opt/xpanel/nezha-agent/nezha-agent"
	chmod 0755 "${CASE}/opt/xpanel/nezha-agent/nezha-agent"
	printf 'server: keep.example.com:443\nclient_secret: keep-secret\nuuid: keep-uuid\n' \
		>"${CASE}/opt/xpanel/nezha-agent/config.yml"
	chmod 0600 "${CASE}/opt/xpanel/nezha-agent/config.yml"
	cp "${UNIT_FILE}" "${CASE}/systemd/xpanel-nezha-agent.service"
	printf 'old-unit\n' >"${CASE}/systemd/xpanel-nezha-agent.service"
	setup_fake_systemctl "${CASE}/bin" "${CASE}/state"
	echo "${was_active}" >"${CASE}/state/xpanel-nezha-agent.active"
	echo "${was_enabled}" >"${CASE}/state/xpanel-nezha-agent.enabled"
	export PATH="${CASE}/bin:${PATH}"
	INSTALL_DIR="${CASE}/opt/xpanel"
	SYSTEMD_DIR="${CASE}/systemd"
	NEZHA_UNIT_SRC="${CASE}/extract/xpanel-nezha-agent.service"
	NEZHA_PACKAGE_DIR="${CASE}/extract"
	NEZHA_CONFIGURE=false
	NEZHA_AGENT_SECRET=''
	NEZHA_SERVER=''
	NEZHA_EXTERNAL_CONFLICT=false
	IS_UPGRADE=true
	NEZHA_WAS_ACTIVE=false
	NEZHA_WAS_ENABLED=false

	capture_nezha_agent_systemd_state
	stop_nezha_agent_if_active
	# Ensure stop only when originally active
	if [ "${was_active}" = "active" ]; then
		if ! grep -q 'stop xpanel-nezha-agent' "${CASE}/state/calls"; then
			fail "${label}: did not stop active agent"
			return
		fi
	else
		if grep -q 'stop xpanel-nezha-agent' "${CASE}/state/calls" 2>/dev/null; then
			fail "${label}: stopped non-active agent"
			return
		fi
	fi

	install_bundled_nezha_agent
	finalize_nezha_agent_service

	if ! grep -q "new-agent-${label}" "${INSTALL_DIR}/nezha-agent/nezha-agent"; then
		fail "${label}: agent binary not upgraded"
		return
	fi
	if ! grep -q 'uuid: keep-uuid' "${INSTALL_DIR}/nezha-agent/config.yml" \
		|| ! grep -q 'client_secret: keep-secret' "${INSTALL_DIR}/nezha-agent/config.yml"; then
		fail "${label}: config was overwritten"
		return
	fi
	if ! grep -q 'node_role: xpanel' "${INSTALL_DIR}/nezha-agent/config.yml"; then
		fail "${label}: node_role was not merged"
		return
	fi
	active="$(cat "${CASE}/state/xpanel-nezha-agent.active")"
	enabled="$(cat "${CASE}/state/xpanel-nezha-agent.enabled")"
	if [ "${active}" = "${expect_active}" ] && [ "${enabled}" = "${expect_enabled}" ]; then
		pass "upgrade state ${label}: active=${active} enabled=${enabled}, config preserved"
	else
		fail "${label}: got active=${active} enabled=${enabled}, want ${expect_active}/${expect_enabled}"
	fi
}

run_upgrade_state_case "active-enabled" "active" "enabled" "active" "enabled"
run_upgrade_state_case "active-disabled" "active" "disabled" "active" "disabled"
run_upgrade_state_case "stopped-enabled" "inactive" "enabled" "inactive" "enabled"
run_upgrade_state_case "disabled" "inactive" "disabled" "inactive" "disabled"

# Historical upgrade without config: install assets, do not start
CASE="$(mktemp -d "${TMP_ROOT}/hist.XXXXXX")"
make_package_extract "${CASE}/extract" ok
mkdir -p "${CASE}/opt/xpanel" "${CASE}/systemd" "${CASE}/bin" "${CASE}/state"
printf '#!/bin/sh\necho old-xpanel\n' >"${CASE}/opt/xpanel/xpanel"
setup_fake_systemctl "${CASE}/bin" "${CASE}/state"
echo inactive >"${CASE}/state/xpanel-nezha-agent.active"
echo disabled >"${CASE}/state/xpanel-nezha-agent.enabled"
export PATH="${CASE}/bin:${PATH}"
INSTALL_DIR="${CASE}/opt/xpanel"
SYSTEMD_DIR="${CASE}/systemd"
NEZHA_UNIT_SRC="${CASE}/extract/xpanel-nezha-agent.service"
NEZHA_PACKAGE_DIR="${CASE}/extract"
NEZHA_CONFIGURE=false
NEZHA_EXTERNAL_CONFLICT=false
IS_UPGRADE=true
NEZHA_WAS_ACTIVE=false
NEZHA_WAS_ENABLED=false
capture_nezha_agent_systemd_state
stop_nezha_agent_if_active
install_bundled_nezha_agent
finalize_nezha_agent_service
if [ -f "${INSTALL_DIR}/nezha-agent/nezha-agent" ] \
	&& [ ! -e "${INSTALL_DIR}/nezha-agent/config.yml" ] \
	&& [ "$(cat "${CASE}/state/xpanel-nezha-agent.active")" = "inactive" ]; then
	pass "historical upgrade without config does not start agent"
else
	fail "historical upgrade without config misbehaved"
fi

# ---------------------------------------------------------------------------
# 8) External Agent conflict: no stop/overwrite external; X-Panel continues; bundled not started
# ---------------------------------------------------------------------------
CASE="$(mktemp -d "${TMP_ROOT}/ext.XXXXXX")"
make_package_extract "${CASE}/extract" ok
mkdir -p "${CASE}/opt/xpanel" "${CASE}/systemd" "${CASE}/bin" "${CASE}/state" "${CASE}/external/nezha/agent"
printf '#!/bin/sh\necho external-agent\n' >"${CASE}/external/nezha/agent/nezha-agent"
chmod 0755 "${CASE}/external/nezha/agent/nezha-agent"
setup_fake_systemctl "${CASE}/bin" "${CASE}/state"
: >"${CASE}/state/external.nezha-agent.service"
echo active >"${CASE}/state/nezha-agent.service.active"
echo enabled >"${CASE}/state/nezha-agent.service.enabled"
echo inactive >"${CASE}/state/xpanel-nezha-agent.active"
echo disabled >"${CASE}/state/xpanel-nezha-agent.enabled"
export PATH="${CASE}/bin:${PATH}"
INSTALL_DIR="${CASE}/opt/xpanel"
SYSTEMD_DIR="${CASE}/systemd"
NEZHA_EXTERNAL_DIRS=("${CASE}/external/nezha/agent")
NEZHA_UNIT_SRC="${CASE}/extract/xpanel-nezha-agent.service"
NEZHA_PACKAGE_DIR="${CASE}/extract"
NEZHA_DASHBOARD_URL='https://dashboard.example.com'
NEZHA_AGENT_SECRET="${SECRET_FIXTURE}"
NEZHA_CONFIGURE=false
NEZHA_SERVER=''
IS_UPGRADE=false
NEZHA_WAS_ACTIVE=false
NEZHA_WAS_ENABLED=false
NEZHA_EXTERNAL_CONFLICT=false

validate_nezha_install_inputs
detect_external_nezha_agent
if [ "${NEZHA_EXTERNAL_CONFLICT}" != true ]; then
	fail "external agent was not detected"
else
	pass "external agent conflict detected"
fi

# External binary must remain untouched; bundled may install files but must not start.
mkdir -p "${INSTALL_DIR}"
install_bundled_nezha_agent
finalize_nezha_agent_service
if grep -q 'external-agent' "${CASE}/external/nezha/agent/nezha-agent" \
	&& [ -f "${INSTALL_DIR}/nezha-agent/nezha-agent" ] \
	&& [ "$(cat "${CASE}/state/xpanel-nezha-agent.active")" = "inactive" ] \
	&& ! grep -q 'stop nezha-agent' "${CASE}/state/calls" 2>/dev/null; then
	pass "external agent preserved; bundled not started; X-Panel path continues"
else
	fail "external conflict handling incorrect"
fi

CASE="$(mktemp -d "${TMP_ROOT}/ext-instance.XXXXXX")"
mkdir -p "${CASE}/opt/xpanel" "${CASE}/bin" "${CASE}/state"
setup_fake_systemctl "${CASE}/bin" "${CASE}/state"
: >"${CASE}/state/external.nezha-agent@node.service"
export PATH="${CASE}/bin:${PATH}"
INSTALL_DIR="${CASE}/opt/xpanel"
NEZHA_EXTERNAL_DIRS=()
NEZHA_EXTERNAL_CONFLICT=false
detect_external_nezha_agent
if [ "${NEZHA_EXTERNAL_CONFLICT}" = true ]; then
	pass "external instantiated Nezha systemd unit detected"
else
	fail "external instantiated Nezha systemd unit was missed"
fi

# ---------------------------------------------------------------------------
# 9) Interactive secret path (--nezha-dashboard + hidden secret) via a real pseudo-TTY
# ---------------------------------------------------------------------------
if grep -q 'read -rsp' "${INSTALLER}" || grep -q 'read -rs' "${INSTALLER}"; then
	pass "hidden interactive secret prompt present"
else
	fail "missing hidden interactive secret read"
fi

TTY_HELPER="${TMP_ROOT}/interactive-helper.sh"
{
	printf '%s\n' '#!/usr/bin/env bash' 'set -euo pipefail'
	sed -n '/^nezha_dashboard_is_https_origin() {$/,/^}$/p' "${INSTALLER}"
	sed -n '/^normalize_nezha_dashboard_server() {$/,/^}$/p' "${INSTALLER}"
	sed -n '/^validate_nezha_install_inputs() {$/,/^}$/p' "${INSTALLER}"
	printf '%s\n' \
		'log_error() { printf "ERROR:%s\\n" "$*" >&2; }' \
		'NEZHA_DASHBOARD_URL="https://dashboard.example.com"' \
		'NEZHA_AGENT_SECRET=""' \
		'NEZHA_SERVER=""' \
		'NEZHA_CONFIGURE=false' \
		'validate_nezha_install_inputs' \
		'printf "RESULT:%s\\n" "${NEZHA_SERVER}"'
} >"${TTY_HELPER}"
chmod 0700 "${TTY_HELPER}"

TTY_OUTPUT="${TMP_ROOT}/interactive.out"
export XPANEL_TEST_TTY_HELPER="${TTY_HELPER}"
export XPANEL_TEST_TTY_SECRET="${SECRET_FIXTURE}"
set +e
expect <<'EXPECT' >"${TTY_OUTPUT}" 2>&1
set timeout 5
spawn bash $env(XPANEL_TEST_TTY_HELPER)
expect -exact "Nezha Agent Secret (input hidden): "
send -- "$env(XPANEL_TEST_TTY_SECRET)\r"
expect -exact "RESULT:dashboard.example.com:443"
expect eof
EXPECT
tty_rc=$?
set -e
unset XPANEL_TEST_TTY_HELPER XPANEL_TEST_TTY_SECRET
if [ "${tty_rc}" -eq 0 ] \
	&& ! grep -Fq "${SECRET_FIXTURE}" "${TTY_OUTPUT}" \
	&& grep -Fq 'RESULT:dashboard.example.com:443' "${TTY_OUTPUT}"; then
	pass "interactive AgentSecret is accepted through a TTY without echo"
else
	fail "interactive AgentSecret TTY flow failed or echoed the secret"
fi

# ---------------------------------------------------------------------------
# 10) Unit file credential-free + required fields
# ---------------------------------------------------------------------------
if grep -qE 'client_secret|AgentSecret|Environment=.*SECRET' "${UNIT_FILE}"; then
	fail "unit file contains credentials"
else
	pass "unit file is credential-free"
fi
for needle in \
	'Type=simple' \
	'WorkingDirectory=/opt/xpanel/nezha-agent' \
	'ExecStart=/opt/xpanel/nezha-agent/nezha-agent -c /opt/xpanel/nezha-agent/config.yml' \
	'Restart=always' \
	'RestartSec=10' \
	'UMask=0077' \
	'WantedBy=multi-user.target'; do
	if ! grep -Fq "${needle}" "${UNIT_FILE}"; then
		fail "unit missing: ${needle}"
	fi
done
pass "unit file fields match design"

# ---------------------------------------------------------------------------
# 11) install-online ordering: agent enable/start after X-Panel start
# ---------------------------------------------------------------------------
python3 - "${INSTALLER}" <<'PY'
import pathlib
import sys
text = pathlib.Path(sys.argv[1]).read_text(encoding="utf-8")
# Preflight validation must appear before service stop / binary install markers.
pre = text.find("validate_nezha_install_inputs")
stop = text.find("systemctl stop $SERVICE_NAME")
# There may be uninstall stop earlier; find upgrade/install stop after extract.
extract = text.find('tar -xzf "$TMP_DIR/xpanel.tar.gz"')
stop_after = text.find("systemctl stop $SERVICE_NAME", extract)
require = text.find("require_nezha_agent_package")
start_panel = text.find("systemctl start $SERVICE_NAME", extract)
finalize = text.find("\nfinalize_nezha_agent_service\n", start_panel)
assert pre != -1, "validate_nezha_install_inputs not found"
assert require != -1, "require_nezha_agent_package not found"
assert extract != -1 and stop_after != -1, "extract/stop markers missing"
assert require < stop_after, "agent package check must precede service stop"
assert start_panel != -1 and finalize != -1, "start/finalize markers missing"
assert start_panel < finalize, "agent finalize must run after X-Panel start"
assert "XPANEL_NEZHA_DASHBOARD_URL" in text
assert "XPANEL_NEZHA_AGENT_SECRET" in text
assert "--nezha-dashboard" in text
# No secret flag
for bad in ("--nezha-secret", "--nezha-agent-secret", "--client-secret"):
    assert bad not in text, bad
print("order-ok")
PY
pass "install-online preflight/start ordering is correct"

# ---------------------------------------------------------------------------
# 12) offline install.sh also gains Nezha support markers
# ---------------------------------------------------------------------------
if grep -q 'nezha-agent' "${OFFLINE_INSTALLER}" \
	&& grep -q 'XPANEL_NEZHA_AGENT_SECRET\|NEZHA_AGENT_SECRET\|nezha-dashboard' "${OFFLINE_INSTALLER}"; then
	pass "install.sh includes Nezha agent install support"
else
	fail "install.sh missing Nezha support"
fi

# ---------------------------------------------------------------------------
# Secret must never appear in this test's own temp tree outside config.yml
# ---------------------------------------------------------------------------
leaks="$(grep -RFl "${SECRET_FIXTURE}" "${TMP_ROOT}" 2>/dev/null | grep -v '/config.yml$' || true)"
if [ -n "${leaks}" ]; then
	fail "secret leaked into non-config files: ${leaks}"
else
	pass "secret only in intended config.yml fixtures"
fi

# ---------------------------------------------------------------------------
echo ""
echo "Passed: ${PASS}"
echo "Failed: ${FAIL}"
if [ "${FAIL}" -ne 0 ]; then
	exit 1
fi
printf 'Nezha agent install fixture tests passed.\n'
