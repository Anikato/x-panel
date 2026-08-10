#!/usr/bin/env bash
# Stage official nezha-agent binary + checked-in license/notice into a release directory.
# Usage: package-nezha-agent.sh VERSION ARCH ZIP CHECKSUMS RELEASE_DIR
set -euo pipefail

usage() {
	echo "usage: $0 VERSION ARCH ZIP CHECKSUMS RELEASE_DIR" >&2
	exit 1
}

if [ "$#" -ne 5 ]; then
	usage
fi

VERSION="$1"
ARCH="$2"
ZIP="$3"
CHECKSUMS="$4"
RELEASE_DIR="$5"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
LICENSE_SRC="${ROOT_DIR}/third_party/nezha-agent/LICENSE"
NOTICE_SRC="${ROOT_DIR}/third_party/nezha-agent/NOTICE.md"

die() {
	echo "package-nezha-agent: $*" >&2
	exit 1
}

# Reusable pinned-style version: v + dotted numeric (e.g. v2.3.1). CI pins the value.
case "${VERSION}" in
v[0-9]* )
	if ! printf '%s' "${VERSION}" | grep -Eq '^v[0-9]+(\.[0-9]+)*$'; then
		die "invalid VERSION (want vNUMERIC, e.g. v2.3.1): ${VERSION}"
	fi
	;;
*)
	die "invalid VERSION (want vNUMERIC, e.g. v2.3.1): ${VERSION}"
	;;
esac

case "${ARCH}" in
amd64 | arm64) ;;
*)
	die "invalid ARCH (want amd64 or arm64): ${ARCH}"
	;;
esac

EXPECTED_NAME="nezha-agent_linux_${ARCH}.zip"
ZIP_BASE="$(basename "${ZIP}")"
if [ "${ZIP_BASE}" != "${EXPECTED_NAME}" ]; then
	die "ZIP basename must be ${EXPECTED_NAME}, got ${ZIP_BASE}"
fi

if [ ! -f "${ZIP}" ] || [ -L "${ZIP}" ]; then
	die "ZIP must be a regular file: ${ZIP}"
fi
if [ ! -f "${CHECKSUMS}" ] || [ -L "${CHECKSUMS}" ]; then
	die "CHECKSUMS must be a regular file: ${CHECKSUMS}"
fi
if [ ! -f "${LICENSE_SRC}" ] || [ -L "${LICENSE_SRC}" ]; then
	die "missing checked-in LICENSE: ${LICENSE_SRC}"
fi
if [ ! -f "${NOTICE_SRC}" ] || [ -L "${NOTICE_SRC}" ]; then
	die "missing checked-in NOTICE: ${NOTICE_SRC}"
fi

# RELEASE_DIR must be a concrete, non-root destination before any mkdir/rm.
if [ -z "${RELEASE_DIR}" ]; then
	die "RELEASE_DIR must be non-empty"
fi
if [ "${RELEASE_DIR}" = "/" ]; then
	die "RELEASE_DIR must not be /"
fi
if [ -e "${RELEASE_DIR}" ] || [ -L "${RELEASE_DIR}" ]; then
	if [ -L "${RELEASE_DIR}" ]; then
		die "RELEASE_DIR must not be a symlink: ${RELEASE_DIR}"
	fi
	if [ ! -d "${RELEASE_DIR}" ]; then
		die "RELEASE_DIR must be a directory: ${RELEASE_DIR}"
	fi
fi

# Exactly one checksum line for this filename; first field is 64-hex hash.
# Official checksums.txt format: HASH space filename
MATCH_COUNT="$(
	awk -v name="${EXPECTED_NAME}" '
		NF >= 2 && $2 == name { count++ }
		END { print count + 0 }
	' "${CHECKSUMS}"
)"
if [ "${MATCH_COUNT}" -ne 1 ]; then
	die "checksums must contain exactly one line for ${EXPECTED_NAME} (found ${MATCH_COUNT})"
fi

HASH="$(
	awk -v name="${EXPECTED_NAME}" '
		NF >= 2 && $2 == name { print $1; exit }
	' "${CHECKSUMS}"
)"
if ! printf '%s' "${HASH}" | grep -Eq '^[0-9a-fA-F]{64}$'; then
	die "checksum hash for ${EXPECTED_NAME} must be 64 hex characters"
fi

RAW_LINE="$(
	awk -v name="${EXPECTED_NAME}" '
		NF >= 2 && $2 == name { print; exit }
	' "${CHECKSUMS}"
)"

WORK_DIR="$(mktemp -d "${TMPDIR:-/tmp}/package-nezha-agent.XXXXXX")"
cleanup() {
	rm -rf "${WORK_DIR}"
}
trap cleanup EXIT

# Isolate checksum verification: original line + basename zip, then sha256sum --check.
CHECK_DIR="${WORK_DIR}/check"
mkdir -p "${CHECK_DIR}"
# Preserve the original checksums line bytes (no reformat).
printf '%s\n' "${RAW_LINE}" >"${CHECK_DIR}/checksums-line.txt"
cp "${ZIP}" "${CHECK_DIR}/${EXPECTED_NAME}"
(
	cd "${CHECK_DIR}"
	sha256sum --check checksums-line.txt
) || die "sha256sum --check failed for ${EXPECTED_NAME}"

EXTRACT_DIR="${WORK_DIR}/extract"
mkdir -p "${EXTRACT_DIR}"
unzip -q "${CHECK_DIR}/${EXPECTED_NAME}" -d "${EXTRACT_DIR}"

AGENT_SRC="${EXTRACT_DIR}/nezha-agent"
if [ -L "${AGENT_SRC}" ]; then
	die "extracted nezha-agent must not be a symlink"
fi
if [ ! -f "${AGENT_SRC}" ]; then
	die "zip must contain a regular file named nezha-agent at archive root"
fi
if [ ! -x "${AGENT_SRC}" ]; then
	die "extracted nezha-agent must be executable"
fi

# Stage completely under a temp directory, then safely replace release/nezha-agent.
mkdir -p "${RELEASE_DIR}"
STAGE_DIR="$(mktemp -d "${RELEASE_DIR}/.nezha-agent-stage.XXXXXX")"
# Ensure stage cleanup on failure before trap replaces WORK_DIR only.
stage_cleanup() {
	rm -rf "${STAGE_DIR}"
	cleanup
}
trap stage_cleanup EXIT

install -m 0755 "${AGENT_SRC}" "${STAGE_DIR}/nezha-agent"
install -m 0644 "${LICENSE_SRC}" "${STAGE_DIR}/LICENSE"
install -m 0644 "${NOTICE_SRC}" "${STAGE_DIR}/NOTICE.md"

TARGET_DIR="${RELEASE_DIR}/nezha-agent"
BACKUP_DIR=""
# Reject symlink or non-directory targets; never follow or replace through a link.
if [ -L "${TARGET_DIR}" ]; then
	die "target must not be a symlink: ${TARGET_DIR}"
fi
if [ -e "${TARGET_DIR}" ]; then
	if [ ! -d "${TARGET_DIR}" ]; then
		die "target must be a directory: ${TARGET_DIR}"
	fi
	# mktemp -d yields a unique, non-empty path in the same directory; free the
	# placeholder name then mv the old target onto it (no $$, no pre-rm of backups).
	BACKUP_DIR="$(mktemp -d "${RELEASE_DIR}/.nezha-agent-backup.XXXXXX")"
	rmdir "${BACKUP_DIR}"
	mv "${TARGET_DIR}" "${BACKUP_DIR}"
fi

if ! mv "${STAGE_DIR}" "${TARGET_DIR}"; then
	if [ -n "${BACKUP_DIR}" ] && [ -e "${BACKUP_DIR}" ]; then
		mv "${BACKUP_DIR}" "${TARGET_DIR}" || true
	fi
	die "failed to install staged nezha-agent into ${TARGET_DIR}"
fi

# Only remove a backup path we created and that still exists; never rm empty/unresolved.
if [ -n "${BACKUP_DIR}" ] && [ -e "${BACKUP_DIR}" ]; then
	rm -rf "${BACKUP_DIR}"
fi

# Success: target owns the staged tree; only remove WORK_DIR.
trap cleanup EXIT

exit 0
