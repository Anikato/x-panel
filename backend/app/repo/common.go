package repo

import (
	"context"

	"xpanel/global"
	"gorm.io/gorm"
)

// DBOption 数据库查询选项函数
type DBOption func(*gorm.DB) *gorm.DB

// WithByID 根据 ID 查询
func WithByID(id uint) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("id = ?", id)
	}
}

// WithByIDs 根据 ID 列表查询
func WithByIDs(ids []uint) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("id in (?)", ids)
	}
}

// WithByName 根据名称查询
func WithByName(name string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("name = ?", name)
	}
}

// WithByKey 根据 key 查询
func WithByKey(key string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("`key` = ?", key)
	}
}

// WithByLikeName 根据名称模糊查询
func WithByLikeName(name string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		if len(name) == 0 {
			return g
		}
		return g.Where("name like ?", "%"+name+"%")
	}
}

// WithByType 根据类型查询
func WithByType(tp string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		if len(tp) == 0 {
			return g
		}
		return g.Where("`type` = ?", tp)
	}
}

// WithByStatus 根据状态查询
func WithByStatus(status string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		if len(status) == 0 {
			return g
		}
		return g.Where("status = ?", status)
	}
}

// WithByAppID 根据应用 ID 查询
func WithByAppID(appID uint) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("app_id = ?", appID)
	}
}

// WithByVersion 根据版本查询
func WithByVersion(version string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("version = ?", version)
	}
}

// WithBySourceID 根据源 ID 查询
func WithBySourceID(sourceID uint) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("source_id = ?", sourceID)
	}
}

// WithLikeName 根据名称模糊查询（别名）
func WithLikeName(name string) DBOption {
	return WithByLikeName(name)
}

// WithLikeDomain 根据域名模糊查询
func WithLikeDomain(domain string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		if len(domain) == 0 {
			return g
		}
		return g.Where("primary_domain like ? OR domains like ?", "%"+domain+"%", "%"+domain+"%")
	}
}

// WithOrderBy 排序
func WithOrderBy(field string, desc bool) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		if desc {
			return g.Order(field + " desc")
		}
		return g.Order(field + " asc")
	}
}

// getDb 获取数据库实例并应用选项
func getDb(opts ...DBOption) *gorm.DB {
	db := global.DB
	for _, opt := range opts {
		db = opt(db)
	}
	return db
}

// getTx 从 context 获取事务或返回普通 DB
func getTx(ctx context.Context, opts ...DBOption) *gorm.DB {
	tx, ok := ctx.Value("DB").(*gorm.DB)
	if ok {
		for _, opt := range opts {
			tx = opt(tx)
		}
		return tx
	}
	return getDb(opts...)
}
