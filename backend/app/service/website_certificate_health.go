package service

import (
	"bytes"
	"context"
	"crypto"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
)

const certificateExpiringDays = 15

type tlsCertificateProbe func(context.Context, string, string) ([]*x509.Certificate, error)

type certificateHealthOps struct {
	now       func() time.Time
	nginxTest func() error
	probe     tlsCertificateProbe
}

func inspectCertificateFiles(certPath, keyPath string, domains []string, now time.Time) dto.CertificateHealthSnapshot {
	snapshot := dto.CertificateHealthSnapshot{
		Status: "unreadable", CertPath: certPath, KeyPath: keyPath,
	}
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		snapshot.Error = err.Error()
		return snapshot
	}
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		snapshot.Error = "证书 PEM 无效"
		return snapshot
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		snapshot.Error = err.Error()
		return snapshot
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		snapshot.Error = err.Error()
		return snapshot
	}
	privateKey, err := parsePrivateKeyPEM(keyPEM)
	if err != nil {
		snapshot.Error = err.Error()
		return snapshot
	}
	result := inspectCertificate(certificate, privateKey, domains, now)
	result.CertPath = certPath
	result.KeyPath = keyPath
	return result
}

func inspectCertificate(certificate *x509.Certificate, privateKey crypto.PrivateKey, domains []string, now time.Time) dto.CertificateHealthSnapshot {
	snapshot := dto.CertificateHealthSnapshot{
		Status:            "valid",
		NotBefore:         certificate.NotBefore,
		NotAfter:          certificate.NotAfter,
		DaysLeft:          certificateDaysLeft(certificate.NotAfter, now),
		DomainMatch:       true,
		KeyMatch:          certificateKeyMatches(certificate, privateKey),
		FingerprintSHA256: certificateFingerprint(certificate),
	}
	for _, domain := range domains {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}
		if err := certificate.VerifyHostname(domain); err != nil {
			snapshot.DomainMatch = false
			snapshot.MismatchedDomains = append(snapshot.MismatchedDomains, domain)
		}
	}
	if !snapshot.KeyMatch {
		snapshot.Status = "key_mismatch"
		snapshot.Error = "证书与私钥不匹配"
	} else if now.Before(certificate.NotBefore) {
		snapshot.Status = "not_yet_valid"
	} else if !now.Before(certificate.NotAfter) {
		snapshot.Status = "expired"
	} else if !snapshot.DomainMatch {
		snapshot.Status = "domain_mismatch"
		snapshot.Error = "证书与一个或多个网站域名不匹配"
	} else if snapshot.DaysLeft <= certificateExpiringDays {
		snapshot.Status = "expiring"
	}
	return snapshot
}

func parsePrivateKeyPEM(content []byte) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(content)
	if block == nil {
		return nil, fmt.Errorf("私钥 PEM 无效")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	return nil, fmt.Errorf("不支持或损坏的私钥")
}

func certificateKeyMatches(certificate *x509.Certificate, privateKey crypto.PrivateKey) bool {
	publicKeyProvider, ok := privateKey.(interface{ Public() crypto.PublicKey })
	if !ok {
		return false
	}
	certificatePublic, err := x509.MarshalPKIXPublicKey(certificate.PublicKey)
	if err != nil {
		return false
	}
	privatePublic, err := x509.MarshalPKIXPublicKey(publicKeyProvider.Public())
	return err == nil && bytes.Equal(certificatePublic, privatePublic)
}

func certificateFingerprint(certificate *x509.Certificate) string {
	sum := sha256.Sum256(certificate.Raw)
	return hex.EncodeToString(sum[:])
}

func certificateDaysLeft(notAfter, now time.Time) int {
	return int(math.Floor(notAfter.Sub(now).Hours() / 24))
}

func probeTLSEndpoint(
	ctx context.Context,
	address, serverName string,
	now time.Time,
	probe tlsCertificateProbe,
	verifyChain bool,
) dto.CertificateEndpointHealth {
	health := dto.CertificateEndpointHealth{
		Status: "unavailable", Domain: serverName, Address: address,
	}
	certificates, err := probe(ctx, address, serverName)
	if err != nil {
		health.Error = err.Error()
		return health
	}
	if len(certificates) == 0 {
		health.Error = "端点未返回证书"
		return health
	}
	certificate := certificates[0]
	health.NotAfter = certificate.NotAfter
	health.DaysLeft = certificateDaysLeft(certificate.NotAfter, now)
	health.FingerprintSHA256 = certificateFingerprint(certificate)
	health.DomainMatch = certificate.VerifyHostname(serverName) == nil
	var chainErr error
	if verifyChain {
		intermediates := x509.NewCertPool()
		for _, intermediate := range certificates[1:] {
			intermediates.AddCert(intermediate)
		}
		_, chainErr = certificate.Verify(x509.VerifyOptions{
			DNSName:       serverName,
			Intermediates: intermediates,
			CurrentTime:   now,
		})
		health.ChainTrusted = chainErr == nil
	}
	switch {
	case now.Before(certificate.NotBefore):
		health.Status = "not_yet_valid"
	case !now.Before(certificate.NotAfter):
		health.Status = "expired"
	case !health.DomainMatch:
		health.Status = "domain_mismatch"
		health.Error = "端点证书与域名不匹配"
	case verifyChain && chainErr != nil:
		health.Status = "untrusted"
		health.Error = "证书链验证失败: " + chainErr.Error()
	case health.DaysLeft <= certificateExpiringDays:
		health.Status = "expiring"
	default:
		health.Status = "valid"
	}
	return health
}

func dialTLSCertificates(ctx context.Context, address, serverName string) ([]*x509.Certificate, error) {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	// Certificate validity and hostname are checked explicitly so expired or mismatched
	// endpoints can be classified instead of being collapsed into a handshake error.
	tlsConnection := tls.Client(connection, &tls.Config{ServerName: serverName, InsecureSkipVerify: true}) // #nosec G402
	if err := tlsConnection.HandshakeContext(ctx); err != nil {
		return nil, err
	}
	return tlsConnection.ConnectionState().PeerCertificates, nil
}

type certificateHealthCheck func(context.Context, model.Website, bool, string) dto.WebsiteCertificateHealthResp

func runWebsiteCertificateHealthBatch(
	ctx context.Context,
	sites []model.Website,
	nginxTest func() error,
	check certificateHealthCheck,
) ([]dto.WebsiteCertificateHealthResp, bool, string) {
	nginxOK := true
	nginxError := ""
	if err := nginxTest(); err != nil {
		nginxOK = false
		nginxError = err.Error()
	}
	results := make([]dto.WebsiteCertificateHealthResp, len(sites))
	semaphore := make(chan struct{}, 5)
	var waitGroup sync.WaitGroup
	for index, site := range sites {
		index, site := index, site
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				results[index] = dto.WebsiteCertificateHealthResp{WebsiteID: site.ID}
				return
			}
			results[index] = check(ctx, site, nginxOK, nginxError)
		}()
	}
	waitGroup.Wait()
	return results, nginxOK, nginxError
}

func (s *WebsiteService) CheckWebsiteCertificateHealth(id uint) (*dto.WebsiteCertificateHealthResp, error) {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}
	site = s.synchronizeSourceSiteMetadata(site)
	ops := s.effectiveCertificateHealthOps()
	nginxOK := true
	nginxError := ""
	if err := ops.nginxTest(); err != nil {
		nginxOK = false
		nginxError = err.Error()
	}
	result := s.checkWebsiteCertificateHealth(context.Background(), site, nginxOK, nginxError, ops)
	return &result, nil
}

func (s *WebsiteService) CheckWebsiteCertificateHealthBatch(req dto.WebsiteCertificateHealthBatchReq) ([]dto.WebsiteCertificateHealthResp, error) {
	var sites []model.Website
	if req.All {
		allSites, err := s.websiteRepo.GetList()
		if err != nil {
			return nil, err
		}
		for _, site := range allSites {
			site = s.synchronizeSourceSiteMetadata(site)
			if site.SSLEnable || site.HttpsPort > 0 || site.CertificateID > 0 {
				sites = append(sites, site)
			}
		}
	} else {
		if len(req.IDs) == 0 {
			return nil, buserr.New(constant.ErrInvalidParams)
		}
		seen := make(map[uint]struct{}, len(req.IDs))
		for _, id := range req.IDs {
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			site, err := s.websiteRepo.Get(repo.WithByID(id))
			if err != nil {
				return nil, buserr.New(constant.ErrRecordNotFound)
			}
			site = s.synchronizeSourceSiteMetadata(site)
			sites = append(sites, site)
		}
	}
	ops := s.effectiveCertificateHealthOps()
	results, _, _ := runWebsiteCertificateHealthBatch(
		context.Background(),
		sites,
		ops.nginxTest,
		func(ctx context.Context, site model.Website, nginxOK bool, nginxError string) dto.WebsiteCertificateHealthResp {
			return s.checkWebsiteCertificateHealth(ctx, site, nginxOK, nginxError, ops)
		},
	)
	return results, nil
}

func (s *WebsiteService) effectiveCertificateHealthOps() certificateHealthOps {
	ops := certificateHealthOps{
		now:       time.Now,
		nginxTest: s.testNginxConfig,
		probe:     dialTLSCertificates,
	}
	if s.certificateHealthOps == nil {
		return ops
	}
	if s.certificateHealthOps.now != nil {
		ops.now = s.certificateHealthOps.now
	}
	if s.certificateHealthOps.nginxTest != nil {
		ops.nginxTest = s.certificateHealthOps.nginxTest
	}
	if s.certificateHealthOps.probe != nil {
		ops.probe = s.certificateHealthOps.probe
	}
	return ops
}

func (s *WebsiteService) checkWebsiteCertificateHealth(
	ctx context.Context,
	site model.Website,
	nginxOK bool,
	nginxError string,
	ops certificateHealthOps,
) dto.WebsiteCertificateHealthResp {
	now := ops.now()
	domains, port, certPath, keyPath, configErr := s.websiteCertificateContext(site)
	response := dto.WebsiteCertificateHealthResp{
		WebsiteID: site.ID, CheckedAt: now, HTTPSPort: port,
		NginxConfigOK: nginxOK, NginxConfigError: nginxError,
		Configured: dto.CertificateHealthSnapshot{Status: "not_configured", CertPath: certPath, KeyPath: keyPath},
		Local:      dto.CertificateEndpointHealth{Status: "not_checked"},
	}
	if configErr != nil {
		response.Configured.Status = "unreadable"
		response.Configured.Error = configErr.Error()
	} else if certPath != "" && keyPath != "" {
		response.Configured = inspectCertificateFiles(certPath, keyPath, domains, now)
	}
	if port <= 0 {
		port = 443
		response.HTTPSPort = port
	}
	if len(domains) == 0 {
		response.Local.Error = "网站未配置有效域名"
		return response
	}

	localAddress := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	localCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	response.Local = probeTLSEndpoint(localCtx, localAddress, domains[0], now, ops.probe, false)
	cancel()
	if response.Configured.FingerprintSHA256 != "" && response.Local.FingerprintSHA256 != "" {
		matches := response.Configured.FingerprintSHA256 == response.Local.FingerprintSHA256
		response.ConfigMatchesLocal = &matches
	}

	for _, domain := range domains {
		address := net.JoinHostPort(domain, strconv.Itoa(port))
		if strings.Contains(domain, "*") {
			response.Public = append(response.Public, dto.CertificateEndpointHealth{
				Status: "unavailable", Domain: domain, Address: address, Error: "通配域名不能直接探测",
			})
			continue
		}
		publicCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		response.Public = append(response.Public, probeTLSEndpoint(publicCtx, address, domain, now, ops.probe, true))
		cancel()
	}
	return response
}

func (s *WebsiteService) verifyWebsiteLocalCertificate(
	ctx context.Context,
	site model.Website,
	ops certificateHealthOps,
) error {
	if ops.now == nil {
		ops.now = time.Now
	}
	if ops.probe == nil {
		ops.probe = dialTLSCertificates
	}
	now := ops.now()
	domains, port, certPath, keyPath, err := s.websiteCertificateContext(site)
	if err != nil {
		return fmt.Errorf("读取配置证书: %w", err)
	}
	var serverName string
	for _, domain := range domains {
		if domain = strings.TrimSpace(domain); domain != "" && !strings.Contains(domain, "*") {
			serverName = domain
			break
		}
	}
	if serverName == "" {
		return fmt.Errorf("网站没有可用于本机 SNI 验证的具体域名")
	}
	if certPath == "" || keyPath == "" {
		return fmt.Errorf("网站未配置可读取的证书文件")
	}
	configured := inspectCertificateFiles(certPath, keyPath, domains, now)
	if configured.Status != "valid" && configured.Status != "expiring" {
		return fmt.Errorf("配置证书状态异常: %s: %s", configured.Status, configured.Error)
	}
	if port <= 0 {
		port = 443
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	local := probeTLSEndpoint(probeCtx, address, serverName, now, ops.probe, false)
	cancel()
	if local.Status != "valid" && local.Status != "expiring" {
		return fmt.Errorf("本机 TLS 证书不可用: %s: %s", local.Status, local.Error)
	}
	if configured.FingerprintSHA256 == "" || local.FingerprintSHA256 == "" || configured.FingerprintSHA256 != local.FingerprintSHA256 {
		return fmt.Errorf("配置证书与本机实际证书指纹不一致")
	}
	return nil
}

func (s *WebsiteService) websiteCertificateContext(site model.Website) ([]string, int, string, string, error) {
	domains := websiteDomains(site)
	port := site.HttpsPort
	if site.ConfigMode == "source" {
		confPath, err := s.resolveSiteConfPath(site)
		if err != nil {
			return domains, port, "", "", err
		}
		content, err := os.ReadFile(confPath)
		if err != nil {
			return domains, port, "", "", err
		}
		metadata, err := parseNginxSiteMetadata(string(content))
		if err != nil {
			return domains, port, "", "", err
		}
		domains = metadata.Domains
		if metadata.HTTPSPort > 0 {
			port = metadata.HTTPSPort
		}
		return domains, port, metadata.CertPath, metadata.KeyPath, nil
	}
	if site.CertificateID == 0 {
		return domains, port, "", "", nil
	}
	certPath, keyPath, err := NewICertificateService().ResolveCertFilePaths(site.CertificateID)
	return domains, port, certPath, keyPath, err
}

func websiteDomains(site model.Website) []string {
	seen := make(map[string]struct{})
	var domains []string
	add := func(domain string) {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			return
		}
		if _, exists := seen[domain]; exists {
			return
		}
		seen[domain] = struct{}{}
		domains = append(domains, domain)
	}
	add(site.PrimaryDomain)
	for _, domain := range strings.FieldsFunc(site.Domains, func(value rune) bool {
		return value == ',' || value == ';' || value == ' ' || value == '\n' || value == '\t'
	}) {
		add(domain)
	}
	return domains
}

func (s *WebsiteService) configuredCertificateSnapshot(site model.Website) *dto.CertificateHealthSnapshot {
	domains, _, certPath, keyPath, err := s.websiteCertificateContext(site)
	if err != nil {
		if !site.SSLEnable && site.HttpsPort == 0 {
			return nil
		}
		return &dto.CertificateHealthSnapshot{
			Status: "unreadable", CertPath: certPath, KeyPath: keyPath, Error: err.Error(),
		}
	}
	if certPath == "" || keyPath == "" {
		return nil
	}
	snapshot := inspectCertificateFiles(certPath, keyPath, domains, time.Now())
	return &snapshot
}
