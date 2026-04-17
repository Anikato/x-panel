package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/global"
	"xpanel/utils/cmd"
)

type IAppBackupService interface {
	// 备份应用
	Backup(req dto.AppBackupReq) error
	
	// 恢复应用
	Restore(req dto.AppRestoreReq) error
	
	// 备份列表
	PageBackups(req dto.AppInstallSearchReq) (int64, []dto.AppBackupDTO, error)
	
	// 删除备份
	DeleteBackup(id uint) error
}

type AppBackupService struct {
	appInstallRepo repo.IAppInstallRepo
	backupRepo     repo.IAppBackupRepo
}

func NewIAppBackupService() IAppBackupService {
	return &AppBackupService{
		appInstallRepo: repo.NewIAppInstallRepo(),
		backupRepo:     repo.NewIAppBackupRepo(),
	}
}

// Backup 备份应用
func (s *AppBackupService) Backup(req dto.AppBackupReq) error {
	ctx := context.Background()

	// 1. 获取安装信息
	install, err := s.appInstallRepo.GetFirst(repo.WithByID(req.InstallID))
	if err != nil {
		return err
	}

	// 2. 创建备份目录
	backupDir := filepath.Join(global.CONF.System.DataDir, "backups", "apps", install.App.Key, install.Name)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return buserr.WithDetail("ErrCreateBackupDir", err.Error(), err)
	}

	// 3. 生成备份文件名
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("backup-%s.tar.gz", timestamp)
	if req.BackupName != "" {
		backupName = fmt.Sprintf("%s-%s.tar.gz", req.BackupName, timestamp)
	}
	backupPath := filepath.Join(backupDir, backupName)

	// 4. 打包应用目录
	installDir := filepath.Join(global.CONF.System.DataDir, "apps", install.App.Key, install.Name)
	
	// 排除日志和临时文件
	excludes := []string{
		"--exclude=logs",
		"--exclude=*.log",
		"--exclude=cache",
	}
	
	args := append([]string{"-czf", backupPath, "-C", filepath.Dir(installDir), filepath.Base(installDir)}, excludes...)
	output, err := cmd.ExecWithTimeoutAndOutput(300*time.Second, "tar", args...)
	if err != nil {
		return buserr.WithDetail("ErrBackupFailed", output, err)
	}

	// 5. 获取文件大小
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		return err
	}

	// 6. 记录到数据库
	record := &model.AppBackupRecord{
		AppInstallID: install.ID,
		BackupName:   backupName,
		BackupPath:   backupPath,
		BackupType:   "full",
		Size:         fileInfo.Size(),
		Status:       "success",
	}

	return s.backupRepo.Create(ctx, record)
}

// Restore 恢复应用
func (s *AppBackupService) Restore(req dto.AppRestoreReq) error {
	ctx := context.Background()

	// 1. 获取备份记录
	backup, err := s.backupRepo.GetFirst(repo.WithByID(req.BackupID))
	if err != nil {
		return err
	}

	// 2. 获取安装信息
	install, err := s.appInstallRepo.GetFirst(repo.WithByID(req.InstallID))
	if err != nil {
		return err
	}

	// 3. 停止应用
	installDir := filepath.Join(global.CONF.System.DataDir, "apps", install.App.Key, install.Name)
	composeFile := filepath.Join(installDir, "docker-compose.yml")
	cmd.ExecWithTimeoutAndOutput(60*time.Second, "docker-compose", "-f", composeFile, "stop")

	// 4. 备份当前状态（用于回滚）
	rollbackPath := filepath.Join(filepath.Dir(backup.BackupPath), fmt.Sprintf("rollback-%s.tar.gz", time.Now().Format("20060102-150405")))
	cmd.ExecWithTimeoutAndOutput(300*time.Second, "tar", "-czf", rollbackPath, "-C", filepath.Dir(installDir), filepath.Base(installDir))

	// 5. 删除当前目录
	if err := os.RemoveAll(installDir); err != nil {
		return err
	}

	// 6. 解压备份
	output, err := cmd.ExecWithTimeoutAndOutput(300*time.Second, "tar", "-xzf", backup.BackupPath, "-C", filepath.Dir(installDir))
	if err != nil {
		// 回滚
		cmd.ExecWithTimeoutAndOutput(300*time.Second, "tar", "-xzf", rollbackPath, "-C", filepath.Dir(installDir))
		return buserr.WithDetail("ErrRestoreFailed", output, err)
	}

	// 7. 启动应用
	output, err = cmd.ExecWithTimeoutAndOutput(60*time.Second, "docker-compose", "-f", composeFile, "up", "-d")
	if err != nil {
		// 回滚
		os.RemoveAll(installDir)
		cmd.ExecWithTimeoutAndOutput(300*time.Second, "tar", "-xzf", rollbackPath, "-C", filepath.Dir(installDir))
		cmd.ExecWithTimeoutAndOutput(60*time.Second, "docker-compose", "-f", composeFile, "up", "-d")
		return buserr.WithDetail("ErrRestoreFailed", output, err)
	}

	// 8. 清理回滚文件
	os.Remove(rollbackPath)

	// 9. 更新状态
	install.Status = "running"
	return s.appInstallRepo.Save(ctx, &install)
}

// PageBackups 分页查询备份记录
func (s *AppBackupService) PageBackups(req dto.AppInstallSearchReq) (int64, []dto.AppBackupDTO, error) {
	var opts []repo.DBOption

	// TODO: 添加筛选条件

	total, backups, err := s.backupRepo.Page(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}

	var backupDTOs []dto.AppBackupDTO
	for _, backup := range backups {
		backupDTOs = append(backupDTOs, s.convertBackupToDTO(backup))
	}

	return total, backupDTOs, nil
}

// DeleteBackup 删除备份
func (s *AppBackupService) DeleteBackup(id uint) error {
	ctx := context.Background()

	backup, err := s.backupRepo.GetFirst(repo.WithByID(id))
	if err != nil {
		return err
	}

	// 删除备份文件
	if err := os.Remove(backup.BackupPath); err != nil {
		global.LOG.Errorf("Failed to remove backup file: %v", err)
	}

	// 删除数据库记录
	return s.backupRepo.Delete(ctx, &backup)
}

// convertBackupToDTO 转换备份记录为 DTO
func (s *AppBackupService) convertBackupToDTO(backup model.AppBackupRecord) dto.AppBackupDTO {
	sizeStr := fmt.Sprintf("%.2f MB", float64(backup.Size)/1024/1024)
	
	return dto.AppBackupDTO{
		ID:           backup.ID,
		AppInstallID: backup.AppInstallID,
		AppName:      backup.AppInstall.Name,
		BackupName:   backup.BackupName,
		BackupPath:   backup.BackupPath,
		BackupType:   backup.BackupType,
		Size:         backup.Size,
		SizeStr:      sizeStr,
		Checksum:     backup.Checksum,
		Status:       backup.Status,
		Message:      backup.Message,
		CreatedAt:    backup.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}
