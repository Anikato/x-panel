package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestNextLocalAutoRenewAtUsesFirstDailyRunAfterWindowStarts(t *testing.T) {
	location := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, location)
	cert := model.Certificate{
		Type:       "autoApply",
		SourceType: "acme",
		AutoRenew:  true,
		Pem:        "pem",
		PrivateKey: "key",
		ExpireDate: time.Date(2026, 8, 1, 10, 0, 0, 0, location),
	}

	got := nextLocalAutoRenewAt(cert, now, location)
	want := time.Date(2026, 7, 18, 2, 0, 0, 0, location)
	if got == nil || !got.Equal(want) {
		t.Fatalf("nextLocalAutoRenewAt() = %v, want %v", got, want)
	}
}

func TestNextLocalAutoRenewAtUsesNextRunInsideWindow(t *testing.T) {
	location := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 7, 20, 1, 15, 0, 0, location)
	cert := model.Certificate{
		Type:       "autoApply",
		SourceType: "acme",
		AutoRenew:  true,
		Pem:        "pem",
		PrivateKey: "key",
		ExpireDate: time.Date(2026, 7, 25, 10, 0, 0, 0, location),
	}

	got := nextLocalAutoRenewAt(cert, now, location)
	want := time.Date(2026, 7, 20, 2, 0, 0, 0, location)
	if got == nil || !got.Equal(want) {
		t.Fatalf("nextLocalAutoRenewAt() = %v, want %v", got, want)
	}
}

func TestNextLocalAutoRenewAtRejectsIneligibleCertificates(t *testing.T) {
	location := time.UTC
	now := time.Date(2026, 7, 20, 1, 15, 0, 0, location)
	base := model.Certificate{
		Type:       "autoApply",
		SourceType: "acme",
		AutoRenew:  true,
		Pem:        "pem",
		PrivateKey: "key",
		ExpireDate: now.Add(10 * 24 * time.Hour),
	}

	tests := []struct {
		name string
		cert model.Certificate
	}{
		{name: "auto renewal disabled", cert: func() model.Certificate { c := base; c.AutoRenew = false; return c }()},
		{name: "uploaded", cert: func() model.Certificate { c := base; c.Type = "upload"; return c }()},
		{name: "synchronized", cert: func() model.Certificate { c := base; c.SourceType = "synced"; return c }()},
		{name: "missing pem", cert: func() model.Certificate { c := base; c.Pem = ""; return c }()},
		{name: "missing key", cert: func() model.Certificate { c := base; c.PrivateKey = ""; return c }()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextLocalAutoRenewAt(tt.cert, now, location); got != nil {
				t.Fatalf("nextLocalAutoRenewAt() = %v, want nil", got)
			}
		})
	}
}

func TestNextCertificateSyncAtRoundsToTenMinuteCronBoundary(t *testing.T) {
	location := time.UTC
	lastSyncAt := time.Date(2026, 7, 1, 0, 3, 0, 0, location)
	source := model.CertSource{
		Enabled:      true,
		SyncInterval: 60,
		LastSyncAt:   &lastSyncAt,
	}

	got := nextCertificateSyncAt(source, time.Date(2026, 7, 1, 0, 30, 0, 0, location), location)
	want := time.Date(2026, 7, 1, 1, 10, 0, 0, location)
	if got == nil || !got.Equal(want) {
		t.Fatalf("nextCertificateSyncAt() = %v, want %v", got, want)
	}
}

func TestNextCertificateSyncAtUsesNextTickWhenOverdue(t *testing.T) {
	location := time.UTC
	lastSyncAt := time.Date(2026, 7, 1, 0, 3, 0, 0, location)
	source := model.CertSource{
		Enabled:      true,
		SyncInterval: 60,
		LastSyncAt:   &lastSyncAt,
	}

	got := nextCertificateSyncAt(source, time.Date(2026, 7, 1, 1, 12, 0, 0, location), location)
	want := time.Date(2026, 7, 1, 1, 20, 0, 0, location)
	if got == nil || !got.Equal(want) {
		t.Fatalf("nextCertificateSyncAt() = %v, want %v", got, want)
	}
}

func TestNextCertificateSyncAtRejectsInactiveSources(t *testing.T) {
	now := time.Date(2026, 7, 1, 1, 12, 0, 0, time.UTC)
	tests := []model.CertSource{
		{Enabled: false, SyncInterval: 60},
		{Enabled: true, ResumeRequired: true, SyncInterval: 60},
		{Enabled: true, SyncInterval: 0},
	}
	for _, source := range tests {
		if got := nextCertificateSyncAt(source, now, time.UTC); got != nil {
			t.Fatalf("nextCertificateSyncAt(%+v) = %v, want nil", source, got)
		}
	}
}

func TestEffectiveRenewalMetadataUsesLocalOwner(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	lastRenewedAt := now.Add(-30 * 24 * time.Hour)
	cert := model.Certificate{
		Type:              "autoApply",
		SourceType:        "acme",
		AutoRenew:         true,
		Pem:               "pem",
		PrivateKey:        "key",
		ExpireDate:        now.Add(40 * 24 * time.Hour),
		LastAutoRenewedAt: &lastRenewedAt,
	}

	autoRenew, gotLast, gotNext := effectiveRenewalMetadata(cert, now, time.UTC)
	if !autoRenew {
		t.Fatal("effective auto renewal must be enabled")
	}
	if gotLast == nil || !gotLast.Equal(lastRenewedAt) {
		t.Fatalf("last renewal = %v, want %v", gotLast, lastRenewedAt)
	}
	if gotNext == nil {
		t.Fatal("next renewal must be available")
	}
}

func TestEffectiveRenewalMetadataKeepsLocalOwnershipWhenScheduleIsUnknown(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	cert := model.Certificate{
		Type:       "autoApply",
		SourceType: "acme",
		AutoRenew:  true,
	}

	autoRenew, _, nextRenewAt := effectiveRenewalMetadata(cert, now, time.UTC)
	if !autoRenew {
		t.Fatal("local ACME auto-renew ownership must remain enabled when the schedule is unknown")
	}
	if nextRenewAt != nil {
		t.Fatalf("unknown local schedule = %v, want nil", nextRenewAt)
	}
}

func TestEffectiveRenewalMetadataPreservesUpstreamOwner(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	lastRenewedAt := now.Add(-20 * 24 * time.Hour)
	nextRenewAt := now.Add(30 * 24 * time.Hour)
	cert := model.Certificate{
		Type:                    "synced",
		SourceType:              "synced",
		AutoRenew:               false,
		UpstreamAutoRenew:       true,
		LastAutoRenewedAt:       &lastRenewedAt,
		UpstreamNextAutoRenewAt: &nextRenewAt,
	}

	autoRenew, gotLast, gotNext := effectiveRenewalMetadata(cert, now, time.UTC)
	if !autoRenew {
		t.Fatal("effective auto renewal must be inherited from upstream")
	}
	if gotLast == nil || !gotLast.Equal(lastRenewedAt) {
		t.Fatalf("last renewal = %v, want %v", gotLast, lastRenewedAt)
	}
	if gotNext == nil || !gotNext.Equal(nextRenewAt) {
		t.Fatalf("next renewal = %v, want %v", gotNext, nextRenewAt)
	}
}

func TestEffectiveRenewalMetadataDoesNotAdvertiseManualCertificateAsRenewable(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	cert := model.Certificate{
		Type:       "upload",
		SourceType: "upload",
		AutoRenew:  true,
		ExpireDate: now.Add(30 * 24 * time.Hour),
	}

	autoRenew, _, nextRenewAt := effectiveRenewalMetadata(cert, now, time.UTC)
	if autoRenew || nextRenewAt != nil {
		t.Fatalf("manual certificate advertised autoRenew=%t next=%v", autoRenew, nextRenewAt)
	}
}

func TestCertificateServerRenewalMetadataUsesOptionalJSONFields(t *testing.T) {
	now := time.Date(2026, 7, 1, 2, 0, 0, 0, time.UTC)
	item := dto.CertServerItem{
		AutoRenew:            true,
		RenewalMetadataKnown: true,
		LastAutoRenewedAt:    &now,
		NextAutoRenewAt:      &now,
	}
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal certificate server item: %v", err)
	}
	text := string(data)
	for _, field := range []string{"autoRenew", "renewalMetadataKnown", "lastAutoRenewedAt", "nextAutoRenewAt"} {
		if !strings.Contains(text, `"`+field+`"`) {
			t.Fatalf("protocol JSON %s does not contain %q", text, field)
		}
	}
}

func TestBuildCertificateRenewalPlanItemForLocalCertificate(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	lastRenewedAt := now.Add(-30 * 24 * time.Hour)
	cert := model.Certificate{
		BaseModel:         model.BaseModel{ID: 1},
		PrimaryDomain:     "local.example.com",
		Type:              "autoApply",
		SourceType:        "acme",
		AutoRenew:         true,
		Pem:               "pem",
		PrivateKey:        "key",
		ExpireDate:        now.Add(60 * 24 * time.Hour),
		LastAutoRenewedAt: &lastRenewedAt,
		Status:            "applied",
	}

	item := buildCertificateRenewalPlanItem(cert, nil, now, time.UTC)

	if item.ManagementType != renewalManagementLocal || item.SourceName != "本机" {
		t.Fatalf("management = %q source = %q", item.ManagementType, item.SourceName)
	}
	if item.NextAutoRenewAt == nil {
		t.Fatal("local auto-renew certificate must have a next renewal")
	}
	if item.LastAutoRenewedAt == nil || !item.LastAutoRenewedAt.Equal(lastRenewedAt) {
		t.Fatalf("last auto renewal = %v, want %v", item.LastAutoRenewedAt, lastRenewedAt)
	}
	if item.Status != renewalPlanScheduled {
		t.Fatalf("status = %q, want %q", item.Status, renewalPlanScheduled)
	}
}

func TestBuildCertificateRenewalPlanItemHidesScheduleWhileApplying(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	cert := model.Certificate{
		Type:       "autoApply",
		SourceType: "acme",
		AutoRenew:  true,
		Pem:        "pem",
		PrivateKey: "key",
		ExpireDate: now.Add(60 * 24 * time.Hour),
		Status:     "applying",
	}

	item := buildCertificateRenewalPlanItem(cert, nil, now, time.UTC)

	if item.Status != renewalPlanApplying {
		t.Fatalf("status = %q, want %q", item.Status, renewalPlanApplying)
	}
	if item.NextAutoRenewAt != nil {
		t.Fatalf("applying certificate schedule = %v, want nil", item.NextAutoRenewAt)
	}
}

func TestBuildCertificateRenewalPlanItemForSyncedCertificate(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 30, 0, 0, time.UTC)
	lastRenewedAt := now.Add(-30 * 24 * time.Hour)
	nextRenewAt := now.Add(30 * 24 * time.Hour)
	lastSyncAt := time.Date(2026, 7, 1, 0, 3, 0, 0, time.UTC)
	cert := model.Certificate{
		BaseModel:                    model.BaseModel{ID: 2},
		PrimaryDomain:                "synced.example.com",
		Type:                         "synced",
		SourceType:                   "synced",
		SourceID:                     9,
		SourceName:                   "primary",
		ExpireDate:                   now.Add(60 * 24 * time.Hour),
		Status:                       "applied",
		UpstreamAutoRenew:            true,
		UpstreamRenewalMetadataKnown: true,
		LastAutoRenewedAt:            &lastRenewedAt,
		UpstreamNextAutoRenewAt:      &nextRenewAt,
	}
	source := model.CertSource{
		BaseModel:    model.BaseModel{ID: 9},
		Name:         "primary",
		Enabled:      true,
		SyncInterval: 60,
		LastSyncAt:   &lastSyncAt,
	}

	item := buildCertificateRenewalPlanItem(cert, &source, now, time.UTC)

	if item.ManagementType != renewalManagementSynced || item.SourceName != "primary" {
		t.Fatalf("management = %q source = %q", item.ManagementType, item.SourceName)
	}
	if item.NextAutoRenewAt == nil || !item.NextAutoRenewAt.Equal(nextRenewAt) {
		t.Fatalf("upstream next renewal = %v, want %v", item.NextAutoRenewAt, nextRenewAt)
	}
	wantNextSync := time.Date(2026, 7, 1, 1, 10, 0, 0, time.UTC)
	if item.NextSyncAt == nil || !item.NextSyncAt.Equal(wantNextSync) {
		t.Fatalf("next sync = %v, want %v", item.NextSyncAt, wantNextSync)
	}
	if item.Status != renewalPlanScheduled {
		t.Fatalf("status = %q, want %q", item.Status, renewalPlanScheduled)
	}
}

func TestBuildCertificateRenewalPlanItemWaitsWhenUpstreamScheduleIsStale(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 30, 0, 0, time.UTC)
	staleNextRenewAt := now.Add(-time.Minute)
	cert := model.Certificate{
		Type:                         "synced",
		SourceType:                   "synced",
		ExpireDate:                   now.Add(60 * 24 * time.Hour),
		UpstreamAutoRenew:            true,
		UpstreamRenewalMetadataKnown: true,
		UpstreamNextAutoRenewAt:      &staleNextRenewAt,
	}
	source := model.CertSource{
		Enabled:      true,
		SyncInterval: 60,
	}

	item := buildCertificateRenewalPlanItem(cert, &source, now, time.UTC)

	if item.Status != renewalPlanWaitingSync {
		t.Fatalf("status = %q, want %q", item.Status, renewalPlanWaitingSync)
	}
}

func TestBuildCertificateRenewalPlanItemIdentifiesLegacyUpstream(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 30, 0, 0, time.UTC)
	nextRenewAt := now.Add(30 * 24 * time.Hour)
	cert := model.Certificate{
		Type:                    "synced",
		SourceType:              "synced",
		ExpireDate:              now.Add(60 * 24 * time.Hour),
		UpstreamAutoRenew:       true,
		UpstreamNextAutoRenewAt: &nextRenewAt,
	}
	source := model.CertSource{Enabled: true, SyncInterval: 60}

	item := buildCertificateRenewalPlanItem(cert, &source, now, time.UTC)

	if item.Status != renewalPlanWaitingSync || !strings.Contains(item.StatusMessage, "升级上游") {
		t.Fatalf("legacy upstream status=%q message=%q", item.Status, item.StatusMessage)
	}
}

func TestBuildCertificateRenewalPlanItemIdentifiesUpstreamManualMaintenance(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 30, 0, 0, time.UTC)
	cert := model.Certificate{
		Type:                         "synced",
		SourceType:                   "synced",
		ExpireDate:                   now.Add(60 * 24 * time.Hour),
		UpstreamRenewalMetadataKnown: true,
		UpstreamAutoRenew:            false,
	}
	source := model.CertSource{Enabled: true, SyncInterval: 60}

	item := buildCertificateRenewalPlanItem(cert, &source, now, time.UTC)

	if item.Status != renewalPlanManual || !strings.Contains(item.StatusMessage, "上游未启用自动续签") {
		t.Fatalf("upstream manual status=%q message=%q", item.Status, item.StatusMessage)
	}
}

func TestBuildCertificateRenewalPlanItemForExpiringManualCertificate(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	cert := model.Certificate{
		BaseModel:     model.BaseModel{ID: 3},
		PrimaryDomain: "manual.example.com",
		Type:          "upload",
		SourceType:    "upload",
		ExpireDate:    now.Add(10 * 24 * time.Hour),
		Status:        "applied",
	}

	item := buildCertificateRenewalPlanItem(cert, nil, now, time.UTC)

	if item.ManagementType != renewalManagementManual {
		t.Fatalf("management = %q, want %q", item.ManagementType, renewalManagementManual)
	}
	if item.NextAutoRenewAt != nil || item.NextSyncAt != nil {
		t.Fatalf("manual certificate must not have automatic actions: %+v", item)
	}
	if item.Status != renewalPlanExpiringManual {
		t.Fatalf("status = %q, want %q", item.Status, renewalPlanExpiringManual)
	}
}

func TestBuildCertificateRenewalPlanItemUsesNullForUnknownExpiry(t *testing.T) {
	item := buildCertificateRenewalPlanItem(
		model.Certificate{
			Type:          "autoApply",
			SourceType:    "acme",
			PrimaryDomain: "pending.example.com",
			Status:        "applying",
		},
		nil,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.UTC,
	)

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal renewal plan item: %v", err)
	}
	if !strings.Contains(string(data), `"expireDate":null`) {
		t.Fatalf("unknown expiry JSON = %s, want null", data)
	}
}

func TestBuildCertificateRenewalPlanItemForExpiredManualCertificateKeepsContext(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	cert := model.Certificate{
		BaseModel:     model.BaseModel{ID: 4},
		PrimaryDomain: "expired.example.com",
		Type:          "upload",
		SourceType:    "upload",
		ExpireDate:    now.Add(-time.Hour),
		Status:        "applied",
	}

	item := buildCertificateRenewalPlanItem(cert, nil, now, time.UTC)

	if item.Status != renewalPlanExpired {
		t.Fatalf("status = %q, want %q", item.Status, renewalPlanExpired)
	}
	if item.ManagementType != renewalManagementManual || item.SourceName != "手动导入" {
		t.Fatalf("expired manual context was lost: %+v", item)
	}
}

func TestSearchCertificateRenewalPlanFiltersAndJoinsSources(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	previousDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previousDB })
	if err := db.AutoMigrate(&model.Certificate{}, &model.CertSource{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	source := model.CertSource{
		Name:         "primary",
		ServerAddr:   "https://upstream.example.com",
		SyncInterval: 60,
		Enabled:      true,
	}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	certs := []model.Certificate{
		{PrimaryDomain: "local.example.com", Type: "autoApply", SourceType: "acme", Status: "applied"},
		{PrimaryDomain: "synced.example.com", Type: "synced", SourceType: "synced", SourceID: source.ID, SourceName: source.Name, Status: "applied"},
		{PrimaryDomain: "manual.example.com", Type: "upload", SourceType: "upload", Status: "applied"},
		{PrimaryDomain: "legacy-upload.example.com", Type: "autoApply", SourceType: "upload", Status: "applied"},
	}
	for i := range certs {
		if err := db.Create(&certs[i]).Error; err != nil {
			t.Fatalf("create certificate %d: %v", i, err)
		}
	}

	service := NewICertificateService()
	total, items, err := service.SearchRenewalPlan(dto.SearchCertRenewalPlanReq{
		PageInfo:       dto.PageInfo{Page: 1, PageSize: 20},
		ManagementType: renewalManagementSynced,
	})
	if err != nil {
		t.Fatalf("search renewal plan: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("total=%d items=%d, want 1", total, len(items))
	}
	if items[0].PrimaryDomain != "synced.example.com" || items[0].SourceName != "primary" {
		t.Fatalf("unexpected synchronized item: %+v", items[0])
	}

	total, items, err = service.SearchRenewalPlan(dto.SearchCertRenewalPlanReq{
		PageInfo:       dto.PageInfo{Page: 1, PageSize: 20},
		Info:           "manual",
		ManagementType: renewalManagementManual,
	})
	if err != nil {
		t.Fatalf("search manual renewal plan: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ManagementType != renewalManagementManual {
		t.Fatalf("unexpected manual search result: total=%d items=%+v", total, items)
	}

	total, items, err = service.SearchRenewalPlan(dto.SearchCertRenewalPlanReq{
		PageInfo:       dto.PageInfo{Page: 1, PageSize: 20},
		Info:           "legacy-upload",
		ManagementType: renewalManagementManual,
	})
	if err != nil {
		t.Fatalf("search legacy manual renewal plan: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ManagementType != renewalManagementManual {
		t.Fatalf("unexpected legacy manual search result: total=%d items=%+v", total, items)
	}
}
