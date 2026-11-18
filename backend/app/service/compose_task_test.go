package service

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"xpanel/app/model"
	"xpanel/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestComposeTaskHistoryAfterRestart(t *testing.T) {
	previous := global.DB
	t.Cleanup(func() { global.DB = previous })
	path := filepath.Join(t.TempDir(), "tasks.db")
	open := func() *gorm.DB {
		t.Helper()
		db, err := gorm.Open(sqlite.Open(path))
		if err != nil {
			t.Fatal(err)
		}
		return db
	}
	db := open()
	if err := db.AutoMigrate(&model.ComposeTask{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	running := model.ComposeTask{
		ID: "ct-active", Name: "Compose · sample · pull", Status: "running",
		Phase: "pull", Progress: 40, Log: "Pulling image", StartedAt: now,
	}
	tasks := make([]model.ComposeTask, 0, composeHistoryLimit+2)
	tasks = append(tasks, running)
	for i := 0; i <= composeHistoryLimit; i++ {
		tasks = append(tasks, model.ComposeTask{
			ID: fmt.Sprintf("ct-history-%03d", i), Name: "Compose · sample · up", Status: "success",
			Phase: "checked", Progress: 100, Log: "[project state] running(1)",
			StartedAt: now.Add(time.Duration(i-composeHistoryLimit-2) * time.Hour), EndedAt: &now,
		})
	}
	if err := db.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	global.DB = open()
	reopened, err := global.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	ctx := context.Background()
	history, err := ListFileTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != composeHistoryLimit+1 || history[0].ID != running.ID {
		t.Fatalf("expected active task plus %d recent completed tasks, found %d", composeHistoryLimit, len(history))
	}
	if err := RecoverInterruptedComposeTasks(); err != nil {
		t.Fatal(err)
	}
	task, err := GetFileTask(ctx, running.ID)
	if err != nil {
		t.Fatal(err)
	}
	if task == nil || task.Status != "interrupted" || task.EndTime < task.StartTime ||
		task.Progress != running.Progress || task.CurrentFile != running.Phase || task.Log != running.Log {
		t.Fatalf("unexpected recovered task: %+v", task)
	}
	historic, err := GetFileTask(ctx, fmt.Sprintf("ct-history-%03d", composeHistoryLimit))
	if err != nil {
		t.Fatal(err)
	}
	if historic == nil || historic.Status != "success" || historic.Progress != 100 || historic.Log != "[project state] running(1)" {
		t.Fatalf("completed task lost data: %+v", historic)
	}
	old, err := GetFileTask(ctx, "ct-history-000")
	if err != nil || old == nil || old.Status != "success" {
		t.Fatalf("older task outside the list should still be queryable, got %+v, %v", old, err)
	}
}
