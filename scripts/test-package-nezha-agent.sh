#!/usr/bin/env bash
# Offline tests for scripts/package-nezha-agent.sh.
# Does not network, touch build/, or use real release assets.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
HELPER="${ROOT_DIR}/scripts/package-nezha-agent.sh"
TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/xpanel-package-nezha-agent.XXXXXX")"
PASS=0
FAIL=0

cleanup() {
	rm -rf "${TMP_ROOT}"
}
trap cleanup EXIT

fail() {
	printf 'FAIL: %s\n' "$*" >&2
	FAIL=$((FAIL + 1))
}

pass() {
	printf 'PASS: %s\n' "$*"
	PASS=$((PASS + 1))
}

# Portable mode bits: GNU stat -c '%a', BSD/macOS stat -f '%Lp'.
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
	# Accept 755 or 0755 style.
	if [ "${got}" = "${want}" ] || [ "${got}" = "0${want}" ] || [ "0${got}" = "${want}" ]; then
		return 0
	fi
	fail "mode of ${path}: got ${got}, want ${want}"
	return 1
}

# Build a zip named nezha-agent_linux_${arch}.zip under $1 with payload kind.
# kind: ok | missing | nonexec | symlink | directory
make_fixture_zip() {
	local out_dir="$1"
	local arch="$2"
	local kind="$3"
	local work zip_path name

	work="$(mktemp -d "${TMP_ROOT}/fixture.XXXXXX")"
	name="nezha-agent_linux_${arch}.zip"
	zip_path="${out_dir}/${name}"

	case "${kind}" in
	ok)
		printf '#!/bin/sh\necho fake-nezha-agent-%s\n' "${arch}" >"${work}/nezha-agent"
		chmod 0755 "${work}/nezha-agent"
		(cd "${work}" && zip -q "${zip_path}" nezha-agent)
		;;
	missing)
		printf 'not-the-agent\n' >"${work}/README.txt"
		(cd "${work}" && zip -q "${zip_path}" README.txt)
		;;
	nonexec)
		printf '#!/bin/sh\necho nonexec\n' >"${work}/nezha-agent"
		chmod 0644 "${work}/nezha-agent"
		(cd "${work}" && zip -q "${zip_path}" nezha-agent)
		;;
	symlink)
		printf 'target-body\n' >"${work}/real-agent"
		ln -s real-agent "${work}/nezha-agent"
		(cd "${work}" && zip -qy "${zip_path}" nezha-agent)
		;;
	directory)
		mkdir -p "${work}/nezha-agent"
		printf 'inside\n' >"${work}/nezha-agent/nested"
		(cd "${work}" && zip -qr "${zip_path}" nezha-agent)
		;;
	*)
		echo "unknown fixture kind: ${kind}" >&2
		return 1
		;;
	esac

	rm -rf "${work}"
	printf '%s\n' "${zip_path}"
}

write_checksums() {
	local checksums_path="$1"
	shift
	local zip_path hash base
	: >"${checksums_path}"
	for zip_path in "$@"; do
		hash="$(sha256sum "${zip_path}" | awk '{print $1}')"
		base="$(basename "${zip_path}")"
		# Official format: HASH space filename
		printf '%s %s\n' "${hash}" "${base}" >>"${checksums_path}"
	done
}

run_helper() {
	local version="$1"
	local arch="$2"
	local zip_path="$3"
	local checksums="$4"
	local release_dir="$5"
	set +e
	"${HELPER}" "${version}" "${arch}" "${zip_path}" "${checksums}" "${release_dir}" \
		>"${TMP_ROOT}/helper.stdout" 2>"${TMP_ROOT}/helper.stderr"
	local rc=$?
	set -e
	return "${rc}"
}

# --- RED gate: helper must exist for GREEN ---
if [ ! -f "${HELPER}" ]; then
	echo "RED: ${HELPER} is absent (expected before implementation)"
	# Still exercise that the test harness itself is runnable and would call the helper.
	fake_release="$(mktemp -d "${TMP_ROOT}/red-release.XXXXXX")"
	fake_zip_dir="$(mktemp -d "${TMP_ROOT}/red-zip.XXXXXX")"
	zip_path="$(make_fixture_zip "${fake_zip_dir}" amd64 ok)"
	checksums="${fake_zip_dir}/checksums.txt"
	write_checksums "${checksums}" "${zip_path}"
	set +e
	bash "${HELPER}" "v2.3.1" "amd64" "${zip_path}" "${checksums}" "${fake_release}"
	rc=$?
	set -e
	if [ "${rc}" -eq 0 ]; then
		echo "unexpected success without helper" >&2
		exit 1
	fi
	echo "test harness confirmed helper invocation fails while helper is missing"
	exit 1
fi

if [ ! -x "${HELPER}" ]; then
	chmod +x "${HELPER}" 2>/dev/null || true
fi

VERSION="v2.3.1"

# Shared good fixtures for amd64 + arm64
ASSET_DIR="$(mktemp -d "${TMP_ROOT}/assets.XXXXXX")"
AMD64_ZIP="$(make_fixture_zip "${ASSET_DIR}" amd64 ok)"
ARM64_ZIP="$(make_fixture_zip "${ASSET_DIR}" arm64 ok)"
CHECKSUMS="${ASSET_DIR}/checksums.txt"
write_checksums "${CHECKSUMS}" "${AMD64_ZIP}" "${ARM64_ZIP}"

check_success_layout() {
	local release_dir="$1"
	local arch="$2"
	local label="$3"
	local agent license notice extra

	agent="${release_dir}/nezha-agent/nezha-agent"
	license="${release_dir}/nezha-agent/LICENSE"
	notice="${release_dir}/nezha-agent/NOTICE.md"

	if [ ! -f "${agent}" ] || [ -L "${agent}" ]; then
		fail "${label}: missing regular agent binary"
		return
	fi
	if [ ! -x "${agent}" ]; then
		fail "${label}: agent not executable"
		return
	fi
	assert_mode "${agent}" "755" || true

	if [ ! -f "${license}" ] || [ -L "${license}" ]; then
		fail "${label}: missing LICENSE"
		return
	fi
	assert_mode "${license}" "644" || true

	if [ ! -f "${notice}" ] || [ -L "${notice}" ]; then
		fail "${label}: missing NOTICE.md"
		return
	fi
	assert_mode "${notice}" "644" || true

	# No other files under release/ or under nezha-agent/.
	extra="$(find "${release_dir}" -mindepth 1 \( -type f -o -type l \) ! -path "${agent}" ! -path "${license}" ! -path "${notice}" | head -n 5 || true)"
	if [ -n "${extra}" ]; then
		fail "${label}: unexpected release files: ${extra}"
		return
	fi

	# Binary content must match staged payload for this arch.
	if ! grep -q "fake-nezha-agent-${arch}" "${agent}"; then
		fail "${label}: agent content does not match fixture for ${arch}"
		return
	fi

	# LICENSE must match checked-in third_party copy.
	if ! cmp -s "${license}" "${ROOT_DIR}/third_party/nezha-agent/LICENSE"; then
		fail "${label}: staged LICENSE differs from third_party"
		return
	fi
	if ! cmp -s "${notice}" "${ROOT_DIR}/third_party/nezha-agent/NOTICE.md"; then
		fail "${label}: staged NOTICE.md differs from third_party"
		return
	fi

	pass "${label}: layout and modes"
}

# --- success: amd64 ---
REL_AMD64="$(mktemp -d "${TMP_ROOT}/release-amd64.XXXXXX")"
if run_helper "${VERSION}" "amd64" "${AMD64_ZIP}" "${CHECKSUMS}" "${REL_AMD64}"; then
	check_success_layout "${REL_AMD64}" "amd64" "amd64 success"
else
	fail "amd64 success path: helper failed: $(cat "${TMP_ROOT}/helper.stderr" 2>/dev/null || true)"
fi

# --- success: arm64 ---
REL_ARM64="$(mktemp -d "${TMP_ROOT}/release-arm64.XXXXXX")"
if run_helper "${VERSION}" "arm64" "${ARM64_ZIP}" "${CHECKSUMS}" "${REL_ARM64}"; then
	check_success_layout "${REL_ARM64}" "arm64" "arm64 success"
else
	fail "arm64 success path: helper failed: $(cat "${TMP_ROOT}/helper.stderr" 2>/dev/null || true)"
fi

# --- success: customized release tag ---
REL_CUSTOM_TAG="$(mktemp -d "${TMP_ROOT}/release-custom-tag.XXXXXX")"
if run_helper "agent-v2.3.1-xpanel.1" "amd64" "${AMD64_ZIP}" "${CHECKSUMS}" "${REL_CUSTOM_TAG}"; then
	check_success_layout "${REL_CUSTOM_TAG}" "amd64" "custom release tag success"
else
	fail "custom release tag should be accepted: $(cat "${TMP_ROOT}/helper.stderr" 2>/dev/null || true)"
fi

# --- wrong checksum ---
BAD_DIR="$(mktemp -d "${TMP_ROOT}/bad-checksum.XXXXXX")"
BAD_ZIP="$(make_fixture_zip "${BAD_DIR}" amd64 ok)"
BAD_SUMS="${BAD_DIR}/checksums.txt"
printf '%s %s\n' "0000000000000000000000000000000000000000000000000000000000000000" "$(basename "${BAD_ZIP}")" >"${BAD_SUMS}"
REL_BAD="$(mktemp -d "${TMP_ROOT}/release-bad-checksum.XXXXXX")"
# Pre-seed a marker so we can detect partial/broken replacement.
mkdir -p "${REL_BAD}/nezha-agent"
printf 'pre-existing\n' >"${REL_BAD}/nezha-agent/marker"
if run_helper "${VERSION}" "amd64" "${BAD_ZIP}" "${BAD_SUMS}" "${REL_BAD}"; then
	fail "wrong checksum: helper should have failed"
else
	if [ -f "${REL_BAD}/nezha-agent/nezha-agent" ]; then
		fail "wrong checksum: partial agent binary was staged"
	elif [ ! -f "${REL_BAD}/nezha-agent/marker" ]; then
		# Replacing/removing existing staging on failure is also unacceptable partial state loss
		# if we only partially wrote; keeping prior tree is fine. If marker gone without new agent,
		# that is a partial failed replace.
		fail "wrong checksum: pre-existing nezha-agent tree was disturbed without successful replace"
	else
		pass "wrong checksum rejected; no partial staging"
	fi
fi

# --- zip missing nezha-agent ---
MISS_DIR="$(mktemp -d "${TMP_ROOT}/missing.XXXXXX")"
MISS_ZIP="$(make_fixture_zip "${MISS_DIR}" amd64 missing)"
MISS_SUMS="${MISS_DIR}/checksums.txt"
write_checksums "${MISS_SUMS}" "${MISS_ZIP}"
REL_MISS="$(mktemp -d "${TMP_ROOT}/release-missing.XXXXXX")"
if run_helper "${VERSION}" "amd64" "${MISS_ZIP}" "${MISS_SUMS}" "${REL_MISS}"; then
	fail "missing agent entry: helper should have failed"
else
	if [ -e "${REL_MISS}/nezha-agent" ]; then
		fail "missing agent entry: left nezha-agent staging behind"
	else
		pass "missing nezha-agent entry rejected; no staging left"
	fi
fi

# --- non-executable agent ---
NEX_DIR="$(mktemp -d "${TMP_ROOT}/nonexec.XXXXXX")"
NEX_ZIP="$(make_fixture_zip "${NEX_DIR}" amd64 nonexec)"
NEX_SUMS="${NEX_DIR}/checksums.txt"
write_checksums "${NEX_SUMS}" "${NEX_ZIP}"
REL_NEX="$(mktemp -d "${TMP_ROOT}/release-nonexec.XXXXXX")"
if run_helper "${VERSION}" "amd64" "${NEX_ZIP}" "${NEX_SUMS}" "${REL_NEX}"; then
	fail "non-executable agent: helper should have failed"
else
	if [ -e "${REL_NEX}/nezha-agent" ]; then
		fail "non-executable agent: left nezha-agent staging behind"
	else
		pass "non-executable agent rejected; no staging left"
	fi
fi

# --- symlink agent ---
SYM_DIR="$(mktemp -d "${TMP_ROOT}/symlink.XXXXXX")"
SYM_ZIP="$(make_fixture_zip "${SYM_DIR}" amd64 symlink)"
SYM_SUMS="${SYM_DIR}/checksums.txt"
write_checksums "${SYM_SUMS}" "${SYM_ZIP}"
REL_SYM="$(mktemp -d "${TMP_ROOT}/release-symlink.XXXXXX")"
if run_helper "${VERSION}" "amd64" "${SYM_ZIP}" "${SYM_SUMS}" "${REL_SYM}"; then
	fail "symlink agent: helper should have failed"
else
	if [ -e "${REL_SYM}/nezha-agent" ]; then
		fail "symlink agent: left nezha-agent staging behind"
	else
		pass "symlink agent rejected; no staging left"
	fi
fi

# --- non-regular (directory) agent ---
DIR_FIX="$(mktemp -d "${TMP_ROOT}/directory.XXXXXX")"
DIR_ZIP="$(make_fixture_zip "${DIR_FIX}" arm64 directory)"
DIR_SUMS="${DIR_FIX}/checksums.txt"
write_checksums "${DIR_SUMS}" "${DIR_ZIP}"
REL_DIR="$(mktemp -d "${TMP_ROOT}/release-directory.XXXXXX")"
if run_helper "${VERSION}" "arm64" "${DIR_ZIP}" "${DIR_SUMS}" "${REL_DIR}"; then
	fail "directory agent: helper should have failed"
else
	if [ -e "${REL_DIR}/nezha-agent" ]; then
		fail "directory agent: left nezha-agent staging behind"
	else
		pass "directory agent rejected; no staging left"
	fi
fi

# --- invalid arch / basename mismatch ---
REL_INV="$(mktemp -d "${TMP_ROOT}/release-invalid.XXXXXX")"
if run_helper "${VERSION}" "riscv64" "${AMD64_ZIP}" "${CHECKSUMS}" "${REL_INV}"; then
	fail "invalid arch: helper should have failed"
else
	pass "invalid arch rejected"
fi

WRONG_NAME_DIR="$(mktemp -d "${TMP_ROOT}/wrong-name.XXXXXX")"
cp "${AMD64_ZIP}" "${WRONG_NAME_DIR}/not-the-official-name.zip"
WRONG_SUMS="${WRONG_NAME_DIR}/checksums.txt"
hash="$(sha256sum "${WRONG_NAME_DIR}/not-the-official-name.zip" | awk '{print $1}')"
printf '%s %s\n' "${hash}" "not-the-official-name.zip" >"${WRONG_SUMS}"
if run_helper "${VERSION}" "amd64" "${WRONG_NAME_DIR}/not-the-official-name.zip" "${WRONG_SUMS}" "${REL_INV}"; then
	fail "wrong zip basename: helper should have failed"
else
	pass "wrong zip basename rejected"
fi

# --- replace existing plain directory target; no backup/stage residue ---
REL_REPLACE="$(mktemp -d "${TMP_ROOT}/release-replace.XXXXXX")"
mkdir -p "${REL_REPLACE}/nezha-agent"
printf 'old-agent-body\n' >"${REL_REPLACE}/nezha-agent/nezha-agent"
printf 'old-license\n' >"${REL_REPLACE}/nezha-agent/LICENSE"
printf 'old-notice\n' >"${REL_REPLACE}/nezha-agent/NOTICE.md"
printf 'stale-extra\n' >"${REL_REPLACE}/nezha-agent/stale-extra.txt"
if run_helper "${VERSION}" "amd64" "${AMD64_ZIP}" "${CHECKSUMS}" "${REL_REPLACE}"; then
	if [ -f "${REL_REPLACE}/nezha-agent/stale-extra.txt" ]; then
		fail "replace existing dir: stale content remained"
	elif ! grep -q "fake-nezha-agent-amd64" "${REL_REPLACE}/nezha-agent/nezha-agent"; then
		fail "replace existing dir: new agent content missing"
	else
		residue="$(find "${REL_REPLACE}" \( -name '.nezha-agent-backup*' -o -name '.nezha-agent-stage*' \) 2>/dev/null | head -n 5 || true)"
		if [ -n "${residue}" ]; then
			fail "replace existing dir: leftover backup/stage paths: ${residue}"
		else
			check_success_layout "${REL_REPLACE}" "amd64" "replace existing dir"
		fi
	fi
else
	fail "replace existing dir: helper failed: $(cat "${TMP_ROOT}/helper.stderr" 2>/dev/null || true)"
fi

# --- target is a symlink: must fail without touching link target or content ---
REL_TSYM="$(mktemp -d "${TMP_ROOT}/release-target-symlink.XXXXXX")"
REAL_TARGET="$(mktemp -d "${TMP_ROOT}/real-target.XXXXXX")"
printf 'protected-payload\n' >"${REAL_TARGET}/keep-me.txt"
printf 'protected-agent\n' >"${REAL_TARGET}/nezha-agent"
ln -s "${REAL_TARGET}" "${REL_TSYM}/nezha-agent"
if run_helper "${VERSION}" "amd64" "${AMD64_ZIP}" "${CHECKSUMS}" "${REL_TSYM}"; then
	fail "target symlink: helper should have failed"
else
	if [ ! -L "${REL_TSYM}/nezha-agent" ]; then
		fail "target symlink: symlink was removed or replaced"
	elif [ ! -f "${REAL_TARGET}/keep-me.txt" ] || ! grep -q 'protected-payload' "${REAL_TARGET}/keep-me.txt"; then
		fail "target symlink: symlink target content was disturbed"
	elif [ ! -f "${REAL_TARGET}/nezha-agent" ] || ! grep -q 'protected-agent' "${REAL_TARGET}/nezha-agent"; then
		fail "target symlink: existing agent content under link target was disturbed"
	elif [ -e "${REAL_TARGET}/LICENSE" ] || [ -e "${REAL_TARGET}/NOTICE.md" ]; then
		fail "target symlink: staged files appeared under symlink target"
	else
		pass "target symlink rejected; link target untouched"
	fi
fi

# --- RELEASE_DIR must not be / ---
if run_helper "${VERSION}" "amd64" "${AMD64_ZIP}" "${CHECKSUMS}" "/"; then
	fail "RELEASE_DIR=/: helper should have failed"
else
	# Must not leave staging artifacts under /
	if [ -e /nezha-agent ] && [ ! -d /nezha-agent ]; then
		fail "RELEASE_DIR=/: created unexpected /nezha-agent non-dir"
	elif find / -maxdepth 1 \( -name '.nezha-agent-stage*' -o -name '.nezha-agent-backup*' \) 2>/dev/null | grep -q .; then
		fail "RELEASE_DIR=/: left stage/backup under /"
	else
		pass "RELEASE_DIR=/ rejected"
	fi
fi

echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
if [ "${FAIL}" -ne 0 ]; then
	exit 1
fi
exit 0
