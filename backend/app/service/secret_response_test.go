package service

import (
	"strings"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
)

func TestSanitizeGostHopsRemovesNestedPasswords(t *testing.T) {
	input := `[{"nodes":[{"addr":"relay.internal:443","dialer":{"auth":{"username":"admin","password":"gost-secret"}}}]}]`
	output := sanitizeGostHops(input)
	if strings.Contains(output, "gost-secret") {
		t.Fatalf("sanitized GOST hops exposed password: %s", output)
	}
	if !strings.Contains(output, "relay.internal:443") || !strings.Contains(output, "admin") {
		t.Fatalf("sanitized GOST hops removed non-secret topology: %s", output)
	}
}

func TestMergeGostHopSecretsRetainsBlankReplacement(t *testing.T) {
	existing := `[{"nodes":[{"dialer":{"auth":{"username":"admin","password":"gost-secret"}}}]}]`
	update := `[{"nodes":[{"dialer":{"auth":{"username":"admin","password":""}}}]}]`
	merged, err := mergeGostHopSecrets(existing, update)
	if err != nil {
		t.Fatalf("merge GOST hop secrets: %v", err)
	}
	if !strings.Contains(merged, "gost-secret") {
		t.Fatalf("blank GOST password did not retain existing secret: %s", merged)
	}
}

func TestMergeGostHopSecretsRejectsBlankSecretAfterNodeReorder(t *testing.T) {
	existing := `[{"name":"hop-0","nodes":[` +
		`{"name":"node-0","addr":"a.internal:443","dialer":{"type":"wss","auth":{"username":"a","password":"pass-a"}}},` +
		`{"name":"node-1","addr":"b.internal:443","dialer":{"type":"wss","auth":{"username":"b","password":"pass-b"}}}` +
		`]}]`
	update := `[{"name":"hop-0","nodes":[` +
		`{"name":"node-0","addr":"b.internal:443","dialer":{"type":"wss","auth":{"username":"b","password":""}}},` +
		`{"name":"node-1","addr":"a.internal:443","dialer":{"type":"wss","auth":{"username":"a","password":""}}}` +
		`]}]`
	if _, err := mergeGostHopSecrets(existing, update); err == nil {
		t.Fatal("reordered GOST nodes with blank secrets should require password re-entry")
	}
}

func TestMergeDNSAuthorizationRetainsBlankAndMissingSecrets(t *testing.T) {
	merged, err := mergeDNSAuthorization(
		`{"token":"dns-secret","zone":"example.com"}`,
		map[string]string{"token": "", "region": "cn"},
	)
	if err != nil {
		t.Fatalf("merge DNS authorization: %v", err)
	}
	if !strings.Contains(merged, "dns-secret") ||
		!strings.Contains(merged, "example.com") ||
		!strings.Contains(merged, `"region":"cn"`) {
		t.Fatalf("merged DNS authorization lost values: %s", merged)
	}
}

func TestDNSProviderChangeDoesNotReusePreviousAuthorization(t *testing.T) {
	if _, err := mergeDNSAuthorizationForUpdate(
		"cloudflare",
		"route53",
		`{"accessKey":"old-access","secretKey":"old-secret"}`,
		map[string]string{"accessKey": "", "secretKey": ""},
	); err == nil {
		t.Fatal("provider change with blank authorization should be rejected")
	}

	updated, err := mergeDNSAuthorizationForUpdate(
		"cloudflare",
		"route53",
		`{"accessKey":"old-access","secretKey":"old-secret"}`,
		map[string]string{"accessKey": "new-access", "secretKey": "new-secret"},
	)
	if err != nil {
		t.Fatalf("provider change with complete authorization: %v", err)
	}
	if strings.Contains(updated, "old-") || !strings.Contains(updated, "new-secret") {
		t.Fatalf("provider change reused old authorization: %s", updated)
	}
}

func TestHAProxyConfigHistoryDoesNotReturnStoredContent(t *testing.T) {
	if output := redactHAProxyConfigHistory("stats auth admin:haproxy-secret"); output != "" {
		t.Fatalf("HAProxy history content was returned: %q", output)
	}
}

func TestSecretBearingDetailsExposeOnlyConfiguredFlags(t *testing.T) {
	website := &dto.WebsiteDetail{BasicPassword: "website-secret"}
	redactWebsiteSecret(website, model.Website{BasicPassword: "website-secret"})
	if website.BasicPassword != "" || !website.BasicPasswordSet {
		t.Fatalf("website detail exposed or lost configured state: %#v", website)
	}
	certificate := &dto.CertificateDetail{PrivateKey: "certificate-private-key"}
	redactCertificateSecret(certificate, model.Certificate{PrivateKey: "certificate-private-key"})
	if certificate.PrivateKey != "" || !certificate.PrivateKeySet {
		t.Fatalf("certificate detail exposed or lost private-key state: %#v", certificate)
	}
}
