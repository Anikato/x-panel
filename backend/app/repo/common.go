package repo

import (
	"xpanel/global"

	"gorm.io/gorm"
)

// DBOption 数据库查询选项
type DBOption func(*gorm.DB) *gorm.DB

// WithByID 按 ID 查询
func WithByID(id uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	}
}

// WithByKey 按 Key 查询（用于 Setting）
func WithByKey(key string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("`key` = ?", key)
	}
}

// WithByName 按 Name 查询
func WithByName(name string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ?", name)
	}
}

// WithByStatus 按 Status 查询
func WithByStatus(status string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", status)
	}
}

// WithLikeName 模糊搜索
func WithLikeName(name string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		if name == "" {
			return db
		}
		return db.Where("name LIKE ?", "%"+name+"%")
	}
}

// WithOrderByDesc 按字段降序
func WithOrderByDesc(field string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order(field + " DESC")
	}
}

// WithLikeDomain 按域名模糊搜索
func WithLikeDomain(domain string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		if domain == "" {
			return db
		}
		return db.Where("primary_domain LIKE ? OR domains LIKE ?", "%"+domain+"%", "%"+domain+"%")
	}
}

// getDB 获取全局 DB 实例
func getDB() *gorm.DB {
	return global.DB
}
