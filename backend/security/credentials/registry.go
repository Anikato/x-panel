package credentials

import "sort"

type FieldSpec struct {
	Table  string
	Column string
	Scope  string
}

var FieldSpecs = []FieldSpec{
	{Table: "hosts", Column: "password", Scope: "hosts.password"},
	{Table: "hosts", Column: "private_key", Scope: "hosts.private_key"},
	{Table: "hosts", Column: "pass_phrase", Scope: "hosts.pass_phrase"},
	{Table: "nodes", Column: "token", Scope: "nodes.token"},
	{Table: "nodes", Column: "ssh_password", Scope: "nodes.ssh_password"},
	{Table: "backup_accounts", Column: "access_key", Scope: "backup_accounts.access_key"},
	{Table: "backup_accounts", Column: "credential", Scope: "backup_accounts.credential"},
	{Table: "database_servers", Column: "password", Scope: "database_servers.password"},
	{Table: "database_instances", Column: "password", Scope: "database_instances.password"},
	{Table: "acme_accounts", Column: "private_key", Scope: "acme_accounts.private_key"},
	{Table: "acme_accounts", Column: "eab_hmac_key", Scope: "acme_accounts.eab_hmac_key"},
	{Table: "dns_accounts", Column: "authorization", Scope: "dns_accounts.authorization"},
	{Table: "certificates", Column: "private_key", Scope: "certificates.private_key"},
	{Table: "cert_sources", Column: "token", Scope: "cert_sources.token"},
	{Table: "websites", Column: "basic_password", Scope: "websites.basic_password"},
	{Table: "gost_services", Column: "auth_pass", Scope: "gost_services.auth_pass"},
	{Table: "gost_chains", Column: "hops", Scope: "gost_chains.hops"},
	{Table: "cronjobs", Column: "encrypt_password", Scope: "cronjobs.encrypt_password"},
	{Table: "ha_proxy_config_versions", Column: "content", Scope: "ha_proxy_config_versions.content"},
}

var SecretSettingKeys = map[string]struct{}{
	"MFASecret":            {},
	"GitHubToken":          {},
	"AgentToken":           {},
	"CertServerToken":      {},
	"FleetInstanceToken":   {},
	"FleetEnrollmentToken": {},
	"HAProxyStatsPass":     {},
	"GostAPIPass":          {},
	"ProxyAddress":         {},
	"SecurityEntrance":     {},
}

func IsSecretSetting(key string) bool {
	_, ok := SecretSettingKeys[key]
	return ok
}

func SettingScope(key string) string {
	return "settings." + key
}

func SecretSettingKeyList() []string {
	keys := make([]string, 0, len(SecretSettingKeys))
	for key := range SecretSettingKeys {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func ScopeFor(table, column string) (string, bool) {
	for _, spec := range FieldSpecs {
		if spec.Table == table && spec.Column == column {
			return spec.Scope, true
		}
	}
	return "", false
}
