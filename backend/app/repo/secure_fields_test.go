package repo

import (
	"path/filepath"
	"strings"
	"testing"

	"xpanel/app/model"
	"xpanel/global"
	"xpanel/security/credentials"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestHostRepositoryProtectsCredentials(t *testing.T) {
	db := openSecureRepoDatabase(t)
	repository := NewIHostRepo()
	host := &model.Host{
		Name:       "server",
		Addr:       "127.0.0.1",
		Password:   "host-password",
		PrivateKey: "host-private-key",
		PassPhrase: "host-passphrase",
	}
	if err := repository.Create(host); err != nil {
		t.Fatalf("create host: %v", err)
	}
	assertRawEncrypted(t, db, "hosts", "password", host.ID, "host-password")
	assertRawEncrypted(t, db, "hosts", "private_key", host.ID, "host-private-key")
	assertRawEncrypted(t, db, "hosts", "pass_phrase", host.ID, "host-passphrase")

	stored, err := repository.Get(WithByID(host.ID))
	if err != nil {
		t.Fatalf("get host: %v", err)
	}
	if stored.Password != "host-password" ||
		stored.PrivateKey != "host-private-key" ||
		stored.PassPhrase != "host-passphrase" {
		t.Fatalf("revealed host credentials = %#v", stored)
	}

	updates := map[string]any{"password": "updated-password", "name": "updated"}
	if err := repository.Update(host.ID, updates); err != nil {
		t.Fatalf("update host: %v", err)
	}
	if updates["password"] != "updated-password" {
		t.Fatalf("repository mutated caller update map: %#v", updates)
	}
	assertRawEncrypted(t, db, "hosts", "password", host.ID, "updated-password")
}

func TestNodeRepositoryProtectsCredentials(t *testing.T) {
	db := openSecureRepoDatabase(t)
	repository := NewINodeRepo()
	node := &model.Node{
		Name:        "node",
		Token:       "node-token",
		SSHPassword: "node-ssh-password",
	}
	if err := repository.Create(node); err != nil {
		t.Fatalf("create node: %v", err)
	}
	assertRawEncrypted(t, db, "nodes", "token", node.ID, "node-token")
	assertRawEncrypted(t, db, "nodes", "ssh_password", node.ID, "node-ssh-password")

	stored, err := repository.Get(node.ID)
	if err != nil {
		t.Fatalf("get node: %v", err)
	}
	if stored.Token != "node-token" || stored.SSHPassword != "node-ssh-password" {
		t.Fatalf("revealed node credentials = %#v", stored)
	}
}

func TestBackupRepositoryProtectsCredentials(t *testing.T) {
	db := openSecureRepoDatabase(t)
	repository := NewIBackupRepo()
	account := &model.BackupAccount{
		Name:       "s3",
		Type:       "s3",
		AccessKey:  "backup-access-key",
		Credential: "backup-credential",
	}
	if err := repository.CreateAccount(account); err != nil {
		t.Fatalf("create backup account: %v", err)
	}
	assertRawEncrypted(t, db, "backup_accounts", "access_key", account.ID, "backup-access-key")
	assertRawEncrypted(t, db, "backup_accounts", "credential", account.ID, "backup-credential")

	stored, err := repository.GetAccount(account.ID)
	if err != nil {
		t.Fatalf("get backup account: %v", err)
	}
	if stored.AccessKey != "backup-access-key" || stored.Credential != "backup-credential" {
		t.Fatalf("revealed backup credentials = %#v", stored)
	}
}

func TestDatabaseRepositoryProtectsCredentials(t *testing.T) {
	db := openSecureRepoDatabase(t)
	repository := NewIDatabaseRepo()
	server := &model.DatabaseServer{
		Name:     "mysql",
		Type:     "mysql",
		Password: "server-password",
	}
	if err := repository.CreateServer(server); err != nil {
		t.Fatalf("create database server: %v", err)
	}
	instance := &model.DatabaseInstance{
		ServerID: server.ID,
		Name:     "app",
		Password: "instance-password",
	}
	if err := repository.CreateInstance(instance); err != nil {
		t.Fatalf("create database instance: %v", err)
	}
	assertRawEncrypted(t, db, "database_servers", "password", server.ID, "server-password")
	assertRawEncrypted(t, db, "database_instances", "password", instance.ID, "instance-password")

	storedServer, err := repository.GetServer(server.ID)
	if err != nil {
		t.Fatalf("get database server: %v", err)
	}
	storedInstance, err := repository.GetInstance(instance.ID)
	if err != nil {
		t.Fatalf("get database instance: %v", err)
	}
	if storedServer.Password != "server-password" || storedInstance.Password != "instance-password" {
		t.Fatalf("revealed database credentials = server:%q instance:%q",
			storedServer.Password, storedInstance.Password)
	}
}

func TestSSLRepositoriesProtectCredentials(t *testing.T) {
	db := openSecureRepoDatabase(t)

	acmeRepo := NewIAcmeAccountRepo()
	acme := &model.AcmeAccount{
		Email:      "admin@example.com",
		Type:       "custom",
		PrivateKey: "acme-private-key",
		EabHmacKey: "acme-eab-key",
	}
	if err := acmeRepo.Create(acme); err != nil {
		t.Fatalf("create ACME account: %v", err)
	}
	assertRawEncrypted(t, db, "acme_accounts", "private_key", acme.ID, "acme-private-key")
	assertRawEncrypted(t, db, "acme_accounts", "eab_hmac_key", acme.ID, "acme-eab-key")
	storedAcme, err := acmeRepo.Get(WithByID(acme.ID))
	if err != nil {
		t.Fatalf("get ACME account: %v", err)
	}
	if storedAcme.PrivateKey != "acme-private-key" || storedAcme.EabHmacKey != "acme-eab-key" {
		t.Fatalf("revealed ACME credentials = %#v", storedAcme)
	}

	dnsRepo := NewIDnsAccountRepo()
	dns := &model.DnsAccount{Name: "dns", Type: "CloudFlare", Authorization: `{"token":"dns-secret"}`}
	if err := dnsRepo.Create(dns); err != nil {
		t.Fatalf("create DNS account: %v", err)
	}
	assertRawEncrypted(t, db, "dns_accounts", "authorization", dns.ID, "dns-secret")
	storedDNS, err := dnsRepo.Get(WithByID(dns.ID))
	if err != nil {
		t.Fatalf("get DNS account: %v", err)
	}
	if storedDNS.Authorization != `{"token":"dns-secret"}` {
		t.Fatalf("revealed DNS authorization = %q", storedDNS.Authorization)
	}

	certRepo := NewICertificateRepo()
	certificate := &model.Certificate{
		PrimaryDomain: "example.com",
		Provider:      "manual",
		PrivateKey:    "certificate-private-key",
	}
	if err := certRepo.Create(certificate); err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	assertRawEncrypted(t, db, "certificates", "private_key", certificate.ID, "certificate-private-key")
	storedCertificate, err := certRepo.Get(WithByID(certificate.ID))
	if err != nil {
		t.Fatalf("get certificate: %v", err)
	}
	if storedCertificate.PrivateKey != "certificate-private-key" {
		t.Fatalf("revealed certificate private key = %q", storedCertificate.PrivateKey)
	}
}

func TestFeatureRepositoriesProtectCredentials(t *testing.T) {
	db := openSecureRepoDatabase(t)

	sourceRepo := NewICertSourceRepo()
	source := &model.CertSource{Name: "source", ServerAddr: "https://internal", Token: "cert-source-token"}
	if err := sourceRepo.Create(source); err != nil {
		t.Fatalf("create certificate source: %v", err)
	}
	assertRawEncrypted(t, db, "cert_sources", "token", source.ID, "cert-source-token")
	storedSource, err := sourceRepo.Get(WithByID(source.ID))
	if err != nil || storedSource.Token != "cert-source-token" {
		t.Fatalf("revealed certificate source = %#v, err=%v", storedSource, err)
	}

	websiteRepo := NewIWebsiteRepo()
	website := &model.Website{
		PrimaryDomain: "site.example.com",
		Alias:         "site",
		BasicPassword: "website-password",
	}
	if err := websiteRepo.Create(website); err != nil {
		t.Fatalf("create website: %v", err)
	}
	assertRawEncrypted(t, db, "websites", "basic_password", website.ID, "website-password")
	storedWebsite, err := websiteRepo.Get(WithByID(website.ID))
	if err != nil || storedWebsite.BasicPassword != "website-password" {
		t.Fatalf("revealed website = %#v, err=%v", storedWebsite, err)
	}

	gostRepo := NewIGostServiceRepo()
	gost := &model.GostService{
		Name:         "relay",
		Type:         "relay_server",
		ListenAddr:   ":9000",
		ListenerType: "tcp",
		AuthPass:     "gost-password",
	}
	if err := gostRepo.Create(gost); err != nil {
		t.Fatalf("create GOST service: %v", err)
	}
	assertRawEncrypted(t, db, "gost_services", "auth_pass", gost.ID, "gost-password")
	storedGost, err := gostRepo.Get(WithByID(gost.ID))
	if err != nil || storedGost.AuthPass != "gost-password" {
		t.Fatalf("revealed GOST service = %#v, err=%v", storedGost, err)
	}

	chainRepo := NewIGostChainRepo()
	chain := &model.GostChain{Name: "chain", Hops: `[{"password":"hop-secret"}]`}
	if err := chainRepo.Create(chain); err != nil {
		t.Fatalf("create GOST chain: %v", err)
	}
	assertRawEncrypted(t, db, "gost_chains", "hops", chain.ID, "hop-secret")
	storedChain, err := chainRepo.Get(WithByID(chain.ID))
	if err != nil || storedChain.Hops != `[{"password":"hop-secret"}]` {
		t.Fatalf("revealed GOST chain = %#v, err=%v", storedChain, err)
	}

	cronRepo := NewICronjobRepo()
	job := &model.Cronjob{
		Name:            "backup",
		Type:            "directory",
		Spec:            "0 0 * * *",
		EncryptPassword: "archive-password",
	}
	if err := cronRepo.Create(job); err != nil {
		t.Fatalf("create cronjob: %v", err)
	}
	assertRawEncrypted(t, db, "cronjobs", "encrypt_password", job.ID, "archive-password")
	storedJob, err := cronRepo.Get(job.ID)
	if err != nil || storedJob.EncryptPassword != "archive-password" {
		t.Fatalf("revealed cronjob = %#v, err=%v", storedJob, err)
	}

	historyRepo := NewIHAProxyConfigVersionRepo()
	history := &model.HAProxyConfigVersion{Content: "stats auth admin:haproxy-secret"}
	if err := historyRepo.Create(history); err != nil {
		t.Fatalf("create HAProxy config history: %v", err)
	}
	assertRawEncrypted(t, db, "ha_proxy_config_versions", "content", history.ID, "haproxy-secret")
	storedHistory, err := historyRepo.Get(history.ID)
	if err != nil || storedHistory.Content != "stats auth admin:haproxy-secret" {
		t.Fatalf("revealed HAProxy history = %#v, err=%v", storedHistory, err)
	}
}

func TestSettingRepositoryEncryptsOnlyRegisteredSecrets(t *testing.T) {
	db := openSecureRepoDatabase(t)
	repository := NewISettingRepo()

	for _, test := range []struct {
		key       string
		value     string
		encrypted bool
	}{
		{key: "NezhaClientSecret", value: "nezha-client-secret", encrypted: true},
		{key: "ProxyAddress", value: "socks5://user:proxy-secret@127.0.0.1:1080", encrypted: true},
		{key: "Password", value: "$2a$10$already-hashed", encrypted: false},
		{key: "PanelName", value: "X-Panel", encrypted: false},
	} {
		if err := repository.CreateOrUpdate(test.key, test.value); err != nil {
			t.Fatalf("create setting %s: %v", test.key, err)
		}
		var raw string
		if err := db.Table("settings").Select("value").Where("key = ?", test.key).Scan(&raw).Error; err != nil {
			t.Fatalf("read raw setting %s: %v", test.key, err)
		}
		if got := global.CREDENTIALS.IsEncrypted(raw); got != test.encrypted {
			t.Fatalf("setting %s encrypted = %v, want %v; raw=%q", test.key, got, test.encrypted, raw)
		}
		value, err := repository.GetValueByKey(test.key)
		if err != nil {
			t.Fatalf("get setting %s: %v", test.key, err)
		}
		if value != test.value {
			t.Fatalf("setting %s value = %q, want %q", test.key, value, test.value)
		}
	}
}

func TestSettingRepositoryCreateOrUpdateManyProtectsAllValues(t *testing.T) {
	db := openSecureRepoDatabase(t)
	repository := NewISettingRepo()
	values := map[string]string{
		"NezhaClientSecret": "nezha-client-secret",
		"NezhaServer":       "dashboard.example.com:443",
	}
	if err := repository.CreateOrUpdateMany(values); err != nil {
		t.Fatal(err)
	}
	for key, want := range values {
		var raw string
		if err := db.Table("settings").Select("value").Where("key = ?", key).Scan(&raw).Error; err != nil {
			t.Fatal(err)
		}
		if credentials.IsSecretSetting(key) {
			if !global.CREDENTIALS.IsEncrypted(raw) || strings.Contains(raw, want) {
				t.Fatalf("secret setting %s stored without protection", key)
			}
		} else if raw != want {
			t.Fatalf("plain setting %s = %q, want %q", key, raw, want)
		}
		got, err := repository.GetValueByKey(key)
		if err != nil || got != want {
			t.Fatalf("GetValueByKey(%s) = %q, %v", key, got, err)
		}
	}
}

func TestSettingRepositoryCreateIfMissingOrEmptyNeverOverwritesExplicitValue(t *testing.T) {
	openSecureRepoDatabase(t)
	repository := NewISettingRepo()
	written, err := repository.CreateIfMissingOrEmpty("NezhaServer", "https://manifest.example.com")
	if err != nil || !written {
		t.Fatalf("initial conditional write = %v, %v", written, err)
	}
	if err := repository.CreateOrUpdate("NezhaServer", "https://explicit.example.com"); err != nil {
		t.Fatal(err)
	}
	written, err = repository.CreateIfMissingOrEmpty("NezhaServer", "https://other.example.com")
	if err != nil || written {
		t.Fatalf("explicit value conditional write = %v, %v", written, err)
	}
	if got, err := repository.GetValueByKey("NezhaServer"); err != nil || got != "https://explicit.example.com" {
		t.Fatalf("NezhaServer = %q, %v", got, err)
	}
}

func openSecureRepoDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "repo.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open repository database: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Host{},
		&model.Node{},
		&model.BackupAccount{},
		&model.DatabaseServer{},
		&model.DatabaseInstance{},
		&model.Setting{},
		&model.AcmeAccount{},
		&model.DnsAccount{},
		&model.Certificate{},
		&model.CertSource{},
		&model.Website{},
		&model.GostService{},
		&model.GostChain{},
		&model.Cronjob{},
		&model.HAProxyConfigVersion{},
	); err != nil {
		t.Fatalf("migrate repository database: %v", err)
	}
	manager, _, err := credentials.LoadOrCreate(
		filepath.Join(t.TempDir(), "secrets", "credential-keyring.json"),
		true,
	)
	if err != nil {
		t.Fatalf("create repository keyring: %v", err)
	}
	global.DB = db
	global.CREDENTIALS = manager
	t.Cleanup(func() {
		global.DB = nil
		global.CREDENTIALS = nil
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func assertRawEncrypted(t *testing.T, db *gorm.DB, table, column string, id uint, plaintext string) {
	t.Helper()
	var value string
	if err := db.Table(table).Select(column).Where("id = ?", id).Scan(&value).Error; err != nil {
		t.Fatalf("read raw %s.%s: %v", table, column, err)
	}
	if strings.Contains(value, plaintext) || !global.CREDENTIALS.IsEncrypted(value) {
		t.Fatalf("%s.%s stored plaintext: %q", table, column, value)
	}
}
