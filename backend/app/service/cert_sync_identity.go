package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"xpanel/app/dto"
	"xpanel/app/model"
)

func canonicalCertificateDomains(primary, domains, dnsNames string) string {
	values := []string{primary}
	values = append(values, strings.Split(domains, ",")...)
	var decoded []string
	if json.Unmarshal([]byte(dnsNames), &decoded) == nil {
		values = append(values, decoded...)
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".")
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return strings.Join(result, ",")
}

func selectLegacyCertificate(remote dto.CertServerItem, candidates []model.Certificate) (model.Certificate, error) {
	return selectLegacyCertificateWithReferences(remote, candidates, nil)
}

func selectLegacyCertificateWithReferences(remote dto.CertServerItem, candidates []model.Certificate, referenced map[uint]bool) (model.Certificate, error) {
	var fingerprintMatches []model.Certificate
	for _, candidate := range candidates {
		if remote.Fingerprint != "" && candidate.Fingerprint == remote.Fingerprint {
			fingerprintMatches = append(fingerprintMatches, candidate)
		}
	}
	if len(fingerprintMatches) == 1 {
		return fingerprintMatches[0], nil
	}
	if len(fingerprintMatches) > 1 {
		return model.Certificate{}, fmt.Errorf("identity_ambiguous: multiple certificates have the upstream fingerprint")
	}

	remoteDomains := canonicalCertificateDomains(remote.PrimaryDomain, remote.Domains, remote.DNSNames)
	var domainMatches []model.Certificate
	for _, candidate := range candidates {
		if canonicalCertificateDomains(candidate.PrimaryDomain, candidate.Domains, candidate.DNSNames) == remoteDomains {
			domainMatches = append(domainMatches, candidate)
		}
	}
	if len(domainMatches) == 1 {
		return domainMatches[0], nil
	}
	if len(domainMatches) > 1 {
		var referencedMatches []model.Certificate
		for _, candidate := range domainMatches {
			if referenced[candidate.ID] {
				referencedMatches = append(referencedMatches, candidate)
			}
		}
		if len(referencedMatches) == 1 {
			return referencedMatches[0], nil
		}
		return model.Certificate{}, fmt.Errorf("identity_ambiguous: multiple same-source certificates have the same SAN set")
	}
	return model.Certificate{}, fmt.Errorf("legacy certificate candidate not found")
}
