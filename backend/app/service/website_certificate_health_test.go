package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
)

func writeCertificateFixture(
	t *testing.T,
	dir string,
	domains []string,
	notBefore, notAfter time.Time,
) (string, string, *x509.Certificate) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: domains[0]},
		DNSNames:     domains,
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse generated certificate: %v", err)
	}
	certPath := filepath.Join(dir, "fullchain.pem")
	keyPath := filepath.Join(dir, "privkey.pem")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create certificate directory: %v", err)
	}
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		t.Fatalf("write certificate: %v", err)
	}
	writePrivateKeyFixture(t, keyPath, key)
	return certPath, keyPath, parsed
}

func writePrivateKeyFixture(t *testing.T, path string, key *rsa.PrivateKey) {
	t.Helper()
	encoded := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
}

func TestCertificateFilesReportValidityDomainKeyAndFingerprint(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	certPath, keyPath, _ := writeCertificateFixture(
		t, t.TempDir(), []string{"example.com", "*.example.net"}, now.Add(-time.Hour), now.Add(30*24*time.Hour),
	)

	valid := inspectCertificateFiles(certPath, keyPath, []string{"example.com", "www.example.net"}, now)
	if valid.Status != "valid" || !valid.DomainMatch || !valid.KeyMatch || len(valid.FingerprintSHA256) != 64 {
		t.Fatalf("valid snapshot = %#v", valid)
	}
	wildcard := inspectCertificateFiles(certPath, keyPath, []string{"*.example.net"}, now)
	if wildcard.Status != "valid" || !wildcard.DomainMatch {
		t.Fatalf("wildcard server name snapshot = %#v", wildcard)
	}
	mismatch := inspectCertificateFiles(certPath, keyPath, []string{"other.example.org"}, now)
	if mismatch.Status != "domain_mismatch" || mismatch.DomainMatch {
		t.Fatalf("domain mismatch snapshot = %#v", mismatch)
	}

	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate mismatched key: %v", err)
	}
	writePrivateKeyFixture(t, keyPath, otherKey)
	keyMismatch := inspectCertificateFiles(certPath, keyPath, []string{"example.com"}, now)
	if keyMismatch.Status != "key_mismatch" || keyMismatch.KeyMatch {
		t.Fatalf("key mismatch snapshot = %#v", keyMismatch)
	}
}

func TestCertificateFilesReportExpiredAndExpiring(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	expiredCert, expiredKey, _ := writeCertificateFixture(
		t, filepath.Join(t.TempDir(), "expired"), []string{"expired.example.com"}, now.Add(-48*time.Hour), now.Add(-time.Hour),
	)
	expired := inspectCertificateFiles(expiredCert, expiredKey, []string{"expired.example.com"}, now)
	if expired.Status != "expired" || expired.DaysLeft >= 0 {
		t.Fatalf("expired snapshot = %#v", expired)
	}
	expiringCert, expiringKey, _ := writeCertificateFixture(
		t, filepath.Join(t.TempDir(), "expiring"), []string{"expiring.example.com"}, now.Add(-time.Hour), now.Add(10*24*time.Hour),
	)
	expiring := inspectCertificateFiles(expiringCert, expiringKey, []string{"expiring.example.com"}, now)
	if expiring.Status != "expiring" || expiring.DaysLeft != 10 {
		t.Fatalf("expiring snapshot = %#v", expiring)
	}
}

func TestCertificateEndpointNetworkFailureIsUnavailable(t *testing.T) {
	now := time.Now().UTC()
	got := probeTLSEndpoint(
		context.Background(), "example.com:8443", "example.com", now,
		func(context.Context, string, string) ([]*x509.Certificate, error) {
			return nil, errors.New("network down")
		},
		false,
	)
	if got.Status != "unavailable" || got.Status == "expired" {
		t.Fatalf("endpoint health = %#v", got)
	}
}

func TestProbeTLSEndpointVerifiesPublicChainButKeepsLocalClassification(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	_, _, certificate := writeCertificateFixture(
		t, t.TempDir(), []string{"example.com"}, now.Add(-time.Hour), now.Add(30*24*time.Hour),
	)
	probe := func(context.Context, string, string) ([]*x509.Certificate, error) {
		return []*x509.Certificate{certificate}, nil
	}

	public := probeTLSEndpoint(context.Background(), "example.com:443", "example.com", now, probe, true)
	if public.Status != "untrusted" || public.ChainTrusted {
		t.Fatalf("public health = %#v", public)
	}
	local := probeTLSEndpoint(context.Background(), "127.0.0.1:443", "example.com", now, probe, false)
	if local.Status != "valid" {
		t.Fatalf("local health = %#v", local)
	}
}

func TestWebsiteCertificateHealthBatchRunsOneNginxTestAndLimitsConcurrency(t *testing.T) {
	var sites []model.Website
	for i := 1; i <= 12; i++ {
		sites = append(sites, model.Website{BaseModel: model.BaseModel{ID: uint(i)}})
	}
	var nginxCalls int32
	var active int32
	var maximum int32
	results, _, _ := runWebsiteCertificateHealthBatch(
		context.Background(),
		sites,
		func() error {
			atomic.AddInt32(&nginxCalls, 1)
			return nil
		},
		func(_ context.Context, site model.Website, _ bool, _ string) dto.WebsiteCertificateHealthResp {
			current := atomic.AddInt32(&active, 1)
			for {
				observed := atomic.LoadInt32(&maximum)
				if current <= observed || atomic.CompareAndSwapInt32(&maximum, observed, current) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&active, -1)
			return dto.WebsiteCertificateHealthResp{WebsiteID: site.ID}
		},
	)
	if nginxCalls != 1 || maximum > 5 || len(results) != len(sites) {
		t.Fatalf("nginx calls=%d max=%d results=%d", nginxCalls, maximum, len(results))
	}
	for i, result := range results {
		if result.WebsiteID != sites[i].ID {
			t.Fatalf("result order = %#v", results)
		}
	}
}

func TestWebsiteCertificateHealthUsesCurrentExternalConfigAndConfiguredPort(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	certificateDir := t.TempDir()
	certPath, keyPath, certificate := writeCertificateFixture(
		t,
		certificateDir,
		[]string{"health.example.com", "www.health.example.com"},
		now.Add(-time.Hour),
		now.Add(30*24*time.Hour),
	)
	confPath, websiteService := setupExternalWebsiteFixture(t, `server {
    listen 9443 ssl;
    server_name health.example.com www.health.example.com;
    ssl_certificate `+certPath+`;
    ssl_certificate_key `+keyPath+`;
}`)
	site, err := websiteService.CreateExternalNginxSite(dto.ExternalNginxSiteCreateReq{Path: confPath})
	if err != nil {
		t.Fatalf("create external site: %v", err)
	}
	_, items, err := websiteService.SearchWithPage(dto.WebsiteSearch{PageInfo: dto.PageInfo{Page: 1, PageSize: 10}})
	if err != nil {
		t.Fatalf("list websites: %v", err)
	}
	if len(items) != 1 || items[0].ConfiguredCertificate == nil || items[0].ConfiguredCertificate.Status != "valid" {
		t.Fatalf("configured certificate summary = %#v", items)
	}
	var mutex sync.Mutex
	var probes []string
	nginxCalls := 0
	websiteService.certificateHealthOps = &certificateHealthOps{
		now: func() time.Time { return now },
		nginxTest: func() error {
			nginxCalls++
			return nil
		},
		probe: func(_ context.Context, address, serverName string) ([]*x509.Certificate, error) {
			mutex.Lock()
			probes = append(probes, address+"|"+serverName)
			mutex.Unlock()
			return []*x509.Certificate{certificate}, nil
		},
	}

	health, err := websiteService.CheckWebsiteCertificateHealth(site.ID)
	if err != nil {
		t.Fatalf("check certificate health: %v", err)
	}
	if nginxCalls != 1 || health.HTTPSPort != 9443 || health.Configured.Status != "valid" || health.Local.Status != "valid" {
		t.Fatalf("health = %#v, nginx calls = %d", health, nginxCalls)
	}
	if health.ConfigMatchesLocal == nil || !*health.ConfigMatchesLocal || len(health.Public) != 2 {
		t.Fatalf("fingerprint/public health = %#v", health)
	}
	mutex.Lock()
	slices.Sort(probes)
	gotProbes := slices.Clone(probes)
	mutex.Unlock()
	wantProbes := []string{
		"127.0.0.1:9443|health.example.com",
		"health.example.com:9443|health.example.com",
		"www.health.example.com:9443|www.health.example.com",
	}
	slices.Sort(wantProbes)
	if !slices.Equal(gotProbes, wantProbes) {
		t.Fatalf("probes = %#v, want %#v", gotProbes, wantProbes)
	}

	writeNginxFixture(t, confPath, `server {
    listen 9443 ssl;
    server_name changed.example.com;
    ssl_certificate `+certPath+`;
    ssl_certificate_key `+keyPath+`;
}`)
	health, err = websiteService.CheckWebsiteCertificateHealth(site.ID)
	if err != nil {
		t.Fatalf("recheck changed external config: %v", err)
	}
	if health.Configured.Status != "domain_mismatch" || len(health.Public) != 1 || health.Public[0].Domain != "changed.example.com" {
		t.Fatalf("external change was not detected: %#v", health)
	}
	stored, err := websiteService.websiteRepo.Get(repo.WithByID(site.ID))
	if err != nil || stored.PrimaryDomain != "changed.example.com" || stored.HttpsPort != 9443 {
		t.Fatalf("certificate check did not synchronize current metadata: %#v, %v", stored, err)
	}
}

func TestWebsiteCertificateHealthBatchAllIncludesOnlyHTTPSWebsites(t *testing.T) {
	externalPath, websiteService := setupExternalWebsiteFixture(t, `server { listen 80; server_name plain.example.com; }`)
	plain, err := websiteService.CreateExternalNginxSite(dto.ExternalNginxSiteCreateReq{Path: externalPath})
	if err != nil {
		t.Fatalf("create plain source website: %v", err)
	}
	https := model.Website{PrimaryDomain: "secure.example.com", Alias: "secure", ConfigMode: "managed", SSLEnable: true, HttpsPort: 443}
	if err := websiteService.websiteRepo.Create(&https); err != nil {
		t.Fatalf("create HTTPS website: %v", err)
	}
	websiteService.certificateHealthOps = &certificateHealthOps{
		nginxTest: func() error { return nil },
		probe: func(context.Context, string, string) ([]*x509.Certificate, error) {
			return nil, errors.New("offline")
		},
	}

	results, err := websiteService.CheckWebsiteCertificateHealthBatch(dto.WebsiteCertificateHealthBatchReq{All: true})
	if err != nil {
		t.Fatalf("batch certificate health: %v", err)
	}
	if len(results) != 1 || results[0].WebsiteID != https.ID {
		t.Fatalf("batch included non-HTTPS source website %d: %#v", plain.ID, results)
	}
}

func TestVerifyWebsiteLocalCertificateMatchesConfiguredFingerprintWithoutPublicProbe(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	certPath, keyPath, certificate := writeCertificateFixture(
		t, t.TempDir(), []string{"verify.example.com"}, now.Add(-time.Hour), now.Add(30*24*time.Hour),
	)
	confPath, websiteService := setupExternalWebsiteFixture(t, `server {
    listen 9443 ssl;
    server_name verify.example.com;
    ssl_certificate `+certPath+`;
    ssl_certificate_key `+keyPath+`;
}`)
	site, err := websiteService.CreateExternalNginxSite(dto.ExternalNginxSiteCreateReq{Path: confPath})
	if err != nil {
		t.Fatalf("create external site: %v", err)
	}
	var addresses []string
	ops := certificateHealthOps{
		now: func() time.Time { return now },
		probe: func(_ context.Context, address, serverName string) ([]*x509.Certificate, error) {
			addresses = append(addresses, address+"|"+serverName)
			return []*x509.Certificate{certificate}, nil
		},
	}
	if err := websiteService.verifyWebsiteLocalCertificate(context.Background(), *site, ops); err != nil {
		t.Fatalf("verify local certificate: %v", err)
	}
	want := []string{"127.0.0.1:9443|verify.example.com"}
	if !slices.Equal(addresses, want) {
		t.Fatalf("probes = %#v, want local-only %#v", addresses, want)
	}
}

func TestVerifyWebsiteLocalCertificateRejectsMismatchUnavailableAndWildcardOnly(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	certPath, keyPath, _ := writeCertificateFixture(
		t, filepath.Join(t.TempDir(), "configured"), []string{"verify.example.com"}, now.Add(-time.Hour), now.Add(30*24*time.Hour),
	)
	_, _, otherCertificate := writeCertificateFixture(
		t, filepath.Join(t.TempDir(), "other"), []string{"verify.example.com"}, now.Add(-time.Hour), now.Add(30*24*time.Hour),
	)
	confPath, websiteService := setupExternalWebsiteFixture(t, `server {
    listen 443 ssl;
    server_name verify.example.com;
    ssl_certificate `+certPath+`;
    ssl_certificate_key `+keyPath+`;
}`)
	site, err := websiteService.CreateExternalNginxSite(dto.ExternalNginxSiteCreateReq{Path: confPath})
	if err != nil {
		t.Fatalf("create external site: %v", err)
	}

	mismatchOps := certificateHealthOps{now: func() time.Time { return now }, probe: func(context.Context, string, string) ([]*x509.Certificate, error) {
		return []*x509.Certificate{otherCertificate}, nil
	}}
	if err := websiteService.verifyWebsiteLocalCertificate(context.Background(), *site, mismatchOps); err == nil || !strings.Contains(err.Error(), "指纹") {
		t.Fatalf("mismatch error = %v", err)
	}
	unavailableOps := certificateHealthOps{now: func() time.Time { return now }, probe: func(context.Context, string, string) ([]*x509.Certificate, error) {
		return nil, errors.New("offline")
	}}
	if err := websiteService.verifyWebsiteLocalCertificate(context.Background(), *site, unavailableOps); err == nil || !strings.Contains(err.Error(), "offline") {
		t.Fatalf("unavailable error = %v", err)
	}

	writeNginxFixture(t, confPath, `server {
    listen 443 ssl;
    server_name *.example.com;
    ssl_certificate `+certPath+`;
    ssl_certificate_key `+keyPath+`;
}`)
	probes := 0
	wildcardOps := certificateHealthOps{now: func() time.Time { return now }, probe: func(context.Context, string, string) ([]*x509.Certificate, error) {
		probes++
		return nil, nil
	}}
	if err := websiteService.verifyWebsiteLocalCertificate(context.Background(), *site, wildcardOps); err == nil || !strings.Contains(err.Error(), "具体域名") {
		t.Fatalf("wildcard error = %v", err)
	}
	if probes != 0 {
		t.Fatalf("wildcard-only site performed %d probes", probes)
	}
}
