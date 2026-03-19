package repo

import (
	"time"

	"xpanel/app/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ITrafficRepo interface {
	// Config CRUD
	ListConfigs() ([]model.TrafficConfig, error)
	GetConfig(interfaceName string) (model.TrafficConfig, error)
	SaveConfig(item *model.TrafficConfig) error
	DeleteConfig(interfaceName string) error

	// Snapshot
	GetSnapshot(interfaceName string) (model.TrafficSnapshot, error)
	SaveSnapshot(item *model.TrafficSnapshot) error
	DeleteSnapshot(interfaceName string) error

	// Hourly records
	UpsertHourly(interfaceName string, ts time.Time, deltaSent, deltaRecv uint64) error
	SumTraffic(interfaceName string, start, end time.Time) (uint64, uint64, error)
	ListHourly(interfaceName string, start, end time.Time) ([]model.TrafficHourly, error)
	DeleteHourlyBefore(t time.Time) error
}

func NewITrafficRepo() ITrafficRepo { return &TrafficRepo{} }

type TrafficRepo struct{}

func (r *TrafficRepo) ListConfigs() ([]model.TrafficConfig, error) {
	var items []model.TrafficConfig
	err := getDB().Order("created_at ASC").Find(&items).Error
	return items, err
}

func (r *TrafficRepo) GetConfig(interfaceName string) (model.TrafficConfig, error) {
	var item model.TrafficConfig
	err := getDB().Where("interface_name = ?", interfaceName).First(&item).Error
	return item, err
}

func (r *TrafficRepo) SaveConfig(item *model.TrafficConfig) error {
	return getDB().Save(item).Error
}

func (r *TrafficRepo) DeleteConfig(interfaceName string) error {
	return getDB().Where("interface_name = ?", interfaceName).Delete(&model.TrafficConfig{}).Error
}

func (r *TrafficRepo) GetSnapshot(interfaceName string) (model.TrafficSnapshot, error) {
	var item model.TrafficSnapshot
	err := getDB().Where("interface_name = ?", interfaceName).First(&item).Error
	return item, err
}

func (r *TrafficRepo) SaveSnapshot(item *model.TrafficSnapshot) error {
	return getDB().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "interface_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"bytes_sent", "bytes_recv", "sampled_at"}),
	}).Create(item).Error
}

func (r *TrafficRepo) DeleteSnapshot(interfaceName string) error {
	return getDB().Where("interface_name = ?", interfaceName).Delete(&model.TrafficSnapshot{}).Error
}

func (r *TrafficRepo) UpsertHourly(interfaceName string, ts time.Time, deltaSent, deltaRecv uint64) error {
	hourStart := time.Date(ts.Year(), ts.Month(), ts.Day(), ts.Hour(), 0, 0, 0, ts.Location())

	result := getDB().Model(&model.TrafficHourly{}).
		Where("interface_name = ? AND timestamp = ?", interfaceName, hourStart).
		Updates(map[string]interface{}{
			"bytes_sent": gorm.Expr("bytes_sent + ?", deltaSent),
			"bytes_recv": gorm.Expr("bytes_recv + ?", deltaRecv),
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return getDB().Create(&model.TrafficHourly{
			InterfaceName: interfaceName,
			Timestamp:     hourStart,
			BytesSent:     deltaSent,
			BytesRecv:     deltaRecv,
		}).Error
	}
	return nil
}

func (r *TrafficRepo) SumTraffic(interfaceName string, start, end time.Time) (uint64, uint64, error) {
	var result struct {
		TotalSent uint64
		TotalRecv uint64
	}
	err := getDB().Model(&model.TrafficHourly{}).
		Select("COALESCE(SUM(bytes_sent), 0) as total_sent, COALESCE(SUM(bytes_recv), 0) as total_recv").
		Where("interface_name = ? AND timestamp >= ? AND timestamp < ?", interfaceName, start, end).
		Scan(&result).Error
	return result.TotalSent, result.TotalRecv, err
}

func (r *TrafficRepo) ListHourly(interfaceName string, start, end time.Time) ([]model.TrafficHourly, error) {
	var items []model.TrafficHourly
	err := getDB().
		Where("interface_name = ? AND timestamp >= ? AND timestamp < ?", interfaceName, start, end).
		Order("timestamp ASC").
		Find(&items).Error
	return items, err
}

func (r *TrafficRepo) DeleteHourlyBefore(t time.Time) error {
	return getDB().Where("timestamp < ?", t).Delete(&model.TrafficHourly{}).Error
}
