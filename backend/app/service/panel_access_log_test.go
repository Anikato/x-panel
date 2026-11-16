package service

import (
	"path/filepath"
	"testing"
	"time"

	"xpanel/app/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestPanelAccessAggregatesAndExpiresRecords(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "access.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.PanelAccessLog{}); err != nil {
		t.Fatal(err)
	}
	buffer := &panelAccessBuffer{pending: make(map[panelAccessKey]model.PanelAccessLog)}
	minute := time.Now().UTC().Truncate(time.Minute)
	buffer.record("192.0.2.20", model.PanelAccessEntranceBlocked, minute.Add(time.Second))
	buffer.record("::ffff:192.0.2.20", model.PanelAccessEntranceBlocked, minute.Add(2*time.Second))
	buffer.record("192.0.2.20", model.PanelAccessEntranceMatched, minute.Add(3*time.Second))
	if err := buffer.flush(db, minute.Add(3*time.Second), false); err != nil {
		t.Fatal(err)
	}
	buffer.record("192.0.2.20", model.PanelAccessEntranceBlocked, minute.Add(4*time.Second))
	buffer.record("192.0.2.20", model.PanelAccessEntranceBlocked, minute.Add(time.Minute))
	if err := buffer.flush(db, minute.Add(time.Minute), false); err != nil {
		t.Fatal(err)
	}
	total, rows, err := PagePanelAccessLogs(db, 1, 20, "192.0.2.20", model.PanelAccessEntranceBlocked)
	if err != nil || total != 2 || len(rows) != 2 || rows[0].Hits != 1 || rows[1].Hits != 3 {
		t.Fatalf("incorrect aggregation, total=%d rows=%#v err=%v", total, rows, err)
	}
	if !rows[1].FirstSeen.Equal(minute.Add(time.Second)) || !rows[1].LastSeen.Equal(minute.Add(4*time.Second)) {
		t.Fatalf("incorrect times: %#v", rows[1])
	}
	if err := buffer.flush(db, minute.Add(panelAccessRetention+2*time.Minute), true); err != nil {
		t.Fatal(err)
	}
	total, _, err = PagePanelAccessLogs(db, 1, 20, "", "")
	if err != nil || total != 0 {
		t.Fatalf("expired records remain: total=%d error=%v", total, err)
	}
}
