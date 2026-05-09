package repo

import (
	"strings"
	"time"

	"xpanel/app/model"
	"xpanel/global"

	"gorm.io/gorm"
)

type INotificationRepo interface {
	Create(notification *model.Notification) error
	Page(page, pageSize int, opts ...DBOption) (int64, []model.Notification, error)
	UnreadCount() (int64, error)
	MarkRead(ids []uint) error
	MarkAllRead() error
	DeleteRead() error
	Delete(id uint) error
}

func NewINotificationRepo() INotificationRepo {
	return &NotificationRepo{}
}

type NotificationRepo struct{}

func (r *NotificationRepo) Create(notification *model.Notification) error {
	return global.DB.Create(notification).Error
}

func (r *NotificationRepo) Page(page, pageSize int, opts ...DBOption) (int64, []model.Notification, error) {
	var total int64
	var items []model.Notification
	db := global.DB.Model(&model.Notification{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at desc").Find(&items).Error
	return total, items, err
}

func (r *NotificationRepo) UnreadCount() (int64, error) {
	var count int64
	err := global.DB.Model(&model.Notification{}).Where("read_at IS NULL").Count(&count).Error
	return count, err
}

func (r *NotificationRepo) MarkRead(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now()
	return global.DB.Model(&model.Notification{}).Where("id IN ?", ids).Where("read_at IS NULL").Update("read_at", now).Error
}

func (r *NotificationRepo) MarkAllRead() error {
	now := time.Now()
	return global.DB.Model(&model.Notification{}).Where("read_at IS NULL").Update("read_at", now).Error
}

func (r *NotificationRepo) DeleteRead() error {
	return global.DB.Where("read_at IS NOT NULL").Delete(&model.Notification{}).Error
}

func (r *NotificationRepo) Delete(id uint) error {
	return global.DB.Delete(&model.Notification{}, id).Error
}

func WithNotificationStatus(status string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		switch status {
		case "unread":
			return db.Where("read_at IS NULL")
		case "read":
			return db.Where("read_at IS NOT NULL")
		default:
			return db
		}
	}
}

func WithNotificationType(t string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		if t == "" {
			return db
		}
		return db.Where("type = ?", t)
	}
}

func WithNotificationSource(source string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		if source == "" {
			return db
		}
		return db.Where("source = ?", source)
	}
}

func WithNotificationKeyword(keyword string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			return db
		}
		like := "%" + keyword + "%"
		return db.Where("title LIKE ? OR content LIKE ?", like, like)
	}
}
