# X-Panel Bundled Nezha Agent Implementation Plan

> **Historical record:** This implementation plan preserves the original task context. It is not a current release runbook or source of truth. Use `AGENTS.md`, `RELEASE.md`, and `docs/dashboard-agent-xpanel.md`; references below to an “official Agent” describe the original plan, not the current artifact source.

> **For agentic workers:** REQUIRED SUB-SKILL: use the test-driven-development workflow task-by-task. Grok Build implements one task at a time; Codex performs specification review, code-quality review, and fresh verification before the next task. Do not create branches, worktrees, commits, or pushes.

**Goal:** Bundle the official Nezha Agent v2.3.1 with X-Panel, manage it as an independent `xpanel-nezha-agent.service`, and remove Fleet Center/Fleet Reporter/Fleet v2 from every active X-Panel path without losing unrelated user changes.

**Architecture:** X-Panel owns only the Agent binary, `config.yml`, encrypted DB mirrors, systemd lifecycle, release packaging, and version-bound upgrades. The official Agent remains a separate root systemd process and owns UUID generation, Dashboard registration, monitoring, tasks, terminal, remote configuration, and key rotation. `config.yml` is authoritative once present; X-Panel merges only explicitly submitted fields and never restores stale DB values into the file.

**Tech Stack:** Go 1.26, Gin, GORM/SQLite, `gopkg.in/yaml.v3`, systemd, Bash, GitHub Actions, Vue 3, TypeScript, Element Plus, Node 24 built-in test runner.

**Pinned supply chain:** `nezhahq/agent` `v2.3.1`; official assets `nezha-agent_linux_amd64.zip`, `nezha-agent_linux_arm64.zip`, and `checksums.txt`. The official SHA256 values at planning time are `5b652f909b7e944c9ce267e50fa814028b8c65832a6bec162914803733028a24` for amd64 and `bc99811ad6c4449e710ad33e0eb9fab57390cefa3a5fe5a346ac6f1e6ff20a0f` for arm64; CI still verifies the downloaded `checksums.txt` rather than trusting these prose values.

---

## Working-tree protection and execution rules

- Treat every existing tracked and untracked change as user-owned. Never restore a file from `HEAD`, never replace a whole dirty file from generated output, and never run `git reset`, `git checkout`, `git clean`, branch, worktree, commit, or push commands.
- Before every Grok task, capture `git status --short`, `git diff --stat`, and the task's target-file diff. After the task, review only the incremental diff relative to that snapshot.
- Preserve the non-Fleet work already present in shared files, especially credential encryption, certificate-lineage migration, manifest shell-quoting hardening, `xpctl` packaging, and unrelated frontend settings work.
- `fleet-center/` is archive-only. Do not build or extend `server/`, `web/`, legacy `backend/`, or legacy `frontend/` there.
- Follow RED → verify expected failure → minimal GREEN → focused regression for every behavior change. Configuration-only files are verified by executable shell/static checks.
- A task is complete only after Codex first checks the reviewed design for missing/extra behavior, then reviews safety/quality, then runs fresh tests.

## Task 1: Secure Agent configuration core

**Files:**

- Create: `backend/app/dto/nezha_agent.go`
- Create: `backend/app/service/nezha_agent_config.go`
- Create: `backend/app/service/nezha_agent_config_test.go`
- Modify: `backend/security/credentials/registry.go`
- Modify: `backend/security/credentials/database_test.go`
- Modify: `backend/app/repo/secure_fields_test.go`

- [ ] **Step 1: Add failing URL-normalization tests.** Cover a default-port origin, an explicit port, uppercase/trailing-slash normalization, and rejection of HTTP, userinfo, paths, queries, fragments, missing host, and ports outside 1–65535.

  ```go
  func TestNormalizeNezhaDashboardOrigin(t *testing.T) {
      got, err := normalizeNezhaDashboardOrigin("https://dashboard.example.com")
      if err != nil || got.Server != "dashboard.example.com:443" || !got.TLS || got.InsecureTLS {
          t.Fatalf("normalize = %#v, %v", got, err)
      }
  }
  ```

- [ ] **Step 2: Run the focused test and confirm RED.**

  Run: `cd backend && go test ./app/service -run 'TestNormalizeNezhaDashboardOrigin' -count=1`

  Expected: build failure because `normalizeNezhaDashboardOrigin` does not exist.

- [ ] **Step 3: Implement the minimal configuration types and normalizer.** `NezhaAgentConfigUpdate` uses pointer fields for dashboard and remote-operations intent; an omitted or empty secret means “leave unchanged.” `normalizeNezhaDashboardOrigin` returns canonical HTTPS origin plus `host:port`, `tls=true`, and `insecure_tls=false`.

  ```go
  type NezhaAgentConfigUpdate struct {
      DashboardURL           *string `json:"dashboardUrl"`
      ClientSecret           *string `json:"clientSecret"`
      RemoteOperationsEnabled *bool  `json:"remoteOperationsEnabled"`
      EnableAndStart         bool    `json:"enableAndStart"`
  }
  ```

- [ ] **Step 4: Add failing YAML merge and filesystem-safety tests.** Verify initial defaults, preservation of `uuid`, rotated `client_secret`, unknown scalar/list/map fields, no-op empty secret, explicit remote-operations change, corrupt YAML rejection, missing-file behavior, symlink/non-regular rejection, `0600` repair, atomic failure leaving original bytes unchanged, and final directory/file modes `0700`/`0600`.

  ```go
  func TestMergeNezhaConfigPreservesRotatedSecretUUIDAndUnknownFields(t *testing.T) {
      input := []byte("server: old:443\nclient_secret: rotated\nuuid: node-uuid\ncustom_ip_api:\n  - https://ip.example\n")
      updated, err := mergeNezhaConfig(input, nezhaConfigPatch{Server: ptr("new:443")})
      if err != nil { t.Fatal(err) }
      assertYAMLValue(t, updated, "client_secret", "rotated")
      assertYAMLValue(t, updated, "uuid", "node-uuid")
  }
  ```

- [ ] **Step 5: Implement generic YAML merge and durable atomic write.** Use `map[string]any`, `os.Lstat`, a same-directory `os.CreateTemp`, `Chmod(0600)`, `Sync`, `Close`, `Rename`, and parent-directory `Sync`. Never read `NezhaClientSecret` from DB to fill a file. First explicit configuration writes `disable_auto_update=true`, `disable_force_update=true`, `disable_command_execute=false`, and no UUID; later saves touch only submitted fields plus TLS fields when the Dashboard origin changes.

- [ ] **Step 6: Register `NezhaClientSecret` as a new encrypted setting and add Nezha repository assertions.** Preserve the generic `CreateOrUpdateMany` transaction because Agent file-to-DB sync uses it. Keep the existing Fleet registrations temporarily; Task 8 removes them only after the raw retirement migration is in place.

  ```go
  var SecretSettingKeys = map[string]struct{}{
      // existing keys remain unchanged until their owning feature is retired
      "NezhaClientSecret": {},
  }
  ```

- [ ] **Step 7: Verify GREEN and focused regressions.**

  Run: `cd backend && go test ./app/service ./app/repo ./security/credentials -run 'Nezha|SettingRepository|Registry' -count=1`

  Expected: all selected tests pass and no secret appears in failure output.

## Task 2: systemd lifecycle, status, conflict detection, and log redaction

**Files:**

- Create: `backend/app/service/nezha_agent.go`
- Create: `backend/app/service/nezha_agent_test.go`
- Create: `scripts/xpanel-nezha-agent.service`
- Modify: `backend/app/service/systemd_service.go`
- Modify: `backend/app/dto/nezha_agent.go`

- [ ] **Step 1: Add failing state and operation tests using injected command/file dependencies.** Cover independent active/enabled states, drift from `NezhaEnabled`, `start`, `stop`, `restart`, `enable --now`, `disable --now`, DB updates only after successful enable/disable, wait-for-stop before file read, restoration after save failure, and restart failure after a successful file replacement.

  ```go
  func TestNezhaStartDoesNotChangeEnabledExpectation(t *testing.T) {
      runner := newFakeNezhaRunner("inactive", "disabled")
      svc := newNezhaAgentService(testPaths(t), runner, fakeSettings{})
      if err := svc.Operate("start"); err != nil { t.Fatal(err) }
      runner.assertCalled(t, "systemctl", "start", "xpanel-nezha-agent")
      runner.assertSettingNotWritten(t, "NezhaEnabled")
  }
  ```

- [ ] **Step 2: Confirm focused RED.**

  Run: `cd backend && go test ./app/service -run 'TestNezha(Start|Stop|Restart|Enable|Disable|Config)' -count=1`

  Expected: build failure because the service does not exist.

- [ ] **Step 3: Implement the focused service.** Constants are `/opt/xpanel/nezha-agent`, `/opt/xpanel/nezha-agent/nezha-agent`, `/opt/xpanel/nezha-agent/config.yml`, and `xpanel-nezha-agent`. Status returns component/config health, active/enabled, desired enabled, drift, binary version, UUID, canonical Dashboard/TLS state, secretConfigured, remoteOperationsEnabled, permissions warning, service error, and conflict details.

- [ ] **Step 4: Implement conflict detection without taking over external installations.** Detect active `nezha-agent.service` and instantiated units, non-bundled running executable paths, and common directories including `/opt/nezha/agent`. Report conflicts; never stop, overwrite, or import their UUID/config.

- [ ] **Step 5: Add failing redaction tests and extend the generic journal reader.** Redact the current AgentSecret, `client-secret`/`client_secret` metadata, Bearer/Authorization values, and secret assignments before returning journal text.

  ```go
  func TestNezhaJournalRedactionRemovesKnownSecretAndAuthorization(t *testing.T) {
      got := redactNezhaJournal("client_secret=secret Authorization: Bearer abc", "secret")
      if strings.Contains(got, "secret") || strings.Contains(got, "abc") { t.Fatal(got) }
  }
  ```

- [ ] **Step 6: Add the credential-free unit file.** It must use `Type=simple`, `WorkingDirectory=/opt/xpanel/nezha-agent`, direct `ExecStart=/opt/xpanel/nezha-agent/nezha-agent -c /opt/xpanel/nezha-agent/config.yml`, `Restart=always`, `RestartSec=10`, `UMask=0077`, and `WantedBy=multi-user.target`.

- [ ] **Step 7: Verify GREEN.**

  Run: `cd backend && go test ./app/service -run 'TestNezha|TestSystemd' -count=1`

  Run: `! rg -n 'client_secret|AgentSecret' scripts/xpanel-nezha-agent.service`

  Expected: tests pass and the unit contains no credential fields.

## Task 3: settings sync, API, startup integration, and security middleware

**Files:**

- Create: `backend/app/api/v1/nezha_agent.go`
- Create: `backend/app/api/v1/nezha_agent_test.go`
- Modify: `backend/app/api/v1/entry.go`
- Modify: `backend/router/router.go`
- Modify: `backend/server/server.go`
- Modify: `backend/middleware/operation_log_sanitize.go`
- Modify: `backend/middleware/operation_log_test.go`
- Modify: `backend/init/migration/migration.go`

- [ ] **Step 1: Add failing file-to-DB sync tests.** A valid file writes canonical `NezhaServer` and encrypted `NezhaClientSecret`; missing/corrupt config returns health information but never reconstructs the file from DB. Sync occurs at X-Panel startup, status read, pre-save, and after first successful Agent start.

- [ ] **Step 2: Add failing handler tests.** Cover authenticated `GET /api/v1/nezha-agent/status`, `PUT /api/v1/nezha-agent/config`, and `POST /api/v1/nezha-agent/operate`. Cover Agent journal access through the existing generic systemd-log API with the fixed bundled unit name. Assert response JSON and logs never contain the submitted or persisted secret and invalid Dashboard input causes no service command/file write.

- [ ] **Step 3: Confirm RED.**

  Run: `cd backend && go test ./app/api/v1 ./app/service -run 'TestNezha' -count=1`

  Expected: handlers/routes are absent or tests fail on missing behavior.

- [ ] **Step 4: Implement the API and route wiring.** Use existing JWT, NodeProxy, operation-log, validation, and response helpers. Allowed operations are exactly `start`, `stop`, `restart`, `enable`, and `disable`. Do not add a Nezha-specific log protocol; the frontend reads the fixed bundled unit through the existing generic systemd-log API.

- [ ] **Step 5: Add Agent startup sync after DB/credential initialization.** Failure logs a warning and does not block the panel. It must not start or enable the Agent.

- [ ] **Step 6: Mark `/api/v1/nezha-agent` as a sensitive operation-log path and keep `clientSecret` in JSON redaction.** Add middleware regression tests for plaintext and malformed inputs.

- [ ] **Step 7: Add `NezhaEnabled=false`, `NezhaServer=""`, and `NezhaClientSecret=""` defaults without adding any Fleet defaults.** This is temporary until Task 8 adds the retirement migration.

- [ ] **Step 8: Verify GREEN.**

  Run: `cd backend && go test ./app/api/v1 ./app/service ./middleware ./server -run 'Nezha|OperationLog' -count=1`

  Expected: selected tests pass; secret bytes are absent from API and captured log bodies.

## Task 4: pinned release supply chain and package layout

**Files:**

- Create: `scripts/package-nezha-agent.sh`
- Create: `scripts/test-package-nezha-agent.sh`
- Create: `third_party/nezha-agent/LICENSE`
- Create: `third_party/nezha-agent/NOTICE.md`
- Modify: `.github/workflows/release.yml`
- Modify: `Makefile`

- [ ] **Step 1: Write a failing offline shell test.** Generate fake amd64/arm64 zip assets and a `checksums.txt`; assert the helper rejects a wrong checksum, rejects an archive without `nezha-agent`, and stages only `nezha-agent/nezha-agent` plus checked-in license/notice with executable permissions.

- [ ] **Step 2: Confirm RED.**

  Run: `bash scripts/test-package-nezha-agent.sh`

  Expected: failure because `scripts/package-nezha-agent.sh` is absent.

- [ ] **Step 3: Implement the packaging helper.** Inputs are version, architecture, zip, checksum file, and release directory. It verifies the exact asset line with `sha256sum --check`, extracts into a temporary directory, confirms a regular executable named `nezha-agent`, and installs it as `release/nezha-agent/nezha-agent` with `0755`.

- [ ] **Step 4: Pin CI to v2.3.1.** For each matrix architecture, download the official zip and `checksums.txt`, run the helper, add `scripts/xpanel-nezha-agent.service`, `third_party/nezha-agent/LICENSE`, and `NOTICE.md` to the release package, then calculate the X-Panel tarball SHA256. Remove all `FLEET_ENDPOINT` manifest variables and fields while preserving the current safe Python environment-variable quoting.

- [ ] **Step 5: Make local `make package` require a caller-supplied verified Agent binary.** Use `NEZHA_AGENT_BINARY` and fail before creating a tarball when it is missing; copy it to the same release layout. Do not download “latest” from Makefile.

- [ ] **Step 6: Verify GREEN and static supply-chain rules.**

  Run: `bash scripts/test-package-nezha-agent.sh`

  Run: `rg -n 'NEZHA_AGENT_VERSION: v2.3.1|checksums.txt|package-nezha-agent.sh' .github/workflows/release.yml`

  Run: `! rg -n 'FLEET_ENDPOINT|manifest\["fleet"\]' .github/workflows/release.yml scripts/gen-version-json.sh`

  Expected: fixture tests pass and Fleet is absent from release manifests.

## Task 5: new installation, historical upgrade, and external-Agent protection

**Files:**

- Create: `scripts/test-nezha-agent-install.sh`
- Modify: `scripts/install.sh`
- Modify: `scripts/install-online.sh`
- Modify: `scripts/xpanel-nezha-agent.service`

- [ ] **Step 1: Add failing installer tests.** Cover no credentials (asset/unit installed, disabled/inactive, no config), paired environment credentials, `--nezha-dashboard` plus hidden interactive secret, non-interactive half-pair preflight failure before replacement, no secret CLI flag, no secret in output/argv capture/temp files, external Agent conflict, historical upgrade without config, and active/stopped/disabled state preservation.

- [ ] **Step 2: Confirm RED.**

  Run: `bash scripts/test-nezha-agent-install.sh`

  Expected: tests fail because Nezha installer behavior is absent.

- [ ] **Step 3: Implement input and preflight semantics.** Accept `--nezha-dashboard`, `XPANEL_NEZHA_DASHBOARD_URL`, and `XPANEL_NEZHA_AGENT_SECRET`; never accept a secret flag. Clear and `unset` the secret environment variable immediately after copying it into a shell variable. Validate paired inputs and HTTPS origin before stopping/copying any service.

- [ ] **Step 4: Install the bundled component.** Require `nezha-agent/nezha-agent` in the extracted package, create `/opt/xpanel/nezha-agent` as `0700`, copy binary `0755`, install the credential-free unit `0644`, and never overwrite an existing `config.yml` during an upgrade.

- [ ] **Step 5: Create initial config securely only for a complete credential pair.** Write through a `0600` temporary file in the component directory and atomic rename; include server `host:port`, secret, TLS on, insecure TLS off, both update disables on, and command execution allowed. Enable/start only after X-Panel starts. With no pair, leave the unit disabled/inactive.

- [ ] **Step 6: Preserve systemd actual state during upgrades and avoid external Agent takeover.** A detected external unit/process/path produces a warning and skips bundled Agent start without failing X-Panel install. Upgrade captures bundled Agent active/enabled state, stops only if active, replaces only the binary/unit, daemon-reloads, and restores only the previous active state.

- [ ] **Step 7: Verify GREEN.**

  Run: `bash scripts/test-nezha-agent-install.sh`

  Run: `bash scripts/test-xpctl.sh && bash scripts/test-install-xpctl.sh`

  Expected: all installer/control tests pass and fixture output contains no test secret.

## Task 6: component-package in-panel upgrade with rollback

**Files:**

- Create: `backend/app/service/component_upgrade.go`
- Create: `backend/app/service/component_upgrade_test.go`
- Modify: `backend/app/service/upgrade.go`
- Modify: `backend/app/service/upgrade_test.go`
- Modify: `backend/app/dto/upgrade.go`
- Modify: `frontend/src/api/interface/index.ts`
- Modify: `frontend/src/api/modules/upgrade.ts`
- Modify: `frontend/src/views/setting/index.vue`

- [ ] **Step 1: Add failing preflight tests.** A package missing X-Panel, missing Agent, containing a symlink/non-regular asset, using the wrong ELF architecture, or failing/missing package checksum must abort before any live binary or service state changes.

- [ ] **Step 2: Add failing transaction tests.** Cover active Agent stop/replace/restart, stopped-but-enabled preservation, disabled preservation, config byte-for-byte preservation, X-Panel replacement failure rolling back Agent, Agent replacement failure leaving X-Panel unchanged, and Agent restart failure rolling back both binaries while retaining config.

- [ ] **Step 3: Confirm RED.**

  Run: `cd backend && go test ./app/service -run 'TestComponentUpgrade|TestUpgrade' -count=1`

  Expected: current single-binary upgrader fails component/rollback assertions.

- [ ] **Step 4: Implement safe archive extraction and preflight with standard library.** Extract only `xpanel` and `nezha-agent/nezha-agent` from gzip/tar into a temp directory; reject traversal, links, duplicates, missing files, non-regular entries, and ELF architecture mismatches. A checksum URL is mandatory for component upgrades.

- [ ] **Step 5: Implement the two-binary transaction.** Capture actual Agent active/enabled state, stop only if active, create separate backups, stage same-directory `.new` files, replace Agent then X-Panel, roll back every already-replaced binary on any failure, never touch `config.yml`, restore Agent active state, and finally initiate X-Panel restart.

- [ ] **Step 6: Remove Fleet manifest DTO/UI plumbing while preserving normal upgrade fields and existing dirty-file work.** `UpgradeInfo` and `UpgradeReq` no longer contain `fleetEndpoint`; self-hosted manifests no longer parse a Fleet block.

- [ ] **Step 7: Verify GREEN.**

  Run: `cd backend && go test ./app/service -run 'TestComponentUpgrade|TestUpgrade' -count=1`

  Run: `cd frontend && npm run build`

  Expected: rollback/state tests and TypeScript production build pass.

## Task 7: Nezha Agent management page

**Files:**

- Create: `frontend/src/api/modules/nezha-agent.ts`
- Create: `frontend/src/routers/modules/nezha-agent.ts`
- Create: `frontend/src/views/nezha-agent/state.ts`
- Create: `frontend/src/views/nezha-agent/state.test.ts`
- Create: `frontend/src/views/nezha-agent/index.vue`
- Modify: `frontend/src/routers/index.ts`
- Modify: `frontend/src/layout/components/Sidebar.vue`
- Modify: `frontend/src/i18n/zh.ts`

- [ ] **Step 1: Add failing pure state tests with Node 24.** Cover pending configuration, stopped, running, corrupt config, external conflict, service failure, drift, secret always blank in the edit form, and the different labels/actions for temporary stop versus complete disable.

  ```ts
  test('external conflict blocks bundled-agent start', () => {
    const vm = deriveNezhaView(baseStatus({ conflict: { blocked: true, message: 'external unit' } }))
    assert.equal(vm.canStart, false)
    assert.equal(vm.kind, 'conflict')
  })
  ```

- [ ] **Step 2: Confirm RED.**

  Run: `node --test frontend/src/views/nezha-agent/state.test.ts`

  Expected: module/function missing.

- [ ] **Step 3: Implement the API module and typed state derivation.** The secret field is write-only and initialized to an empty string on every dialog open; status uses only `secretConfigured`.

- [ ] **Step 4: Build one utilitarian component-management page consistent with X-Panel's existing GOST/host cards.** Show component/config/systemd status, version, UUID, Dashboard/TLS, remote-operations permission, drift/conflict/error alerts, and actions for configure/start/stop/restart/enable/disable/logs. Use the existing theme variables and Element Plus; add no dependency, font, node-control plane, chart, terminal, or independent update action.

- [ ] **Step 5: Add explicit safety copy.** Before disabling remote operations, explain that Dashboard command, terminal, file, ApplyConfig, and key rotation stop and can only be re-enabled locally. “停止” changes runtime only; “完全禁用” performs disable-and-stop while retaining UUID/config.

- [ ] **Step 6: Add a single sidebar entry and route.** Do not create a Nezha submenu or node list.

- [ ] **Step 7: Verify GREEN.**

  Run: `node --test frontend/src/views/nezha-agent/state.test.ts`

  Run: `cd frontend && npm run build`

  Expected: state tests and production build pass with no new dependency.

## Task 8: Fleet retirement migration and active-code removal

**Files:**

- Modify: `backend/init/migration/migration.go`
- Replace test: `backend/init/migration/fleet_v2_only_migration_test.go` with `backend/init/migration/fleet_retirement_migration_test.go`
- Modify: `backend/security/credentials/registry.go`
- Modify: `backend/security/credentials/database_test.go`
- Modify: `backend/app/dto/setting.go`
- Modify: `backend/app/service/setting.go`
- Modify: `backend/app/service/notification.go`
- Modify: `backend/init/cron/cron.go`
- Modify: `backend/server/server.go`
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/cmd/server/main_test.go`
- Modify: `scripts/gen-version-json.sh`
- Modify: `scripts/install-online.sh`
- Modify: `.github/workflows/release.yml`
- Archive: `docs/fleet-reporter.md` to `docs/archive/fleet-reporter.md`
- Remove: Fleet-only Go/service/CLI/test/testdata files and `scripts/test-fleet-bootstrap-args.sh`

- [ ] **Step 1: Add a failing idempotent retirement migration test.** Seed every `Fleet*` setting named in the design plus an unknown future `FleetUnexpectedKey`, run migration, and assert zero rows match `key LIKE 'Fleet%'`; run again and assert success. Assert `Nezha*` settings and unrelated encrypted settings survive.

  ```go
  func TestFleetRetirementMigrationDeletesEveryFleetSettingIdempotently(t *testing.T) {
      seedSettings(t, "FleetEndpoint", "FleetUnexpectedKey", "NezhaServer", "GitHubToken")
      if err := migrateFleetRetirementSettings(); err != nil { t.Fatal(err) }
      assertSettingPrefixCount(t, "Fleet", 0)
      if err := migrateFleetRetirementSettings(); err != nil { t.Fatal(err) }
  }
  ```

- [ ] **Step 2: Confirm RED before deleting production Fleet code.**

  Run: `cd backend && go test ./init/migration -run 'TestFleetRetirement' -count=1`

  Expected: migration missing or Fleet rows remain.

- [ ] **Step 3: Implement one raw, transactional retirement migration.** Before default-setting insertion, delete all rows whose key begins with `Fleet`, then write non-Fleet marker `_mig_fleet_retirement_cleanup=done`. Do not decrypt, reuse, or convert Fleet IDs/keys. Remove all Fleet defaults and all Fleet entries from the sensitive-field registry only after the raw deletion logic exists.

- [ ] **Step 4: Remove active Fleet entry points surgically.** Remove Reporter startup, notification forwarding/comment, Fleet auto-upgrade override/release URL, Settings DTO/update keys, `fleet-enroll`/`fleet-recover` CLI dispatch, Fleet update-manifest fields, installer arguments/env/bootstrap code, Fleet release variables, Fleet DTOs/services/tests/dependencies used only by Fleet, and frontend upgrade hints.

- [ ] **Step 5: Archive rather than continue Fleet Center.** Preserve the independent `fleet-center/` source tree and its dirty worktree. Add a concise deprecation banner to its README and append an archive-only record to `docs/v2/WORKLOG.md`; do not alter code, build files, APIs, or deployment behavior. Move the X-Panel Fleet Reporter document under `docs/archive/` with a deprecation banner.

- [ ] **Step 6: Remove the now-unused Fleet source/test files only after explicit destructive-action confirmation.** Use file-targeted patches, never recursive deletion. Keep all non-Fleet user changes in shared dirty files.

- [ ] **Step 7: Verify Fleet is absent from active paths.**

  Run: `! rg -n -i 'fleet[-_ ]?center|fleetreporter|fleetenrollment|fleet[_A-Z]' backend frontend scripts .github Makefile README.md -g '!backend/cmd/server/web/assets/**' -g '!frontend/dist/**'`

  Run: `cd backend && go test ./...`

  Run: `cd frontend && npm run build`

  Expected: no active Fleet match; full backend tests and frontend build pass.

## Task 9: operator documentation and end-to-end acceptance

**Files:**

- Create: `docs/nezha-agent.md`
- Modify: `README.md`
- Modify: `docs/superpowers/specs/2026-08-09-xpanel-bundled-nezha-agent-design.md`

- [ ] **Step 1: Document install and operations without leaking secrets.** Human instructions use hidden `read -rsp`, exported secret variables, `sudo --preserve-env`, and immediate `unset`; do not show inline environment assignment or secret command arguments. Explain root-level Dashboard trust, TLS-only input, remote-operations consequences, UUID preservation, external conflict handling, service actions, and version-bound updates.

- [ ] **Step 2: Update current architecture/product docs.** Mark the design “approved/implemented” only after verification, remove Fleet as a current X-Panel component, and link the archived historical material separately.

- [ ] **Step 3: Run static secret and product-path scans.**

  Run: `rg -n 'XPANEL_NEZHA_AGENT_SECRET|client_secret|NezhaClientSecret' docs README.md scripts .github backend frontend`

  Inspect every match to ensure it is a variable/key/test fixture, never a real credential or unsafe command example.

- [ ] **Step 4: Run the complete local verification matrix.**

  Run: `cd backend && gofmt -w app/dto/nezha_agent.go app/service/nezha_agent_config.go app/service/nezha_agent_config_test.go app/service/nezha_agent.go app/service/nezha_agent_test.go app/service/systemd_service.go app/api/v1/nezha_agent.go app/api/v1/nezha_agent_test.go app/api/v1/entry.go router/router.go server/server.go middleware/operation_log_sanitize.go middleware/operation_log_test.go init/migration/migration.go security/credentials/registry.go security/credentials/database_test.go app/repo/secure_fields_test.go app/repo/setting.go app/service/component_upgrade.go app/service/component_upgrade_test.go app/service/upgrade.go app/service/upgrade_test.go app/dto/upgrade.go app/dto/setting.go app/service/setting.go app/service/notification.go init/cron/cron.go cmd/server/main.go cmd/server/main_test.go init/migration/fleet_retirement_migration_test.go && go test ./...`

  Run: `node --test frontend/src/views/nezha-agent/state.test.ts && cd frontend && npm run build`

  Run: `bash scripts/test-package-nezha-agent.sh && bash scripts/test-nezha-agent-install.sh && bash scripts/test-xpctl.sh && bash scripts/test-install-xpctl.sh`

  Run: `git diff --check`

  Expected: every command exits 0; frontend may retain only the pre-existing large-chunk warning.

- [ ] **Step 5: Run Linux/systemd acceptance where available.** Validate no-credential install, complete-credential install, UUID generation, Dashboard registration within 30 seconds, ApplyConfig reflection, key rotation reflection, UUID/unknown-field preservation, remote-operations disable, independent active/enabled operations, external conflict, and active/stopped/disabled component upgrades. If the current macOS workspace lacks the Linux/systemd/Dashboard runtime, report these scenarios as unexecuted environmental acceptance checks rather than claiming them passed.

- [ ] **Step 6: Final independent review.** Compare the final diff line-by-line with all 19 design sections and this plan, verify `fleet-center/` contains no development changes beyond archive markers, verify no unrelated dirty change was lost, and report exact tests plus any remaining environment-only acceptance work.
