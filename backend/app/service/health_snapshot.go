package service

import (
	"errors"
	"sort"
	"time"

	"xpanel/app/model"
	"xpanel/constant"

	"gorm.io/gorm"
)

const (
	HealthSnapshotSchemaVersion   = 1
	HealthSnapshotFailureLimit    = 20
	HealthSnapshotRecordScanLimit = 1000
)

type HealthPanelSnapshot struct {
	Version          string `json:"version"`
	CommitHash       string `json:"commit_hash"`
	BuildTime        string `json:"build_time"`
	StartedAt        string `json:"started_at,omitempty"`
	DatabaseReadable bool   `json:"database_readable"`
}

type HealthServiceSnapshot struct {
	Name          string `json:"name"`
	Unit          string `json:"unit"`
	ActiveState   string `json:"active_state"`
	UnitFileState string `json:"unit_file_state"`
	StartedAt     string `json:"started_at,omitempty"`
}

type HealthSecuritySummary struct {
	DatabasePermissionsSecure bool     `json:"database_permissions_secure"`
	DataDirPermissionsSecure  bool     `json:"data_dir_permissions_secure"`
	Issues                    []string `json:"issues"`
}

type HealthWebsiteSummary struct {
	Total      int64 `json:"total"`
	Running    int64 `json:"running"`
	Stopped    int64 `json:"stopped"`
	SSLEnabled int64 `json:"ssl_enabled"`
}

type HealthCertificateSummary struct {
	Total          int64 `json:"total"`
	Expired        int64 `json:"expired"`
	Expiring7Days  int64 `json:"expiring_7_days"`
	Expiring15Days int64 `json:"expiring_15_days"`
	Error          int64 `json:"error"`
}

type HealthBackupFailure struct {
	Type string    `json:"type"`
	Name string    `json:"name"`
	At   time.Time `json:"at"`
}

type HealthBackupSummary struct {
	LatestSuccessAt *time.Time            `json:"latest_success_at,omitempty"`
	LatestFailureAt *time.Time            `json:"latest_failure_at,omitempty"`
	Success48Hours  int64                 `json:"success_48_hours"`
	Failed48Hours   int64                 `json:"failed_48_hours"`
	RecentFailures  []HealthBackupFailure `json:"recent_failures"`
	Truncated       bool                  `json:"truncated"`
}

type HealthCronjobFailure struct {
	ID                  uint      `json:"id"`
	Name                string    `json:"name"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastFailedAt        time.Time `json:"last_failed_at"`
}

type HealthCronjobSummary struct {
	Total              int64                  `json:"total"`
	Enabled            int64                  `json:"enabled"`
	Failed24Hours      int64                  `json:"failed_24_hours"`
	ConsecutiveFailure []HealthCronjobFailure `json:"consecutive_failures"`
	RecordsTruncated   bool                   `json:"records_truncated"`
}

type HealthStorageSnapshot struct {
	Name            string  `json:"name"`
	TotalBytes      uint64  `json:"total_bytes"`
	AvailableBytes  uint64  `json:"available_bytes"`
	UsedPercent     float64 `json:"used_percent"`
	InodesTotal     uint64  `json:"inodes_total"`
	InodesAvailable uint64  `json:"inodes_available"`
	InodesUsedPct   float64 `json:"inodes_used_percent"`
}

type HealthSnapshotData struct {
	SchemaVersion int                      `json:"schema_version"`
	Panel         HealthPanelSnapshot      `json:"panel"`
	Services      []HealthServiceSnapshot  `json:"services"`
	Websites      HealthWebsiteSummary     `json:"websites"`
	Certificates  HealthCertificateSummary `json:"certificates"`
	Backups       HealthBackupSummary      `json:"backups"`
	Cronjobs      HealthCronjobSummary     `json:"cronjobs"`
	Storage       []HealthStorageSnapshot  `json:"storage"`
	Security      HealthSecuritySummary    `json:"security"`
}

func BuildHealthSnapshot(db *gorm.DB, now time.Time) (*HealthSnapshotData, error) {
	if db == nil {
		return nil, errors.New("database is nil")
	}
	now = now.UTC()
	data := &HealthSnapshotData{
		SchemaVersion: HealthSnapshotSchemaVersion,
		Panel:         HealthPanelSnapshot{DatabaseReadable: true},
		Services:      make([]HealthServiceSnapshot, 0),
		Backups:       HealthBackupSummary{RecentFailures: make([]HealthBackupFailure, 0)},
		Cronjobs:      HealthCronjobSummary{ConsecutiveFailure: make([]HealthCronjobFailure, 0)},
		Storage:       make([]HealthStorageSnapshot, 0),
		Security: HealthSecuritySummary{
			DatabasePermissionsSecure: true,
			DataDirPermissionsSecure:  true,
			Issues:                    make([]string, 0),
		},
	}

	if err := loadWebsiteHealth(db, &data.Websites); err != nil {
		return nil, err
	}
	if err := loadCertificateHealth(db, now, &data.Certificates); err != nil {
		return nil, err
	}
	if err := loadBackupHealth(db, now, &data.Backups); err != nil {
		return nil, err
	}
	if err := loadCronjobHealth(db, now, &data.Cronjobs); err != nil {
		return nil, err
	}
	return data, nil
}

func loadWebsiteHealth(db *gorm.DB, summary *HealthWebsiteSummary) error {
	if err := db.Model(&model.Website{}).Count(&summary.Total).Error; err != nil {
		return err
	}
	if err := db.Model(&model.Website{}).Where("status = ?", "running").Count(&summary.Running).Error; err != nil {
		return err
	}
	if err := db.Model(&model.Website{}).Where("status <> ?", "running").Count(&summary.Stopped).Error; err != nil {
		return err
	}
	return db.Model(&model.Website{}).Where("ssl_enable = ?", true).Count(&summary.SSLEnabled).Error
}

func loadCertificateHealth(db *gorm.DB, now time.Time, summary *HealthCertificateSummary) error {
	validAfter := time.Unix(0, 0).UTC()
	if err := db.Model(&model.Certificate{}).Count(&summary.Total).Error; err != nil {
		return err
	}
	if err := db.Model(&model.Certificate{}).
		Where("expire_date > ? AND expire_date <= ?", validAfter, now).
		Count(&summary.Expired).Error; err != nil {
		return err
	}
	if err := db.Model(&model.Certificate{}).
		Where("expire_date > ? AND expire_date <= ?", now, now.Add(7*24*time.Hour)).
		Count(&summary.Expiring7Days).Error; err != nil {
		return err
	}
	if err := db.Model(&model.Certificate{}).
		Where("expire_date > ? AND expire_date <= ?", now, now.Add(15*24*time.Hour)).
		Count(&summary.Expiring15Days).Error; err != nil {
		return err
	}
	return db.Model(&model.Certificate{}).Where("status = ?", "error").Count(&summary.Error).Error
}

func loadBackupHealth(db *gorm.DB, now time.Time, summary *HealthBackupSummary) error {
	latest := func(status string) (*time.Time, error) {
		var row model.BackupRecord
		err := db.Select("created_at").Where("status = ?", status).Order("created_at DESC").First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		value := row.CreatedAt.UTC()
		return &value, nil
	}
	var err error
	if summary.LatestSuccessAt, err = latest(constant.StatusSuccess); err != nil {
		return err
	}
	if summary.LatestFailureAt, err = latest(constant.StatusFailed); err != nil {
		return err
	}
	cutoff := now.Add(-48 * time.Hour)
	if err := db.Model(&model.BackupRecord{}).Where("status = ? AND created_at >= ?", constant.StatusSuccess, cutoff).Count(&summary.Success48Hours).Error; err != nil {
		return err
	}
	if err := db.Model(&model.BackupRecord{}).Where("status = ? AND created_at >= ?", constant.StatusFailed, cutoff).Count(&summary.Failed48Hours).Error; err != nil {
		return err
	}
	var rows []model.BackupRecord
	if err := db.Select("type", "name", "created_at").Where("status = ?", constant.StatusFailed).
		Order("created_at DESC").Limit(HealthSnapshotFailureLimit + 1).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) > HealthSnapshotFailureLimit {
		summary.Truncated = true
		rows = rows[:HealthSnapshotFailureLimit]
	}
	for _, row := range rows {
		summary.RecentFailures = append(summary.RecentFailures, HealthBackupFailure{
			Type: row.Type,
			Name: row.Name,
			At:   row.CreatedAt.UTC(),
		})
	}
	return nil
}

func loadCronjobHealth(db *gorm.DB, now time.Time, summary *HealthCronjobSummary) error {
	if err := db.Model(&model.Cronjob{}).Count(&summary.Total).Error; err != nil {
		return err
	}
	if err := db.Model(&model.Cronjob{}).Where("status = ?", constant.StatusEnable).Count(&summary.Enabled).Error; err != nil {
		return err
	}
	if err := db.Model(&model.CronjobRecord{}).
		Where("status = ? AND created_at >= ?", constant.StatusFailed, now.Add(-24*time.Hour)).
		Count(&summary.Failed24Hours).Error; err != nil {
		return err
	}

	var jobs []model.Cronjob
	if err := db.Select("id", "name").Find(&jobs).Error; err != nil {
		return err
	}
	names := make(map[uint]string, len(jobs))
	for _, job := range jobs {
		names[job.ID] = job.Name
	}

	var records []model.CronjobRecord
	if err := db.Select("cronjob_id", "status", "created_at").Order("created_at DESC").
		Limit(HealthSnapshotRecordScanLimit + 1).Find(&records).Error; err != nil {
		return err
	}
	if len(records) > HealthSnapshotRecordScanLimit {
		summary.RecordsTruncated = true
		records = records[:HealthSnapshotRecordScanLimit]
	}
	type failureState struct {
		count int
		at    time.Time
		stop  bool
	}
	states := make(map[uint]*failureState)
	for _, record := range records {
		state := states[record.CronjobID]
		if state == nil {
			state = &failureState{}
			states[record.CronjobID] = state
		}
		if state.stop {
			continue
		}
		if record.Status != constant.StatusFailed {
			state.stop = true
			continue
		}
		if state.count == 0 {
			state.at = record.CreatedAt.UTC()
		}
		state.count++
	}
	for id, state := range states {
		if state.count == 0 {
			continue
		}
		summary.ConsecutiveFailure = append(summary.ConsecutiveFailure, HealthCronjobFailure{
			ID:                  id,
			Name:                names[id],
			ConsecutiveFailures: state.count,
			LastFailedAt:        state.at,
		})
	}
	sort.Slice(summary.ConsecutiveFailure, func(i, j int) bool {
		return summary.ConsecutiveFailure[i].LastFailedAt.After(summary.ConsecutiveFailure[j].LastFailedAt)
	})
	if len(summary.ConsecutiveFailure) > HealthSnapshotFailureLimit {
		summary.ConsecutiveFailure = summary.ConsecutiveFailure[:HealthSnapshotFailureLimit]
		summary.RecordsTruncated = true
	}
	return nil
}
