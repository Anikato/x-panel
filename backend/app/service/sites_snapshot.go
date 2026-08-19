package service

import (
	"strings"
	"time"

	"xpanel/app/model"

	"gorm.io/gorm"
)

const SitesSnapshotLimit = 500

type SiteSnapshot struct {
	ID            uint       `json:"id"`
	Alias         string     `json:"alias"`
	PrimaryDomain string     `json:"primary_domain"`
	Domains       []string   `json:"domains"`
	Type          string     `json:"type"`
	Status        string     `json:"status"`
	SSLEnable     bool       `json:"ssl_enable"`
	ExpireAt      *time.Time `json:"expire_at"`
	SourceType    string     `json:"source_type"`
	SourceName    string     `json:"source_name,omitempty"`
}

type SitesSnapshotData struct {
	Sites     []SiteSnapshot `json:"sites"`
	Truncated bool           `json:"truncated"`
}

func MapCertificateSourceType(cert *model.Certificate, bound bool) string {
	if !bound || cert == nil {
		return "none"
	}
	if cert.Type == "synced" || cert.SourceType == "synced" {
		return "synced"
	}
	if cert.Type == "upload" || cert.SourceType == "upload" {
		return "upload"
	}
	return "acme"
}

func snapshotDomains(primary, extra string) []string {
	out := make([]string, 0)
	seen := map[string]struct{}{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(primary)
	for _, part := range strings.Split(extra, ",") {
		add(part)
	}
	return out
}

func BuildSitesSnapshot(db *gorm.DB) (*SitesSnapshotData, error) {
	var websites []model.Website
	if err := db.Order("id").Limit(SitesSnapshotLimit + 1).Find(&websites).Error; err != nil {
		return nil, err
	}
	truncated := len(websites) > SitesSnapshotLimit
	if truncated {
		websites = websites[:SitesSnapshotLimit]
	}

	certIDs := make([]uint, 0)
	seenID := map[uint]struct{}{}
	for _, w := range websites {
		if !w.SSLEnable || w.CertificateID == 0 {
			continue
		}
		if _, ok := seenID[w.CertificateID]; ok {
			continue
		}
		seenID[w.CertificateID] = struct{}{}
		certIDs = append(certIDs, w.CertificateID)
	}

	certs := make(map[uint]model.Certificate, len(certIDs))
	if len(certIDs) > 0 {
		var rows []model.Certificate
		if err := db.Where("id IN ?", certIDs).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, c := range rows {
			certs[c.ID] = c
		}
	}

	sites := make([]SiteSnapshot, 0, len(websites))
	for _, w := range websites {
		bound := w.SSLEnable && w.CertificateID != 0
		var cert *model.Certificate
		if bound {
			if c, ok := certs[w.CertificateID]; ok {
				copied := c
				cert = &copied
			} else {
				bound = false
			}
		}
		snap := SiteSnapshot{
			ID:            w.ID,
			Alias:         w.Alias,
			PrimaryDomain: w.PrimaryDomain,
			Domains:       snapshotDomains(w.PrimaryDomain, w.Domains),
			Type:          w.Type,
			Status:        w.Status,
			SSLEnable:     w.SSLEnable,
			SourceType:    MapCertificateSourceType(cert, bound),
		}
		if bound && cert != nil && !cert.ExpireDate.IsZero() {
			expire := cert.ExpireDate.UTC()
			snap.ExpireAt = &expire
		}
		if snap.SourceType == "synced" && cert != nil {
			snap.SourceName = cert.SourceName
		}
		sites = append(sites, snap)
	}

	return &SitesSnapshotData{Sites: sites, Truncated: truncated}, nil
}
