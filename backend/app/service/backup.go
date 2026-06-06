package service

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
	archiveUtil "xpanel/utils/backup"
	cs "xpanel/utils/cloud_storage"
	dbUtil "xpanel/utils/database"
)

type IBackupService interface {
	CreateAccount(req dto.BackupAccountCreate) error
	UpdateAccount(req dto.BackupAccountUpdate) error
	TestAccount(req dto.BackupAccountTest) error
	DeleteAccount(id uint) error
	ListAccounts() ([]dto.BackupAccountInfo, error)
	GetAccount(id uint) (*model.BackupAccount, error)

	Backup(req dto.BackupCreate) error
	SearchRecords(req dto.BackupRecordSearch) (int64, []dto.BackupRecordInfo, error)
	DeleteRecord(id uint) error
	PrepareRecordFile(id uint) (string, func(), error)
	CreateRecordForFile(backupType, name string, accountID uint, cronjobID uint, filePath string, size int64, status string, message string) error
	CleanSuccessfulRecords(cronjobID uint, retainCopies uint) error
	ListStorageObjects(req dto.BackupStorageReq) ([]dto.BackupStorageObject, error)
	ReadStorageObject(req dto.BackupStorageReq) (string, error)
	SaveStorageObject(req dto.BackupStorageReq) error
	DeleteStorageObject(req dto.BackupStorageReq) error
	UploadStorageObject(accountID uint, targetPath, srcFile string) error
	PrepareStorageObject(req dto.BackupStorageReq) (string, func(), error)

	PerformBackup(backupType, name, dbType, sourceDir string, accountID uint) (string, error)
	PerformBackupWithInfo(backupType, name, dbType, sourceDir string, accountID uint) (*BackupOutput, error)
	PerformDatabaseInstanceBackupWithInfo(instanceID uint, accountID uint) (*BackupOutput, error)
}

type BackupOutput struct {
	Path string
	Size int64
}

const (
	backupTempSpaceBuffer int64 = 256 * 1024 * 1024
	storageEditorMaxBytes int64 = 2 * 1024 * 1024
)

func NewIBackupService() IBackupService {
	return &BackupService{repo: repo.NewIBackupRepo(), dbRepo: repo.NewIDatabaseRepo()}
}

type BackupService struct {
	repo   repo.IBackupRepo
	dbRepo repo.IDatabaseRepo
}

func (s *BackupService) CreateAccount(req dto.BackupAccountCreate) error {
	return s.repo.CreateAccount(&model.BackupAccount{
		Name: req.Name, Type: req.Type, Bucket: req.Bucket,
		AccessKey: req.AccessKey, Credential: req.Credential,
		BackupPath: req.BackupPath, Vars: req.Vars,
	})
}

func (s *BackupService) UpdateAccount(req dto.BackupAccountUpdate) error {
	fields := map[string]interface{}{
		"name": req.Name, "bucket": req.Bucket,
		"backup_path": req.BackupPath, "vars": req.Vars,
	}
	if req.AccessKey != "" {
		fields["access_key"] = req.AccessKey
	}
	if req.Credential != "" {
		fields["credential"] = req.Credential
	}
	return s.repo.UpdateAccount(req.ID, fields)
}

func (s *BackupService) TestAccount(req dto.BackupAccountTest) error {
	credential := req.Credential
	if credential == "" && req.ID != 0 {
		account, err := s.repo.GetAccount(req.ID)
		if err != nil {
			return buserr.New(constant.ErrRecordNotFound)
		}
		credential = account.Credential
	}
	client, err := cs.NewClient(req.Type, req.Bucket, req.AccessKey, credential, req.BackupPath, req.Vars)
	if err != nil {
		return fmt.Errorf("create storage client failed: %v", err)
	}

	tmpDir := filepath.Join(os.TempDir(), "xpanel-backup-test")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return err
	}
	filePath := filepath.Join(tmpDir, "xpanel")
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(file)
	_, _ = writer.WriteString("XPanel backup account test file.\n")
	_, _ = writer.WriteString("XPanel 备份账户测试文件。\n")
	if err := writer.Flush(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	defer os.Remove(filePath)

	target := path.Join("test", fmt.Sprintf("xpanel-%d", time.Now().UnixNano()))
	if err := client.Upload(filePath, target); err != nil {
		return fmt.Errorf("upload test file failed: %v", err)
	}
	if err := client.Delete(target); err != nil {
		global.LOG.Warnf("delete backup test file failed: %v", err)
	}
	return nil
}

func (s *BackupService) DeleteAccount(id uint) error {
	return s.repo.DeleteAccount(id)
}

func (s *BackupService) ListAccounts() ([]dto.BackupAccountInfo, error) {
	accounts, err := s.repo.ListAccounts()
	if err != nil {
		return nil, err
	}
	var items []dto.BackupAccountInfo
	for _, a := range accounts {
		items = append(items, dto.BackupAccountInfo{
			ID: a.ID, CreatedAt: a.CreatedAt, Name: a.Name,
			Type: a.Type, Bucket: a.Bucket, BackupPath: a.BackupPath, Vars: a.Vars,
		})
	}
	return items, nil
}

func (s *BackupService) GetAccount(id uint) (*model.BackupAccount, error) {
	return s.repo.GetAccount(id)
}

func (s *BackupService) storageClient(accountID uint) (cs.CloudStorageClient, error) {
	account, err := s.GetAccount(accountID)
	if err != nil {
		return nil, err
	}
	client, err := cs.NewClient(account.Type, account.Bucket, account.AccessKey, account.Credential, account.BackupPath, account.Vars)
	if err != nil {
		return nil, fmt.Errorf("create storage client failed: %v", err)
	}
	return client, nil
}

func (s *BackupService) ListStorageObjects(req dto.BackupStorageReq) ([]dto.BackupStorageObject, error) {
	account, err := s.GetAccount(req.AccountID)
	if err != nil {
		return nil, err
	}
	client, err := cs.NewClient(account.Type, account.Bucket, account.AccessKey, account.Credential, account.BackupPath, account.Vars)
	if err != nil {
		return nil, fmt.Errorf("create storage client failed: %v", err)
	}
	keys, err := client.ListObjects(cleanObjectPath(req.Prefix))
	if err != nil {
		return nil, err
	}
	items := make([]dto.BackupStorageObject, 0, len(keys))
	for _, key := range keys {
		key = stripStorageBasePath(key, account.BackupPath)
		items = append(items, dto.BackupStorageObject{Name: path.Base(key), Path: key})
	}
	return items, nil
}

func (s *BackupService) ReadStorageObject(req dto.BackupStorageReq) (string, error) {
	tmp, release, err := s.PrepareStorageObject(req)
	if err != nil {
		return "", err
	}
	defer release()
	info, err := os.Stat(tmp)
	if err != nil {
		return "", err
	}
	if info.Size() > storageEditorMaxBytes {
		return "", fmt.Errorf("object is too large for online editing: max %s", formatSize(storageEditorMaxBytes))
	}
	data, err := os.ReadFile(tmp)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *BackupService) SaveStorageObject(req dto.BackupStorageReq) error {
	client, err := s.storageClient(req.AccountID)
	if err != nil {
		return err
	}
	target := cleanObjectPath(req.Path)
	if target == "" {
		return fmt.Errorf("object path is required")
	}
	if err := os.MkdirAll(backupTempDir(), 0750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(backupTempDir(), "xpanel-storage-save-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.WriteString(req.Content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return client.Upload(tmpPath, target)
}

func (s *BackupService) DeleteStorageObject(req dto.BackupStorageReq) error {
	client, err := s.storageClient(req.AccountID)
	if err != nil {
		return err
	}
	target := cleanObjectPath(req.Path)
	if target == "" {
		return fmt.Errorf("object path is required")
	}
	return client.Delete(target)
}

func (s *BackupService) UploadStorageObject(accountID uint, targetPath, srcFile string) error {
	client, err := s.storageClient(accountID)
	if err != nil {
		return err
	}
	target := cleanObjectPath(targetPath)
	if target == "" {
		return fmt.Errorf("object path is required")
	}
	return client.Upload(srcFile, target)
}

func (s *BackupService) PrepareStorageObject(req dto.BackupStorageReq) (string, func(), error) {
	client, err := s.storageClient(req.AccountID)
	if err != nil {
		return "", nil, err
	}
	source := cleanObjectPath(req.Path)
	if source == "" {
		return "", nil, fmt.Errorf("object path is required")
	}
	if err := os.MkdirAll(backupTempDir(), 0750); err != nil {
		return "", nil, err
	}
	tmp, err := os.CreateTemp(backupTempDir(), "xpanel-storage-*"+filepath.Ext(source))
	if err != nil {
		return "", nil, err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	if err := client.Download(source, tmpPath); err != nil {
		os.Remove(tmpPath)
		return "", nil, err
	}
	return tmpPath, func() { _ = os.Remove(tmpPath) }, nil
}

func cleanObjectPath(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(path.Clean("/"+value), "/")
	if value == "." {
		return ""
	}
	return value
}

func stripStorageBasePath(key, basePath string) string {
	key = cleanObjectPath(key)
	base := cleanObjectPath(basePath)
	if base == "" {
		return key
	}
	return strings.TrimPrefix(strings.TrimPrefix(key, base), "/")
}

func (s *BackupService) Backup(req dto.BackupCreate) error {
	go func() {
		output, err := s.PerformBackupWithInfo(req.Type, req.Name, req.DBType, req.SourceDir, req.AccountID)
		record := &model.BackupRecord{
			Type: req.Type, Name: req.Name, AccountID: req.AccountID,
		}
		if err != nil {
			record.Status = constant.StatusFailed
			record.Message = err.Error()
		} else {
			record.Status = constant.StatusSuccess
			record.FileName = filepath.Base(output.Path)
			record.FileDir = filepath.Dir(output.Path)
			record.Message = output.Path
			record.Size = output.Size
		}
		if err := s.repo.CreateRecord(record); err != nil {
			global.LOG.Errorf("save backup record failed: %v", err)
		}
	}()
	return nil
}

func (s *BackupService) PerformBackup(backupType, name, dbType, sourceDir string, accountID uint) (string, error) {
	output, err := s.PerformBackupWithInfo(backupType, name, dbType, sourceDir, accountID)
	if err != nil {
		return "", err
	}
	return output.Path, nil
}

func (s *BackupService) PerformBackupWithInfo(backupType, name, dbType, sourceDir string, accountID uint) (*BackupOutput, error) {
	account, err := s.repo.GetAccount(accountID)
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}

	client, err := cs.NewClient(account.Type, account.Bucket, account.AccessKey, account.Credential, account.BackupPath, account.Vars)
	if err != nil {
		return nil, fmt.Errorf("create storage client failed: %v", err)
	}

	timestamp := time.Now().Format("20060102150405")
	tmpDir := backupTempDir()
	os.MkdirAll(tmpDir, 0755)

	var localFile string
	var targetPath string

	switch backupType {
	case "website":
		localFile, targetPath, err = s.backupWebsite(name, tmpDir, timestamp)
	case "database":
		localFile, targetPath, err = s.backupDatabase(name, dbType, tmpDir, timestamp)
	case "directory":
		localFile, targetPath, err = s.backupDirectory(sourceDir, name, tmpDir, timestamp)
	default:
		return nil, fmt.Errorf("unsupported backup type: %s", backupType)
	}
	if err != nil {
		return nil, err
	}
	defer os.Remove(localFile)
	size := fileSize(localFile)

	if err := client.Upload(localFile, targetPath); err != nil {
		return nil, fmt.Errorf("upload failed: %v", err)
	}

	return &BackupOutput{Path: targetPath, Size: size}, nil
}

func (s *BackupService) PerformDatabaseInstanceBackupWithInfo(instanceID uint, accountID uint) (*BackupOutput, error) {
	account, err := s.repo.GetAccount(accountID)
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}

	client, err := cs.NewClient(account.Type, account.Bucket, account.AccessKey, account.Credential, account.BackupPath, account.Vars)
	if err != nil {
		return nil, fmt.Errorf("create storage client failed: %v", err)
	}

	timestamp := time.Now().Format("20060102150405")
	tmpDir := backupTempDir()
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, err
	}

	localFile, targetPath, err := s.backupDatabaseInstance(instanceID, tmpDir, timestamp)
	if err != nil {
		return nil, err
	}
	defer os.Remove(localFile)
	size := fileSize(localFile)

	if err := client.Upload(localFile, targetPath); err != nil {
		return nil, fmt.Errorf("upload failed: %v", err)
	}

	return &BackupOutput{Path: targetPath, Size: size}, nil
}

func (s *BackupService) backupWebsite(name, tmpDir, timestamp string) (string, string, error) {
	websiteRepo := repo.NewIWebsiteRepo()
	website, err := websiteRepo.Get(repo.WithByPrimaryDomain(name))
	var siteDir string
	if err == nil && website.SiteDir != "" {
		siteDir = website.SiteDir
	} else {
		siteDir = filepath.Join("/var/www", name)
	}

	fileName := fmt.Sprintf("website_%s_%s.tar.gz", name, timestamp)
	if err := ensureBackupTempSpace(siteDir, tmpDir); err != nil {
		return "", "", err
	}
	outFile, err := archiveUtil.CreateArchive(archiveUtil.ArchiveOptions{
		SourceDir: siteDir,
		OutFile:   filepath.Join(tmpDir, fileName),
	})
	if err != nil {
		return "", "", err
	}
	return outFile, filepath.Join("website", name, filepath.Base(outFile)), nil
}

func backupTempDir() string {
	if global.CONF.System.DataDir != "" {
		return filepath.Join(global.CONF.System.DataDir, "tmp", "xpanel-backup")
	}
	return filepath.Join(os.TempDir(), "xpanel-backup")
}

func ensureBackupTempSpace(sourceDir, tmpDir string) error {
	sourceSize, err := estimateDirectorySize(sourceDir)
	if err != nil {
		return fmt.Errorf("estimate source size failed: %v", err)
	}
	available, err := availableBytes(tmpDir)
	if err != nil {
		return fmt.Errorf("check backup temp space failed: %v", err)
	}
	required := sourceSize + backupTempSpaceBuffer
	if available < required {
		return fmt.Errorf("backup temp space not enough: need at least %s, available %s, temp dir %s",
			formatSize(required), formatSize(available), tmpDir)
	}
	return nil
}

func estimateDirectorySize(dir string) (int64, error) {
	var total int64
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			total += info.Size()
		}
		return nil
	})
	return total, err
}

func availableBytes(dir string) (int64, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, err
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return 0, err
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func CleanBackupTempDir(maxAge time.Duration) {
	tmpDir := backupTempDir()
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if entry.IsDir() || !isBackupTempArchive(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(tmpDir, entry.Name())); err != nil && global.LOG != nil {
			global.LOG.Warnf("clean backup temp file failed: %v", err)
		}
	}
}

func isBackupTempArchive(name string) bool {
	return strings.HasSuffix(name, ".tar.gz") ||
		strings.HasSuffix(name, ".tar.zst") ||
		strings.HasSuffix(name, ".tar.xz") ||
		strings.HasSuffix(name, ".tar.gz.enc") ||
		strings.HasSuffix(name, ".tar.zst.enc") ||
		strings.HasSuffix(name, ".tar.xz.enc")
}

func (s *BackupService) backupDatabase(name, dbType, tmpDir, timestamp string) (string, string, error) {
	servers, _ := s.dbRepo.ListServers(repo.WithServerType(dbType))
	if len(servers) == 0 {
		return "", "", fmt.Errorf("no %s server found", dbType)
	}

	// Find which server actually has this database instance
	var targetServer *model.DatabaseServer
	for i := range servers {
		instances, _ := s.dbRepo.ListInstancesByServerID(servers[i].ID)
		for _, inst := range instances {
			if inst.Name == name {
				targetServer = &servers[i]
				break
			}
		}
		if targetServer != nil {
			break
		}
	}
	if targetServer == nil {
		targetServer = &servers[0]
	}

	fileName := fmt.Sprintf("db_%s_%s_%s.sql", name, dbType, timestamp)
	if dbType == "postgresql" {
		fileName = fmt.Sprintf("db_%s_%s_%s.dump", name, dbType, timestamp)
	}
	localFile := filepath.Join(tmpDir, fileName)

	switch dbType {
	case "mysql":
		client, err := dbUtil.NewMysqlClient(targetServer.Address, targetServer.Port, targetServer.Username, targetServer.Password)
		if err != nil {
			return "", "", err
		}
		defer client.Close()
		if err := client.Backup(name, localFile); err != nil {
			return "", "", err
		}
	case "postgresql":
		client, err := dbUtil.NewPostgresClient(targetServer.Address, targetServer.Port, targetServer.Username, targetServer.Password)
		if err != nil {
			return "", "", err
		}
		defer client.Close()
		if err := client.Backup(name, localFile); err != nil {
			return "", "", err
		}
	}
	return localFile, filepath.Join("database", name, fileName), nil
}

func (s *BackupService) backupDatabaseInstance(instanceID uint, tmpDir, timestamp string) (string, string, error) {
	instance, err := s.dbRepo.GetInstance(instanceID)
	if err != nil {
		return "", "", buserr.New(constant.ErrRecordNotFound)
	}
	server, err := s.dbRepo.GetServer(instance.ServerID)
	if err != nil {
		return "", "", buserr.New(constant.ErrRecordNotFound)
	}

	fileName := fmt.Sprintf("db_%s_%s_%s.sql", instance.Name, server.Type, timestamp)
	if server.Type == "postgresql" {
		fileName = fmt.Sprintf("db_%s_%s_%s.dump", instance.Name, server.Type, timestamp)
	}
	localFile := filepath.Join(tmpDir, fileName)

	switch server.Type {
	case "mysql":
		client, err := dbUtil.NewMysqlClient(server.Address, server.Port, server.Username, server.Password)
		if err != nil {
			return "", "", err
		}
		defer client.Close()
		if err := client.Backup(instance.Name, localFile); err != nil {
			return "", "", err
		}
	case "postgresql":
		client, err := dbUtil.NewPostgresClient(server.Address, server.Port, server.Username, server.Password)
		if err != nil {
			return "", "", err
		}
		defer client.Close()
		if err := client.Backup(instance.Name, localFile); err != nil {
			return "", "", err
		}
	default:
		return "", "", fmt.Errorf("unsupported database type: %s", server.Type)
	}
	return localFile, filepath.Join("database", instance.Name, fileName), nil
}

func (s *BackupService) backupDirectory(sourceDir, name, tmpDir, timestamp string) (string, string, error) {
	if name == "" {
		name = filepath.Base(sourceDir)
	}
	fileName := fmt.Sprintf("dir_%s_%s.tar.gz", name, timestamp)
	if err := ensureBackupTempSpace(sourceDir, tmpDir); err != nil {
		return "", "", err
	}
	outFile, err := archiveUtil.CreateArchive(archiveUtil.ArchiveOptions{
		SourceDir: sourceDir,
		OutFile:   filepath.Join(tmpDir, fileName),
	})
	if err != nil {
		return "", "", err
	}
	return outFile, filepath.Join("directory", name, filepath.Base(outFile)), nil
}

func (s *BackupService) SearchRecords(req dto.BackupRecordSearch) (int64, []dto.BackupRecordInfo, error) {
	opts := []repo.DBOption{
		repo.WithBackupType(req.Type),
		repo.WithAccountID(req.AccountID),
		repo.WithBackupName(req.Name),
		repo.WithBackupStatus(req.Status),
	}
	total, records, err := s.repo.PageRecord(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}
	var items []dto.BackupRecordInfo
	for _, r := range records {
		items = append(items, dto.BackupRecordInfo{
			ID: r.ID, CreatedAt: r.CreatedAt, Type: r.Type,
			Name: r.Name, AccountID: r.AccountID, CronjobID: r.CronjobID, FileName: r.FileName,
			FileDir: r.FileDir, Size: r.Size, Status: r.Status, Message: r.Message,
		})
	}
	return total, items, nil
}

func (s *BackupService) DeleteRecord(id uint) error {
	record, err := s.repo.GetRecord(id)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if record.Status == constant.StatusSuccess {
		if err := s.deleteRecordFile(record); err != nil {
			return fmt.Errorf("delete backup file failed: %v", err)
		}
	}
	return s.repo.DeleteRecord(id)
}

func (s *BackupService) CreateRecordForFile(backupType, name string, accountID uint, cronjobID uint, filePath string, size int64, status string, message string) error {
	record := &model.BackupRecord{
		CronjobID: cronjobID,
		Type:      backupType,
		Name:      name,
		AccountID: accountID,
		Status:    status,
		Message:   message,
	}
	if filePath != "" {
		record.FileName = filepath.Base(filePath)
		record.FileDir = filepath.Dir(filePath)
		record.Size = size
		if record.Size == 0 {
			record.Size = s.recordFileSize(accountID, record.FileDir, record.FileName)
		}
	}
	return s.repo.CreateRecord(record)
}

func (s *BackupService) CleanSuccessfulRecords(cronjobID uint, retainCopies uint) error {
	if cronjobID == 0 || retainCopies == 0 {
		return nil
	}
	records, err := s.repo.ListRecords(
		repo.WithBackupCronjobID(cronjobID),
		repo.WithBackupStatus(constant.StatusSuccess),
	)
	if err != nil {
		return err
	}
	retained := make(map[string]uint)
	for _, record := range records {
		key := fmt.Sprintf("%s/%s/%d", record.Type, record.Name, record.AccountID)
		retained[key]++
		if retained[key] <= retainCopies {
			continue
		}
		if err := s.deleteRecordFile(&record); err != nil {
			global.LOG.Warnf("delete retained backup file failed: %v", err)
		}
		_ = s.repo.DeleteRecord(record.ID)
	}
	return nil
}

func (s *BackupService) PrepareRecordFile(id uint) (string, func(), error) {
	record, err := s.repo.GetRecord(id)
	if err != nil {
		return "", nil, buserr.New(constant.ErrRecordNotFound)
	}
	if record.Status != constant.StatusSuccess {
		return "", nil, fmt.Errorf("backup record is not successful")
	}
	relPath := recordPath(record)
	if relPath == "" {
		return "", nil, fmt.Errorf("backup record file is empty")
	}
	if record.AccountID == 0 {
		return relPath, func() {}, nil
	}
	account, err := s.repo.GetAccount(record.AccountID)
	if err != nil {
		return "", nil, buserr.New(constant.ErrRecordNotFound)
	}
	client, err := cs.NewClient(account.Type, account.Bucket, account.AccessKey, account.Credential, account.BackupPath, account.Vars)
	if err != nil {
		return "", nil, err
	}
	tmp, err := os.CreateTemp("", "xpanel-backup-record-*"+filepath.Ext(record.FileName))
	if err != nil {
		return "", nil, err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	if err := client.Download(relPath, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", nil, err
	}
	return tmpPath, func() { _ = os.Remove(tmpPath) }, nil
}

func (s *BackupService) deleteRecordFile(record *model.BackupRecord) error {
	filePath := recordPath(record)
	if filePath == "" {
		return nil
	}
	if record.AccountID == 0 {
		err := os.Remove(filePath)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	account, err := s.repo.GetAccount(record.AccountID)
	if err != nil {
		return err
	}
	client, err := cs.NewClient(account.Type, account.Bucket, account.AccessKey, account.Credential, account.BackupPath, account.Vars)
	if err != nil {
		return err
	}
	err = client.Delete(filePath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *BackupService) recordFileSize(accountID uint, fileDir, fileName string) int64 {
	if accountID != 0 {
		return 0
	}
	return fileSize(filepath.Join(fileDir, fileName))
}

func fileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return info.Size()
}

func recordPath(record *model.BackupRecord) string {
	if record == nil || record.FileName == "" {
		return ""
	}
	if record.FileDir == "" || record.FileDir == "." {
		return record.FileName
	}
	if filepath.IsAbs(record.FileDir) {
		return filepath.Join(record.FileDir, record.FileName)
	}
	return strings.TrimPrefix(filepath.ToSlash(filepath.Join(record.FileDir, record.FileName)), "/")
}
