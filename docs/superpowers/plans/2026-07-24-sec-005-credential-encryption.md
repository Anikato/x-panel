# SEC-005 Credential Encryption Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Encrypt X-Panel recoverable credentials with a separately stored, versioned root-only keyring while preserving direct upgrades, backups, restores, Fleet Center, and existing business behavior.

**Architecture:** A focused `security/credentials` package owns AES-256-GCM envelope encryption, keyring persistence, the sensitive-field registry, database migration, validation, and rotation. Repositories explicitly protect registered values before writes and reveal them after reads; startup initializes the keyring and finishes the idempotent migration before any background worker or HTTP route starts.

**Tech Stack:** Go 1.26 standard-library cryptography, GORM, glebarez SQLite, Bash installer/xpctl tests, Vue 3/TypeScript, Vitest where applicable.

---

## File map

### New backend files

- `backend/security/credentials/envelope.go`: versioned AES-GCM envelope encoding and parsing.
- `backend/security/credentials/envelope_test.go`: cryptographic round-trip, AAD, nonce, tamper, and malformed-input tests.
- `backend/security/credentials/keyring.go`: root-only keyring creation, loading, persistence, and active KEK rotation.
- `backend/security/credentials/keyring_test.go`: keyring format, permissions, symlink, corruption, and rotation tests.
- `backend/security/credentials/registry.go`: single source of truth for table columns, setting keys, scopes, and repository field maps.
- `backend/security/credentials/database.go`: state scan, snapshot, migration, scrub, validation, and re-encryption.
- `backend/security/credentials/database_test.go`: legacy, mixed, encrypted, rollback, scrub, missing-key, and rotation tests.
- `backend/app/repo/secure_fields.go`: small repository helpers for struct fields and map updates.
- `backend/app/repo/secure_fields_test.go`: helper and representative repository persistence tests.
- `backend/init/permission/permission.go`: non-recursive X-Panel-owned path hardening.
- `backend/init/permission/permission_test.go`: mode repair and unsafe-path rejection.
- `backend/init/credential/credential.go`: startup keyring and database state orchestration.
- `backend/init/credential/credential_test.go`: first-upgrade state-machine tests.
- `backend/app/service/credential_security.go`: authenticated online key rotation service.
- `backend/app/service/credential_security_test.go`: rotation and retry behavior.

### Modified backend files

- `backend/global/global.go`: credential protector interface/global and `credential_key_path`.
- `backend/init/viper/viper.go`: derive the old-config-compatible default keyring path.
- `backend/init/db/db.go`: `0700` DB directories and `0600` SQLite/WAL/SHM files.
- `backend/init/log/log.go`: `0700` log directory and `0600` log file.
- `backend/server/server.go`: initialize permissions and credentials before workers.
- `backend/cmd/server/setup.go`: initialize credentials in offline setup.
- `backend/cmd/server/fleet_enroll.go`: protect Fleet Enrollment Token before direct DB write.
- `backend/cmd/server/main.go`: support the constrained secret-initialization/verification commands required by installer and xpctl.
- `backend/app/repo/{host,node,backup,database,ssl,cert_sync,website,gost,cronjob,haproxy,setting}.go`: explicit protect/reveal calls.
- `backend/app/model/{host,node,backup,database,ssl,cert_sync,website,gost,cronjob}.go`: prevent secret JSON serialization.
- `backend/app/service/{setting,cronjob}.go`: secret-presence response fields and empty-on-update semantics.
- `backend/app/dto/setting.go`: `githubTokenSet` and `agentTokenSet`.
- `backend/app/api/v1/setting.go`: authenticated rotation handler.
- `backend/router/router.go`: private rotation endpoint.

### Modified operational/frontend files

- `scripts/install-online.sh`: `umask 077`, explicit modes, key path, pre-start secret writes through X-Panel.
- `scripts/xpanel.service`: `UMask=0077`.
- `scripts/xpctl`: pre-restore credential/key compatibility verification.
- `scripts/test-install-xpctl.sh`: mode and restore-gate regression cases.
- `frontend/src/api/interface/index.ts`: secret-presence response types.
- `frontend/src/views/setting/index.vue`: do not repopulate GitHub/Agent Token plaintext.
- `docs/security-audit-remediation-plan-2026-07-24.md`: SEC-005 status/evidence.

No Git commit is created by this plan because repository instructions prohibit unrequested commits.

### Task 1: AES-GCM envelope primitive

**Files:**
- Create: `backend/security/credentials/envelope.go`
- Create: `backend/security/credentials/envelope_test.go`

- [x] **Step 1: Write failing envelope tests**

Create table-driven tests with these exact assertions:

```go
func TestEnvelopeRoundTripAndRandomNonce(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)
	first, err := sealEnvelope("kek-test", key, "hosts.password", "secret")
	require.NoError(t, err)
	second, err := sealEnvelope("kek-test", key, "hosts.password", "secret")
	require.NoError(t, err)
	require.NotEqual(t, first, second)
	require.Equal(t, "secret", mustOpen(t, key, "hosts.password", first))
}

func TestEnvelopeRejectsWrongScopeAndTampering(t *testing.T) {
	value := mustSeal(t, "kek-test", testKey(), "hosts.password", "secret")
	_, err := openEnvelope(testKey(), "nodes.token", value)
	require.Error(t, err)
	tampered := value[:len(value)-1] + "A"
	_, err = openEnvelope(testKey(), "hosts.password", tampered)
	require.Error(t, err)
}
```

Also cover empty/malformed segments, unknown `v2`, invalid Base64, short nonce, and a 16-byte KEK rejection.

- [x] **Step 2: Run the tests and confirm RED**

Run:

```bash
cd backend
go test ./security/credentials -run 'TestEnvelope' -v
```

Expected: build failure because `sealEnvelope` and `openEnvelope` do not exist.

- [x] **Step 3: Implement the versioned envelope**

Implement:

```go
const envelopePrefix = "xpanel:enc:v1:"

type parsedEnvelope struct {
	KeyID      string
	WrappedDEK []byte
	Ciphertext []byte
}

func sealEnvelope(keyID string, kek []byte, scope, plaintext string) (string, error)
func openEnvelope(kek []byte, scope, encoded string) (string, error)
func parseEnvelope(encoded string) (parsedEnvelope, error)
func IsEncrypted(value string) bool
func EnvelopeKeyID(value string) (string, error)
```

Use a fresh 32-byte DEK, `aes.NewCipher`, `cipher.NewGCM`, two independently generated standard-size nonces, and `base64.RawURLEncoding`. Use `xpanel:dek:v1:<key-id>` as KEK-layer AAD and `xpanel:data:v1:<scope>` as data-layer AAD.

- [x] **Step 4: Run focused tests and confirm GREEN**

Run:

```bash
cd backend
go test ./security/credentials -run 'TestEnvelope' -v
go test -race ./security/credentials -run 'TestEnvelope'
```

Expected: both commands pass.

### Task 2: Root-only keyring and protector

**Files:**
- Create: `backend/security/credentials/keyring.go`
- Create: `backend/security/credentials/keyring_test.go`
- Modify: `backend/global/global.go`

- [x] **Step 1: Write failing keyring tests**

Cover these public behaviors:

```go
func TestLoadOrCreateKeyringUsesRestrictiveModes(t *testing.T)
func TestLoadOrCreateKeyringDoesNotReplaceExistingKey(t *testing.T)
func TestLoadKeyringRejectsSymlinkAndMalformedJSON(t *testing.T)
func TestManagerProtectRevealAndRotation(t *testing.T)
```

The first test must assert directory mode `0700`, file mode `0600`, JSON `version == 1`, one 32-byte decoded key, and the decoded key ID equals `activeKeyId`. The second must load twice and assert the active key and ciphertext remain decryptable.

- [x] **Step 2: Run the tests and confirm RED**

```bash
cd backend
go test ./security/credentials -run 'TestLoad|TestManager' -v
```

Expected: build failure because `LoadOrCreate` and `Manager` are undefined.

- [x] **Step 3: Implement keyring persistence and global contract**

Add to `global/global.go`:

```go
type CredentialProtector interface {
	Protect(scope, value string) (string, error)
	Reveal(scope, value string) (string, error)
	Validate(scope, value string) error
	IsEncrypted(value string) bool
	ActiveKeyID() string
	AddActiveKey() (string, error)
	KeyIDs() []string
}

var CREDENTIALS CredentialProtector
```

Implement `Manager`:

```go
type Manager struct {
	mu   sync.RWMutex
	path string
	doc  keyringDocument
}

func LoadOrCreate(path string, allowCreate bool) (*Manager, bool, error)
func (m *Manager) Protect(scope, value string) (string, error)
func (m *Manager) Reveal(scope, value string) (string, error)
func (m *Manager) Validate(scope, value string) error
func (m *Manager) IsEncrypted(value string) bool
func (m *Manager) ActiveKeyID() string
func (m *Manager) AddActiveKey() (string, error)
func (m *Manager) KeyIDs() []string
```

`Protect` keeps a valid envelope unchanged when it already uses the active KEK, re-encrypts a valid envelope using an older KEK, and encrypts plaintext. `Reveal` accepts legacy plaintext for migration compatibility but never falls back when the value starts with the encrypted prefix. Persist keyring changes atomically with restrictive modes and `fsync`.

- [x] **Step 4: Run focused tests and confirm GREEN**

```bash
cd backend
go test ./security/credentials -run 'TestLoad|TestManager' -v
go test -race ./security/credentials
```

Expected: pass with no races.

### Task 3: Sensitive registry and SQLite migration

**Files:**
- Create: `backend/security/credentials/registry.go`
- Create: `backend/security/credentials/database.go`
- Create: `backend/security/credentials/database_test.go`

- [x] **Step 1: Write failing registry and state tests**

Build a temp SQLite database containing every table in the approved design and unique sentinels. Assert:

```go
state, err := ScanDatabase(db)
require.NoError(t, err)
require.True(t, state.HasPlaintext)
require.False(t, state.HasEncrypted)

require.NoError(t, MigrateDatabase(db, manager, backupDir))
state, err = ScanDatabase(db)
require.NoError(t, err)
require.False(t, state.HasPlaintext)
require.True(t, state.HasEncrypted)
require.NoError(t, ValidateDatabase(db, manager))
```

Direct SQL reads must not contain any sentinel after success, while repository-independent `Reveal` calls recover all originals. Add tests for mixed values, unknown KEK, tampered ciphertext, transaction rollback, successful temporary-snapshot deletion, failed-migration snapshot retention, and idempotent second migration.

- [x] **Step 2: Run database tests and confirm RED**

```bash
cd backend
go test ./security/credentials -run 'TestDatabase|TestRegistry|TestMigrate' -v
```

Expected: build failure because the registry and migration functions are undefined.

- [x] **Step 3: Implement one registry used by migration and repositories**

Define:

```go
type FieldSpec struct {
	Table  string
	Column string
	Scope  string
}

var FieldSpecs = []FieldSpec{
	{Table: "hosts", Column: "password", Scope: "hosts.password"},
	// Include every table/column from the approved design.
}

var SecretSettingKeys = map[string]struct{}{
	"MFASecret": {}, "GitHubToken": {}, "AgentToken": {},
	"CertServerToken": {}, "FleetInstanceToken": {},
	"FleetEnrollmentToken": {}, "HAProxyStatsPass": {},
	"GostAPIPass": {}, "ProxyAddress": {}, "SecurityEntrance": {},
}

func IsSecretSetting(key string) bool
func SettingScope(key string) string
func ScopeFor(table, column string) (string, bool)
```

Identifiers come only from these constants and must never accept caller-controlled SQL identifiers.

- [x] **Step 4: Implement transactional migration and validation**

Implement:

```go
type DatabaseState struct {
	HasPlaintext bool
	HasEncrypted bool
	KeyIDs       map[string]struct{}
}

func ScanDatabase(db *gorm.DB) (DatabaseState, error)
func MigrateDatabase(db *gorm.DB, protector global.CredentialProtector, backupDir string) error
func ValidateDatabase(db *gorm.DB, protector global.CredentialProtector) error
func ReencryptDatabase(db *gorm.DB, protector global.CredentialProtector) error
```

Use `VACUUM INTO` for a consistent `0600` migration snapshot before the transaction. Set `PRAGMA secure_delete=ON`, commit only after all values are protected and validated, then run `wal_checkpoint(TRUNCATE)` and `VACUUM`. Write the completion marker only after scrubbing succeeds. Remove the temporary snapshot only after full success.

- [x] **Step 5: Run focused tests and scan sentinels**

```bash
cd backend
go test ./security/credentials -run 'TestDatabase|TestRegistry|TestMigrate' -v
go test -race ./security/credentials
```

Expected: all tests pass and sentinel byte scans report zero matches in DB/WAL/SHM after migration.

### Task 4: Startup state machine and compatible key path

**Files:**
- Create: `backend/init/credential/credential.go`
- Create: `backend/init/credential/credential_test.go`
- Modify: `backend/global/global.go`
- Modify: `backend/init/viper/viper.go`
- Modify: `backend/server/server.go`
- Modify: `backend/cmd/server/setup.go`

- [x] **Step 1: Write failing state-machine tests**

Test:

```go
func TestInitCreatesKeyAndMigratesLegacyDatabase(t *testing.T)
func TestInitLoadsExistingKeyForEncryptedDatabase(t *testing.T)
func TestInitRejectsEncryptedDatabaseWithoutKey(t *testing.T)
func TestInitRejectsWrongKeyAndCorruptCiphertext(t *testing.T)
func TestDefaultCredentialKeyPathFollowsCustomDataDir(t *testing.T)
```

For an old config with `data_dir: /srv/custom/data` and no key path, expect `/srv/custom/secrets/credential-keyring.json`.

- [x] **Step 2: Run tests and confirm RED**

```bash
cd backend
go test ./init/credential ./init/viper -run 'TestInit|TestDefaultCredential' -v
```

Expected: missing package/functions or wrong default path.

- [x] **Step 3: Implement initialization ordering**

Add:

```go
type SystemConfig struct {
	// existing fields
	CredentialKeyPath string `mapstructure:"credential_key_path"`
}
```

Implement `credential.Init()` to scan first, call `LoadOrCreate(path, !state.HasEncrypted)`, assign `global.CREDENTIALS`, run migration when needed, and always validate before returning. Panic at server startup with a concise actionable error; return errors from testable internal functions.

Call it after structural/default migrations but before proxy sync, Cron, node heartbeat, Fleet Reporter, GOST sync, or HTTP startup. Use the same initializer in offline setup and Fleet enrollment.

- [x] **Step 4: Run state tests and existing server command tests**

```bash
cd backend
go test ./init/credential ./init/viper ./cmd/server ./server -v
```

Expected: pass.

### Task 5: Repository protection helpers and core credentials

**Files:**
- Create: `backend/app/repo/secure_fields.go`
- Create: `backend/app/repo/secure_fields_test.go`
- Modify: `backend/app/repo/host.go`
- Modify: `backend/app/repo/node.go`
- Modify: `backend/app/repo/backup.go`
- Modify: `backend/app/repo/database.go`

- [x] **Step 1: Write failing helper and repository tests**

Define representative tests for Host, Node, BackupAccount, DatabaseServer, and DatabaseInstance:

```go
require.NoError(t, repo.Create(&model.Host{Password: sentinel}))
var raw string
require.NoError(t, db.Table("hosts").Select("password").Scan(&raw).Error)
require.NotContains(t, raw, sentinel)
stored, err := repo.Get(repo.WithByID(id))
require.NoError(t, err)
require.Equal(t, sentinel, stored.Password)
```

For map updates, assert the caller map is not mutated and only registered keys are encrypted.

- [x] **Step 2: Run tests and confirm RED**

```bash
cd backend
go test ./app/repo -run 'TestSecure|TestHostCredentials|TestNodeCredentials|TestBackupCredentials|TestDatabaseCredentials' -v
```

Expected: sentinels remain plaintext.

- [x] **Step 3: Implement reusable explicit helpers**

Implement:

```go
type secureField struct {
	Scope string
	Value *string
}

func protectFields(fields ...secureField) error
func revealFields(fields ...secureField) error
func protectUpdates(table string, updates map[string]interface{}) (map[string]interface{}, error)
```

`protectUpdates` returns a new map. All helpers fail if `global.CREDENTIALS` is unavailable and a non-empty registered secret is encountered.

- [x] **Step 4: Integrate core repositories**

For Create/Save, encrypt a struct copy so the caller still receives plaintext plus generated ID/timestamps. For Get/List/Page, reveal every registered field before returning. For Update, pass the cloned protected map to GORM.

- [x] **Step 5: Run focused repository tests**

```bash
cd backend
go test ./app/repo -run 'TestSecure|TestHostCredentials|TestNodeCredentials|TestBackupCredentials|TestDatabaseCredentials' -v
go test -race ./app/repo
```

Expected: pass with no plaintext direct SQL values.

### Task 6: Remaining repositories and conditional settings

**Files:**
- Modify: `backend/app/repo/ssl.go`
- Modify: `backend/app/repo/cert_sync.go`
- Modify: `backend/app/repo/website.go`
- Modify: `backend/app/repo/gost.go`
- Modify: `backend/app/repo/cronjob.go`
- Modify: `backend/app/repo/haproxy.go`
- Modify: `backend/app/repo/setting.go`
- Extend: `backend/app/repo/secure_fields_test.go`

- [x] **Step 1: Add failing coverage for every remaining registered field**

Use subtests named after each scope. Each subtest writes a unique sentinel through Create/Update, asserts direct SQL has only an envelope, and asserts Get/List/Page returns the sentinel.

Settings tests must assert:

```go
requireEncryptedSetting(t, "FleetInstanceToken")
requireEncryptedSetting(t, "ProxyAddress")
requirePlainSetting(t, "PanelName")
requirePlainSetting(t, "Password") // bcrypt hash is already non-recoverable
```

- [x] **Step 2: Run the expanded repository suite and confirm RED**

```bash
cd backend
go test ./app/repo -run 'TestRegisteredCredentialFields|TestSettingEncryptionPolicy' -v
```

Expected: direct SQL contains plaintext for unintegrated repositories.

- [x] **Step 3: Integrate all remaining repositories**

Apply the same explicit copy/protect/reveal pattern. For `settings`, determine the scope from the row Key on Create/Get/List and from the method Key on Update/CreateOrUpdate. Do not encrypt `Password`.

- [x] **Step 4: Eliminate direct secret writes**

Change `saveFleetEnrollmentToken` to protect `settings.FleetEnrollmentToken`. Add a constrained offline command that accepts only approved bootstrap keys and reads values from environment variables; use it for installer `SecurityEntrance` and `AgentToken` instead of raw `sqlite3 UPDATE`.

- [x] **Step 5: Run repository, service, and command tests**

```bash
cd backend
go test ./app/repo ./app/service ./cmd/server -v
```

Expected: pass.

### Task 7: Secret response redaction and update semantics

**Files:**
- Modify: `backend/app/model/node.go`
- Modify: `backend/app/model/backup.go`
- Modify: `backend/app/model/database.go`
- Modify: `backend/app/model/ssl.go`
- Modify: `backend/app/model/cronjob.go`
- Modify: `backend/app/dto/setting.go`
- Modify: `backend/app/service/setting.go`
- Modify: `backend/app/service/cronjob.go`
- Modify: `frontend/src/api/interface/index.ts`
- Modify: `frontend/src/views/setting/index.vue`
- Create or extend relevant backend/frontend tests

- [x] **Step 1: Write failing serialization and setting-response tests**

Marshal models populated with unique secrets and assert JSON does not contain any secret. For `GetSettingInfo`, assert token plaintext is absent while `GitHubTokenSet` and `AgentTokenSet` are true.

- [x] **Step 2: Run tests and confirm RED**

```bash
cd backend
go test ./app/model ./app/service -run 'Test.*Secret|TestSettingInfo' -v
```

Expected: Node, DatabaseServer, Cronjob, or settings responses expose a sentinel.

- [x] **Step 3: Redact response models and preserve edits**

Use `json:"-"` on recoverable secret model fields. Add:

```go
GitHubTokenSet bool `json:"githubTokenSet"`
AgentTokenSet  bool `json:"agentTokenSet"`
```

Do not populate token plaintext in `SettingInfo`. Change Cron update so an empty `EncryptPassword` retains the existing encrypted password; provide an explicit clear action only if an existing API already distinguishes clear from unchanged.

- [x] **Step 4: Update the settings UI**

Do not repopulate token fields from GET responses. Show a localized “configured; leave blank to keep unchanged” hint from the boolean flags. Submitting a non-empty value replaces the secret.

- [x] **Step 5: Run backend and frontend checks**

```bash
cd backend
go test ./app/model ./app/service -v
cd ../frontend
pnpm run type-check
pnpm run build
```

Expected: all pass.

### Task 8: Filesystem permission hardening

**Files:**
- Create: `backend/init/permission/permission.go`
- Create: `backend/init/permission/permission_test.go`
- Modify: `backend/init/db/db.go`
- Modify: `backend/init/log/log.go`
- Modify: `backend/server/server.go`
- Modify: `backend/cmd/server/setup.go`
- Modify: `scripts/install-online.sh`
- Modify: `scripts/xpanel.service`

- [x] **Step 1: Write failing permission tests**

Create temp config, DB, WAL, SHM, log, key, and directory paths with modes `0755/0644`. Assert hardening changes only registered X-Panel-owned paths to `0700/0600`, rejects symlink targets, and leaves an external certificate path untouched.

- [x] **Step 2: Run tests and confirm RED**

```bash
cd backend
go test ./init/permission ./init/db ./init/log -run 'Test.*Permission|Test.*Mode' -v
```

Expected: missing package or over-broad modes.

- [x] **Step 3: Implement non-recursive hardening**

Set process umask `0077` before creating runtime files. Use `Lstat`; accept only regular files/directories; use `MkdirAll(..., 0700)` followed by explicit `Chmod`. Refuse unsafe types and return path-specific errors.

Change DB/log creation modes and explicitly harden SQLite, WAL, SHM, and `xpanel.log` after creation.

- [x] **Step 4: Harden installer/systemd**

At the start of `install-online.sh`, add `umask 077`. Create data, db, log, secrets, and backup directories with explicit modes. Write `credential_key_path` to new configs, preserve old configs on upgrade, and add `UMask=0077` to both generated and packaged systemd units.

- [x] **Step 5: Run Go and shell regression tests**

```bash
cd backend
go test ./init/permission ./init/db ./init/log -v
cd ..
bash -n scripts/install-online.sh
bash -n scripts/xpanel.service 2>/dev/null || true
```

Expected: Go tests and installer syntax pass.

### Task 9: Restore compatibility gate and online rotation

**Files:**
- Modify: `backend/app/service/credential_security.go`
- Modify: `backend/app/service/credential_security_test.go`
- Modify: `backend/app/api/v1/setting.go`
- Modify: `backend/router/router.go`
- Modify: `backend/cmd/server/main.go`
- Create: `backend/cmd/server/credentials.go`
- Create: `backend/cmd/server/credentials_test.go`
- Modify: `scripts/xpctl`
- Modify: `scripts/test-install-xpctl.sh`

- [x] **Step 1: Write failing rotation tests**

Seed old-key ciphertext, invoke the service, and assert:

```go
oldID := manager.ActiveKeyID()
newID, err := service.RotateCredentialKey()
require.NoError(t, err)
require.NotEqual(t, oldID, newID)
require.NoError(t, credentials.ValidateDatabase(db, manager))
requireAllEnvelopesUseKey(t, db, newID)
require.Contains(t, manager.KeyIDs(), oldID)
```

Inject a migration failure and assert the old and new keys remain, old ciphertext still decrypts, and retry succeeds.

- [x] **Step 2: Write failing xpctl restore-gate tests**

Extend the fake X-Panel binary in `scripts/test-install-xpctl.sh` to record:

```text
credentials verify --db <candidate>
```

Assert restore calls verification before copying, a verification failure leaves the current DB byte-identical, and a successful verification keeps the existing backup/integrity behavior.

- [x] **Step 3: Implement authenticated online rotation**

Add `POST /api/v1/settings/credentials/rotate` inside the existing private JWT group. Serialize rotations with a process mutex. Call `AddActiveKey`, then `ReencryptDatabase`; never delete old KEKs. Return only the new key ID and record an operation log without secrets.

- [x] **Step 4: Implement read-only credential verification command**

Add:

```text
xpanel credentials verify --db <path>
```

The command loads config and the existing keyring, opens the candidate database without running migrations, and calls `ValidateDatabase`. It prints no values or ciphertext.

- [x] **Step 5: Add the xpctl pre-replacement gate**

In `validate_restore_file`, after SQLite integrity passes, run:

```bash
"$INSTALL_DIR/xpanel" credentials verify --db "$backup_file"
```

Abort before `backup_db`, temporary copies, or replacement if verification fails.

- [x] **Step 6: Run rotation and rescue tests**

```bash
cd backend
go test ./app/service ./app/api/v1 ./cmd/server -run 'Test.*Credential|Test.*Rotate' -v
cd ..
bash scripts/test-install-xpctl.sh
```

Expected: pass.

### Task 10: Full upgrade, sentinel, and documentation verification

**Files:**
- Create or extend integration tests under `backend/init/credential` and `scripts/`
- Modify: `docs/security-audit-remediation-plan-2026-07-24.md`
- Modify: `docs/sec-005-credential-encryption-design-2026-07-24.md`

- [ ] **Step 1: Build a current-version legacy fixture**

Create a SQLite fixture with all registered fields in plaintext and no keyring. Start the new initialization path against it and assert automatic key creation, migration, permission repair, and successful service initialization.

- [ ] **Step 2: Run the full sentinel workflow**

Write unique sentinels through public services, perform DB backup, restore validation, key rotation, and restart validation. Scan SQLite, WAL, SHM, `xpanel.log`, operation logs, and the ordinary xpctl backup for plaintext. Expected matches: zero.

- [x] **Step 3: Run complete verification**

```bash
cd backend
gofmt -w security/credentials init/credential init/permission app/repo
go test ./security/credentials ./init/credential ./init/permission ./app/repo ./app/service ./cmd/server ./server -v
go test ./...
go test -race ./...
go vet ./...
cd ../frontend
pnpm run type-check
pnpm run build
cd ..
bash -n scripts/install-online.sh
bash -n scripts/xpctl
bash scripts/test-install-xpctl.sh
git diff --check
```

Expected: all tests and builds pass. Existing unrelated vet findings, if still present, must be reported separately rather than hidden.

- [x] **Step 4: Update remediation evidence**

Mark SEC-005 fixed only after all acceptance checks pass. Record exact commands, encrypted field coverage, keyring/DB modes, upgrade fixture, restore mismatch rejection, rotation evidence, sentinel scans, and any deliberately deferred KMS/Vault integration.

- [x] **Step 5: Review the final diff**

Confirm:

- no secret, real key, generated database, or recovery snapshot is tracked;
- no unrelated user changes were overwritten;
- the current-version upgrade path has a test;
- normal database backup excludes the keyring;
- missing/wrong key failures are actionable and contain no secret data;
- no Git commit or push was performed.
