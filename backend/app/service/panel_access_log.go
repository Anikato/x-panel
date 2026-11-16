package service

import (
	"net/netip"
	"sync"
	"time"

	"xpanel/app/model"
	"xpanel/global"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	panelAccessRetention  = 7 * 24 * time.Hour
	panelAccessMaxRows    = 10000
	panelAccessMaxPending = 2048
	panelAccessFlushEvery = 3 * time.Second
)

type panelAccessKey struct {
	ip     string
	kind   model.PanelAccessKind
	minute int64
}

type panelAccessBuffer struct {
	mu      sync.Mutex
	pending map[panelAccessKey]model.PanelAccessLog
}

var panelAccess = panelAccessBuffer{pending: make(map[panelAccessKey]model.PanelAccessLog)}

func RecordPanelAccess(ip string, kind model.PanelAccessKind) {
	panelAccess.record(ip, kind, time.Now().UTC())
}

func (b *panelAccessBuffer) record(ip string, kind model.PanelAccessKind, now time.Time) {
	address, err := netip.ParseAddr(ip)
	if err != nil {
		return
	}
	key := panelAccessKey{ip: address.Unmap().WithZone("").String(), kind: kind, minute: now.Unix() / 60}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.merge(key, model.PanelAccessLog{
		IP: key.ip, Kind: kind, Bucket: key.minute, Hits: 1, FirstSeen: now, LastSeen: now,
	})
}

func (b *panelAccessBuffer) merge(key panelAccessKey, row model.PanelAccessLog) {
	if pending, ok := b.pending[key]; ok {
		row.Hits += pending.Hits
		if pending.FirstSeen.Before(row.FirstSeen) {
			row.FirstSeen = pending.FirstSeen
		}
		if pending.LastSeen.After(row.LastSeen) {
			row.LastSeen = pending.LastSeen
		}
	} else if len(b.pending) >= panelAccessMaxPending {
		return
	}
	b.pending[key] = row
}

func StartPanelAccessAudit(db *gorm.DB) {
	go func() {
		ticker := time.NewTicker(panelAccessFlushEvery)
		defer ticker.Stop()
		var nextPrune time.Time
		for now := range ticker.C {
			if err := panelAccess.flush(db, now, !now.Before(nextPrune)); err != nil {
				if global.LOG != nil {
					global.LOG.Warnf("Could not store panel access audit: %v", err)
				}
				continue
			}
			if !now.Before(nextPrune) {
				nextPrune = now.Add(time.Hour)
			}
		}
	}()
}

func (b *panelAccessBuffer) flush(db *gorm.DB, now time.Time, prune bool) error {
	b.mu.Lock()
	pending := b.pending
	b.pending = make(map[panelAccessKey]model.PanelAccessLog)
	b.mu.Unlock()
	if len(pending) == 0 && !prune {
		return nil
	}

	rows := make([]model.PanelAccessLog, 0, len(pending))
	cutoff := now.Add(-panelAccessRetention).UTC()
	for _, row := range pending {
		if !row.LastSeen.Before(cutoff) {
			rows = append(rows, row)
		}
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if len(rows) > 0 {
			upsert := clause.OnConflict{
				Columns: []clause.Column{{Name: "ip"}, {Name: "kind"}, {Name: "bucket"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"hits":       gorm.Expr("panel_access_logs.hits + excluded.hits"),
					"first_seen": gorm.Expr("MIN(panel_access_logs.first_seen, excluded.first_seen)"),
					"last_seen":  gorm.Expr("MAX(panel_access_logs.last_seen, excluded.last_seen)"),
				}),
			}
			if err := tx.Clauses(upsert).CreateInBatches(rows, 100).Error; err != nil {
				return err
			}
			overflow := tx.Raw("SELECT id FROM panel_access_logs ORDER BY last_seen DESC, id DESC LIMIT -1 OFFSET ?", panelAccessMaxRows)
			if err := tx.Where("id IN (?)", overflow).Delete(&model.PanelAccessLog{}).Error; err != nil {
				return err
			}
		}
		if prune {
			return tx.Where("last_seen < ?", cutoff).Delete(&model.PanelAccessLog{}).Error
		}
		return nil
	})
	if err != nil {
		b.mu.Lock()
		for key, row := range pending {
			b.merge(key, row)
		}
		b.mu.Unlock()
	}
	return err
}

func PagePanelAccessLogs(db *gorm.DB, page, pageSize int, ip string, kind model.PanelAccessKind) (int64, []model.PanelAccessLog, error) {
	logs := make([]model.PanelAccessLog, 0, pageSize)
	query := db.Model(&model.PanelAccessLog{}).Where("last_seen >= ?", time.Now().UTC().Add(-panelAccessRetention))
	if kind != "" {
		query = query.Where("kind = ?", kind)
	}
	if ip != "" {
		address, err := netip.ParseAddr(ip)
		if err != nil {
			return 0, logs, nil
		}
		query = query.Where("ip = ?", address.Unmap().WithZone("").String())
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	offset := int64(page-1) * int64(pageSize)
	if offset >= total {
		return total, logs, nil
	}
	err := query.Order("last_seen DESC, id DESC").Offset(int(offset)).Limit(pageSize).Find(&logs).Error
	return total, logs, err
}
