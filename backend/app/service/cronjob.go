package service

import (
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
	archiveUtil "xpanel/utils/backup"

	"github.com/robfig/cron/v3"
)

type ICronjobService interface {
	Create(req dto.CronjobCreate) error
	Update(req dto.CronjobUpdate) error
	Delete(id uint) error
	Get(id uint) (*dto.CronjobInfo, error)
	SearchWithPage(req dto.CronjobSearch) (int64, []dto.CronjobInfo, error)
	UpdateStatus(id uint, status string) error
	HandleOnce(id uint) error
	SearchRecords(req dto.CronjobRecordSearch) (int64, []dto.CronjobRecordInfo, error)
	StartAllJobs()
}

func NewICronjobService() ICronjobService {
	return &CronjobService{
		cronjobRepo: repo.NewICronjobRepo(),
	}
}

type CronjobService struct {
	cronjobRepo repo.ICronjobRepo
}

func (s *CronjobService) Create(req dto.CronjobCreate) error {
	job := &model.Cronjob{
		Name:            req.Name,
		Type:            req.Type,
		Spec:            req.Spec,
		Status:          constant.StatusEnable,
		Script:          req.Script,
		URL:             req.URL,
		Website:         req.Website,
		DBType:          req.DBType,
		DBName:          req.DBName,
		DBInstanceID:    req.DBInstanceID,
		SourceDir:       req.SourceDir,
		TargetAccountID: req.TargetAccountID,
		RetainCopies:    req.RetainCopies,
		ExclusionRules:  req.ExclusionRules,
		CompressFormat:  req.CompressFormat,
		EncryptPassword: req.EncryptPassword,
	}
	if err := s.validateJobConfig(job); err != nil {
		return err
	}
	if err := s.cronjobRepo.Create(job); err != nil {
		return err
	}
	if err := s.addCronJob(job); err != nil {
		return err
	}
	return nil
}

func (s *CronjobService) Update(req dto.CronjobUpdate) error {
	job, err := s.cronjobRepo.Get(req.ID)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	s.removeCronJob(job)
	fields := map[string]interface{}{
		"name":              req.Name,
		"type":              req.Type,
		"spec":              req.Spec,
		"script":            req.Script,
		"url":               req.URL,
		"website":           req.Website,
		"db_type":           req.DBType,
		"db_name":           req.DBName,
		"db_instance_id":    req.DBInstanceID,
		"source_dir":        req.SourceDir,
		"target_account_id": req.TargetAccountID,
		"retain_copies":     req.RetainCopies,
		"exclusion_rules":   req.ExclusionRules,
		"compress_format":   req.CompressFormat,
		"encrypt_password":  req.EncryptPassword,
	}
	updatedJob := *job
	updatedJob.Name = req.Name
	updatedJob.Type = req.Type
	updatedJob.Spec = req.Spec
	updatedJob.Script = req.Script
	updatedJob.URL = req.URL
	updatedJob.Website = req.Website
	updatedJob.DBType = req.DBType
	updatedJob.DBName = req.DBName
	updatedJob.DBInstanceID = req.DBInstanceID
	updatedJob.SourceDir = req.SourceDir
	updatedJob.TargetAccountID = req.TargetAccountID
	updatedJob.RetainCopies = req.RetainCopies
	updatedJob.ExclusionRules = req.ExclusionRules
	updatedJob.CompressFormat = req.CompressFormat
	updatedJob.EncryptPassword = req.EncryptPassword
	if err := s.validateJobConfig(&updatedJob); err != nil {
		return err
	}
	if err := s.cronjobRepo.Update(req.ID, fields); err != nil {
		return err
	}
	if job.Status == constant.StatusEnable {
		updated, _ := s.cronjobRepo.Get(req.ID)
		if updated != nil {
			_ = s.addCronJob(updated)
		}
	}
	return nil
}

func (s *CronjobService) Delete(id uint) error {
	job, err := s.cronjobRepo.Get(id)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	s.removeCronJob(job)
	_ = s.cronjobRepo.DeleteRecordByCronjobID(id)
	return s.cronjobRepo.Delete(id)
}

func (s *CronjobService) Get(id uint) (*dto.CronjobInfo, error) {
	job, err := s.cronjobRepo.Get(id)
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}
	return toCronjobInfo(job), nil
}

func (s *CronjobService) SearchWithPage(req dto.CronjobSearch) (int64, []dto.CronjobInfo, error) {
	opts := []repo.DBOption{
		repo.WithCronjobType(req.Type),
		repo.WithCronjobStatus(req.Status),
	}
	if req.Info != "" {
		opts = append(opts, repo.WithLikeName(req.Info))
	}
	total, jobs, err := s.cronjobRepo.Page(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}
	var items []dto.CronjobInfo
	for _, j := range jobs {
		items = append(items, *toCronjobInfo(&j))
	}
	return total, items, nil
}

func (s *CronjobService) UpdateStatus(id uint, status string) error {
	job, err := s.cronjobRepo.Get(id)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if status == constant.StatusEnable {
		_ = s.addCronJob(job)
	} else {
		s.removeCronJob(job)
	}
	return s.cronjobRepo.Update(id, map[string]interface{}{"status": status})
}

func (s *CronjobService) HandleOnce(id uint) error {
	job, err := s.cronjobRepo.Get(id)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	go s.executeJob(job)
	return nil
}

func (s *CronjobService) SearchRecords(req dto.CronjobRecordSearch) (int64, []dto.CronjobRecordInfo, error) {
	opts := []repo.DBOption{
		repo.WithCronjobID(req.CronjobID),
		repo.WithRecordStatus(req.Status),
	}
	total, records, err := s.cronjobRepo.PageRecord(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}
	var items []dto.CronjobRecordInfo
	for _, r := range records {
		items = append(items, dto.CronjobRecordInfo{
			ID:        r.ID,
			CronjobID: r.CronjobID,
			StartTime: r.StartTime,
			Duration:  r.Duration,
			Status:    r.Status,
			Message:   r.Message,
			File:      r.File,
		})
	}
	return total, items, nil
}

func (s *CronjobService) StartAllJobs() {
	jobs, err := s.cronjobRepo.List(repo.WithCronjobStatus(constant.StatusEnable))
	if err != nil {
		global.LOG.Errorf("load cronjobs failed: %v", err)
		return
	}
	for i := range jobs {
		if err := s.addCronJob(&jobs[i]); err != nil {
			global.LOG.Errorf("add cronjob [%s] failed: %v", jobs[i].Name, err)
		}
	}
}

func (s *CronjobService) addCronJob(job *model.Cronjob) error {
	if global.CRON == nil {
		return nil
	}
	entryID, err := global.CRON.AddFunc(job.Spec, func() {
		s.executeJob(job)
	})
	if err != nil {
		return err
	}
	return s.cronjobRepo.Update(job.ID, map[string]interface{}{"entry_id": int(entryID)})
}

func (s *CronjobService) validateJobConfig(job *model.Cronjob) error {
	if strings.TrimSpace(job.Spec) == "" {
		return fmt.Errorf("cron spec is empty")
	}
	if _, err := cron.ParseStandard(job.Spec); err != nil {
		return fmt.Errorf("invalid cron spec: %v", err)
	}
	switch job.Type {
	case "shell":
		if strings.TrimSpace(job.Script) == "" {
			return fmt.Errorf("script is empty")
		}
	case "curl":
		if strings.TrimSpace(job.URL) == "" {
			return fmt.Errorf("url is empty")
		}
	case "database":
		if strings.TrimSpace(job.DBType) == "" {
			return fmt.Errorf("database type is empty")
		}
		if job.DBInstanceID == 0 && strings.TrimSpace(job.DBName) == "" {
			return fmt.Errorf("database instance is empty")
		}
	case "website":
		if strings.TrimSpace(job.Website) == "" {
			return fmt.Errorf("website name is empty")
		}
	case "directory":
		if strings.TrimSpace(job.SourceDir) == "" {
			return fmt.Errorf("source directory is empty")
		}
	default:
		return fmt.Errorf("unsupported job type: %s", job.Type)
	}
	return nil
}

func (s *CronjobService) removeCronJob(job *model.Cronjob) {
	if global.CRON != nil && job.EntryID > 0 {
		global.CRON.Remove(cron.EntryID(job.EntryID))
	}
}

func (s *CronjobService) executeJob(job *model.Cronjob) {
	start := time.Now()
	var msg string
	var status string
	var file string

	switch job.Type {
	case "shell":
		msg, status = s.execShell(job)
	case "curl":
		msg, status = s.execCurl(job)
	case "database":
		msg, status, file = s.execDatabaseBackup(job)
	case "website":
		msg, status, file = s.execWebsiteBackup(job)
	case "directory":
		msg, status, file = s.execDirectoryBackup(job)
	default:
		msg = fmt.Sprintf("unsupported job type: %s", job.Type)
		status = constant.StatusFailed
	}

	duration := time.Since(start).Seconds()
	record := &model.CronjobRecord{
		CronjobID: job.ID,
		StartTime: start,
		Duration:  duration,
		Status:    status,
		Message:   msg,
		File:      file,
	}
	if err := s.cronjobRepo.CreateRecord(record); err != nil {
		global.LOG.Errorf("save cronjob record failed: %v", err)
	}
	s.notifyJobResult(job, status, msg)
	if job.RetainCopies > 0 {
		_ = s.cronjobRepo.CleanRecords(job.ID, int(job.RetainCopies))
		_ = NewIBackupService().CleanSuccessfulRecords(job.ID, job.RetainCopies)
	}
}

func (s *CronjobService) notifyJobResult(job *model.Cronjob, status, message string) {
	notificationType := "success"
	title := fmt.Sprintf("计划任务「%s」执行成功", job.Name)
	if status != constant.StatusSuccess {
		notificationType = "error"
		title = fmt.Sprintf("计划任务「%s」执行失败", job.Name)
	}
	content := strings.TrimSpace(message)
	if len(content) > 500 {
		content = content[:500] + "\n...(truncated)"
	}
	CreateNotification(dto.NotificationCreate{
		Type:      notificationType,
		Event:     "cronjob." + status,
		Title:     title,
		Content:   content,
		Source:    "cronjob",
		TargetURL: "/cronjob",
	})
}

func (s *CronjobService) execShell(job *model.Cronjob) (string, string) {
	cmd := exec.Command("bash", "-c", job.Script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output) + "\n" + err.Error(), constant.StatusFailed
	}
	result := strings.TrimSpace(string(output))
	if len(result) > 10000 {
		result = result[:10000] + "\n...(truncated)"
	}
	return result, constant.StatusSuccess
}

func (s *CronjobService) execCurl(job *model.Cronjob) (string, string) {
	client := http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(job.URL)
	if err != nil {
		return err.Error(), constant.StatusFailed
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Sprintf("HTTP %d", resp.StatusCode), constant.StatusFailed
	}
	return fmt.Sprintf("HTTP %d", resp.StatusCode), constant.StatusSuccess
}

func (s *CronjobService) execDatabaseBackup(job *model.Cronjob) (string, string, string) {
	if job.DBType == "" {
		return "database type is empty", constant.StatusFailed, ""
	}
	backupService := NewIBackupService()
	dbRepo := repo.NewIDatabaseRepo()
	if job.DBInstanceID > 0 {
		instance, _, err := dbRepo.GetInstanceWithServer(job.DBInstanceID, job.DBType)
		if err != nil {
			return fmt.Sprintf("database instance [%d] not found", job.DBInstanceID), constant.StatusFailed, ""
		}
		return s.backupOneDatabase(job, backupService, instance.ID, instance.Name)
	}
	if job.DBName == "" {
		return "database name is empty", constant.StatusFailed, ""
	}
	servers, _ := dbRepo.ListServers(repo.WithServerType(job.DBType))
	if len(servers) == 0 {
		return fmt.Sprintf("no %s server found", job.DBType), constant.StatusFailed, ""
	}
	if isAllDatabases(job.DBName) {
		successCount := 0
		failedCount := 0
		var lastFile string
		var messages []string
		for _, server := range servers {
			instances, _ := dbRepo.ListInstancesByServerID(server.ID)
			for _, inst := range instances {
				msg, status, file := s.backupOneDatabase(job, backupService, inst.ID, inst.Name)
				messages = append(messages, fmt.Sprintf("%s: %s", inst.Name, msg))
				if status == constant.StatusSuccess {
					successCount++
					lastFile = file
				} else {
					failedCount++
				}
			}
		}
		if successCount == 0 {
			if len(messages) == 0 {
				return "no database instances found", constant.StatusFailed, ""
			}
			return strings.Join(messages, "\n"), constant.StatusFailed, ""
		}
		summary := fmt.Sprintf("backup %d database(s), failed %d database(s)\n%s", successCount, failedCount, strings.Join(messages, "\n"))
		if failedCount > 0 {
			return summary, constant.StatusFailed, lastFile
		}
		return summary, constant.StatusSuccess, lastFile
	}
	var matches []model.DatabaseInstance
	for _, server := range servers {
		instances, _ := dbRepo.ListInstancesByServerID(server.ID)
		for _, inst := range instances {
			if inst.Name == job.DBName {
				matches = append(matches, inst)
			}
		}
	}
	if len(matches) == 1 {
		return s.backupOneDatabase(job, backupService, matches[0].ID, matches[0].Name)
	}
	if len(matches) > 1 {
		return fmt.Sprintf("database instance [%s] is ambiguous, please select a specific instance", job.DBName), constant.StatusFailed, ""
	}
	return fmt.Sprintf("database instance [%s] not found", job.DBName), constant.StatusFailed, ""
}

func (s *CronjobService) backupOneDatabase(job *model.Cronjob, backupService IBackupService, instanceID uint, dbName string) (string, string, string) {
	if job.TargetAccountID > 0 {
		output, err := backupService.PerformDatabaseInstanceBackupWithInfo(instanceID, job.TargetAccountID)
		if err != nil {
			_ = backupService.CreateRecordForFile("database", dbName, job.TargetAccountID, job.ID, "", 0, constant.StatusFailed, err.Error())
			return fmt.Sprintf("backup failed: %v", err), constant.StatusFailed, ""
		}
		_ = backupService.CreateRecordForFile("database", dbName, job.TargetAccountID, job.ID, output.Path, output.Size, constant.StatusSuccess, output.Path)
		return fmt.Sprintf("backup uploaded: %s", output.Path), constant.StatusSuccess, output.Path
	}
	dbService := NewIDatabaseService()
	outFile, err := dbService.BackupInstance(instanceID)
	if err != nil {
		_ = backupService.CreateRecordForFile("database", dbName, 0, job.ID, "", 0, constant.StatusFailed, err.Error())
		return fmt.Sprintf("backup failed: %v", err), constant.StatusFailed, ""
	}
	_ = backupService.CreateRecordForFile("database", dbName, 0, job.ID, outFile, 0, constant.StatusSuccess, outFile)
	return fmt.Sprintf("backup saved: %s", outFile), constant.StatusSuccess, outFile
}

func (s *CronjobService) execWebsiteBackup(job *model.Cronjob) (string, string, string) {
	if job.Website == "" {
		return "website name is empty", constant.StatusFailed, ""
	}
	backupService := NewIBackupService()
	if job.TargetAccountID > 0 {
		output, err := backupService.PerformBackupWithInfo("website", job.Website, "", "", job.TargetAccountID)
		if err != nil {
			_ = backupService.CreateRecordForFile("website", job.Website, job.TargetAccountID, job.ID, "", 0, constant.StatusFailed, err.Error())
			return fmt.Sprintf("backup failed: %v", err), constant.StatusFailed, ""
		}
		_ = backupService.CreateRecordForFile("website", job.Website, job.TargetAccountID, job.ID, output.Path, output.Size, constant.StatusSuccess, output.Path)
		return fmt.Sprintf("backup uploaded: %s", output.Path), constant.StatusSuccess, output.Path
	}
	msg, status := s.localBackupTar(job, "website", job.Website, "")
	file := extractBackupFile(msg)
	_ = backupService.CreateRecordForFile("website", job.Website, 0, job.ID, file, 0, status, msg)
	return msg, status, file
}

func (s *CronjobService) execDirectoryBackup(job *model.Cronjob) (string, string, string) {
	if job.SourceDir == "" {
		return "source directory is empty", constant.StatusFailed, ""
	}
	backupService := NewIBackupService()
	if job.TargetAccountID > 0 {
		output, err := backupService.PerformBackupWithInfo("directory", "", "", job.SourceDir, job.TargetAccountID)
		if err != nil {
			_ = backupService.CreateRecordForFile("directory", filepath.Base(job.SourceDir), job.TargetAccountID, job.ID, "", 0, constant.StatusFailed, err.Error())
			return fmt.Sprintf("backup failed: %v", err), constant.StatusFailed, ""
		}
		name := filepath.Base(job.SourceDir)
		_ = backupService.CreateRecordForFile("directory", name, job.TargetAccountID, job.ID, output.Path, output.Size, constant.StatusSuccess, output.Path)
		return fmt.Sprintf("backup uploaded: %s", output.Path), constant.StatusSuccess, output.Path
	}
	name := filepath.Base(job.SourceDir)
	msg, status := s.localBackupTar(job, "directory", job.SourceDir, job.SourceDir)
	file := extractBackupFile(msg)
	_ = backupService.CreateRecordForFile("directory", name, 0, job.ID, file, 0, status, msg)
	return msg, status, file
}

func (s *CronjobService) localBackupTar(job *model.Cronjob, backupType, name, sourceDir string) (string, string) {
	backupDir := fmt.Sprintf("%s/backup/%s", global.CONF.System.DataDir, backupType)
	timestamp := time.Now().Format("20060102150405")

	var tarDir string
	if backupType == "website" {
		websiteRepo := repo.NewIWebsiteRepo()
		website, err := websiteRepo.Get(repo.WithByPrimaryDomain(name))
		if err != nil || website.SiteDir == "" {
			tarDir = fmt.Sprintf("/var/www/%s", name)
		} else {
			tarDir = website.SiteDir
		}
	} else {
		tarDir = sourceDir
	}

	baseName := name
	if backupType == "directory" {
		baseName = filepath.Base(tarDir)
	}
	fileName := fmt.Sprintf("%s_%s_%s.tar.gz", backupType, baseName, timestamp)

	outFile, err := archiveUtil.CreateArchive(archiveUtil.ArchiveOptions{
		SourceDir:       tarDir,
		OutFile:         filepath.Join(backupDir, fileName),
		CompressFormat:  job.CompressFormat,
		EncryptPassword: job.EncryptPassword,
		ExclusionRules:  job.ExclusionRules,
	})
	if err != nil {
		return fmt.Sprintf("backup failed: %v", err), constant.StatusFailed
	}
	return fmt.Sprintf("backup saved: %s", outFile), constant.StatusSuccess
}

func isAllDatabases(name string) bool {
	name = strings.TrimSpace(strings.ToLower(name))
	return name == "all" || name == "*" || name == "__all__" || name == "全部"
}

func extractBackupFile(message string) string {
	const prefix = "backup saved: "
	if strings.HasPrefix(message, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(message, prefix))
	}
	return ""
}

func toCronjobInfo(j *model.Cronjob) *dto.CronjobInfo {
	return &dto.CronjobInfo{
		ID:              j.ID,
		CreatedAt:       j.CreatedAt,
		Name:            j.Name,
		Type:            j.Type,
		Spec:            j.Spec,
		Status:          j.Status,
		EntryID:         j.EntryID,
		Script:          j.Script,
		URL:             j.URL,
		Website:         j.Website,
		DBType:          j.DBType,
		DBName:          j.DBName,
		DBInstanceID:    j.DBInstanceID,
		SourceDir:       j.SourceDir,
		TargetAccountID: j.TargetAccountID,
		RetainCopies:    j.RetainCopies,
		ExclusionRules:  j.ExclusionRules,
		CompressFormat:  j.CompressFormat,
		EncryptPassword: j.EncryptPassword,
	}
}
