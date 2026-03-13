package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
	cs "xpanel/utils/cloud_storage"
	dbUtil "xpanel/utils/database"
)

type IBackupService interface {
	CreateAccount(req dto.BackupAccountCreate) error
	UpdateAccount(req dto.BackupAccountUpdate) error
	DeleteAccount(id uint) error
	ListAccounts() ([]dto.BackupAccountInfo, error)
	GetAccount(id uint) (*model.BackupAccount, error)

	Backup(req dto.BackupCreate) error
	SearchRecords(req dto.BackupRecordSearch) (int64, []dto.BackupRecordInfo, error)
	DeleteRecord(id uint) error

	PerformBackup(backupType, name, dbType, sourceDir string, accountID uint) (string, error)
}

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

func (s *BackupService) Backup(req dto.BackupCreate) error {
	go func() {
		msg, err := s.PerformBackup(req.Type, req.Name, req.DBType, req.SourceDir, req.AccountID)
		status := constant.StatusSuccess
		if err != nil {
			status = constant.StatusFailed
			msg = err.Error()
		}
		record := &model.BackupRecord{
			Type: req.Type, Name: req.Name, AccountID: req.AccountID,
			FileName: filepath.Base(msg), FileDir: filepath.Dir(msg),
			Status: status, Message: msg,
		}
		if err := s.repo.CreateRecord(record); err != nil {
			global.LOG.Errorf("save backup record failed: %v", err)
		}
	}()
	return nil
}

func (s *BackupService) PerformBackup(backupType, name, dbType, sourceDir string, accountID uint) (string, error) {
	account, err := s.repo.GetAccount(accountID)
	if err != nil {
		return "", buserr.New(constant.ErrRecordNotFound)
	}

	client, err := cs.NewClient(account.Type, account.Bucket, account.AccessKey, account.Credential, account.BackupPath, account.Vars)
	if err != nil {
		return "", fmt.Errorf("create storage client failed: %v", err)
	}

	timestamp := time.Now().Format("20060102150405")
	tmpDir := filepath.Join(os.TempDir(), "xpanel-backup")
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
		return "", fmt.Errorf("unsupported backup type: %s", backupType)
	}
	if err != nil {
		return "", err
	}
	defer os.Remove(localFile)

	if err := client.Upload(localFile, targetPath); err != nil {
		return "", fmt.Errorf("upload failed: %v", err)
	}

	return targetPath, nil
}

func (s *BackupService) backupWebsite(name, tmpDir, timestamp string) (string, string, error) {
	siteDir := filepath.Join("/var/www", name)
	if _, err := os.Stat(siteDir); err != nil {
		siteDir = filepath.Join(global.CONF.Nginx.GetConfDir(), "conf.d")
	}

	fileName := fmt.Sprintf("website_%s_%s.tar.gz", name, timestamp)
	localFile := filepath.Join(tmpDir, fileName)

	cmd := exec.Command("tar", "-czf", localFile, "-C", filepath.Dir(siteDir), filepath.Base(siteDir))
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("tar failed: %s", string(output))
	}
	return localFile, filepath.Join("website", name, fileName), nil
}

func (s *BackupService) backupDatabase(name, dbType, tmpDir, timestamp string) (string, string, error) {
	servers, _ := s.dbRepo.ListServers(repo.WithServerType(dbType))
	if len(servers) == 0 {
		return "", "", fmt.Errorf("no %s server found", dbType)
	}
	server := servers[0]
	fileName := fmt.Sprintf("db_%s_%s_%s.sql", name, dbType, timestamp)
	if dbType == "postgresql" {
		fileName = fmt.Sprintf("db_%s_%s_%s.dump", name, dbType, timestamp)
	}
	localFile := filepath.Join(tmpDir, fileName)

	switch dbType {
	case "mysql":
		client, err := dbUtil.NewMysqlClient(server.Address, server.Port, server.Username, server.Password)
		if err != nil {
			return "", "", err
		}
		defer client.Close()
		if err := client.Backup(name, localFile); err != nil {
			return "", "", err
		}
	case "postgresql":
		client, err := dbUtil.NewPostgresClient(server.Address, server.Port, server.Username, server.Password)
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

func (s *BackupService) backupDirectory(sourceDir, name, tmpDir, timestamp string) (string, string, error) {
	if name == "" {
		name = filepath.Base(sourceDir)
	}
	fileName := fmt.Sprintf("dir_%s_%s.tar.gz", name, timestamp)
	localFile := filepath.Join(tmpDir, fileName)

	cmd := exec.Command("tar", "-czf", localFile, "-C", filepath.Dir(sourceDir), filepath.Base(sourceDir))
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("tar failed: %s", string(output))
	}
	return localFile, filepath.Join("directory", name, fileName), nil
}

func (s *BackupService) SearchRecords(req dto.BackupRecordSearch) (int64, []dto.BackupRecordInfo, error) {
	opts := []repo.DBOption{
		repo.WithBackupType(req.Type),
		repo.WithAccountID(req.AccountID),
	}
	total, records, err := s.repo.PageRecord(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}
	var items []dto.BackupRecordInfo
	for _, r := range records {
		items = append(items, dto.BackupRecordInfo{
			ID: r.ID, CreatedAt: r.CreatedAt, Type: r.Type,
			Name: r.Name, AccountID: r.AccountID, FileName: r.FileName,
			FileDir: r.FileDir, Size: r.Size, Status: r.Status, Message: r.Message,
		})
	}
	return total, items, nil
}

func (s *BackupService) DeleteRecord(id uint) error {
	return s.repo.DeleteRecord(id)
}
