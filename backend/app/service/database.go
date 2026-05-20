package service

import (
	"fmt"
	"os"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
	dbUtil "xpanel/utils/database"
)

type IDatabaseService interface {
	CreateServer(req dto.DatabaseServerCreate) error
	UpdateServer(req dto.DatabaseServerUpdate) error
	DeleteServer(id uint) error
	SearchServer(req dto.DatabaseServerSearch) (int64, []dto.DatabaseServerInfo, error)
	GetServer(id uint) (*model.DatabaseServer, error)
	TestConnection(id uint) error

	CreateInstance(req dto.DatabaseInstanceCreate) error
	DeleteInstance(id uint) error
	SearchInstance(req dto.DatabaseInstanceSearch) (int64, []dto.DatabaseInstanceInfo, error)
	SyncInstances(serverID uint) error
	ChangeInstancePassword(req dto.DatabaseInstanceChangePassword) error
	BackupInstance(id uint) (string, error)
	BackupInstanceAsync(id uint) (*FileTaskStatus, error)
	RestoreInstance(req dto.DatabaseInstanceRestore) error
	RestoreInstanceAsync(req dto.DatabaseInstanceRestore) (*FileTaskStatus, error)
	ChangeInstancePrivileges(req dto.DatabaseInstanceChangePrivileges) error
}

func NewIDatabaseService() IDatabaseService {
	return &DatabaseService{repo: repo.NewIDatabaseRepo()}
}

type DatabaseService struct {
	repo repo.IDatabaseRepo
}

func (s *DatabaseService) CreateServer(req dto.DatabaseServerCreate) error {
	server := &model.DatabaseServer{
		Name:     req.Name,
		Type:     req.Type,
		From:     req.From,
		Address:  req.Address,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
	}
	if server.Address == "" {
		server.Address = "127.0.0.1"
	}
	return s.repo.CreateServer(server)
}

func (s *DatabaseService) UpdateServer(req dto.DatabaseServerUpdate) error {
	fields := map[string]interface{}{
		"name":     req.Name,
		"address":  req.Address,
		"port":     req.Port,
		"username": req.Username,
	}
	if req.Password != "" {
		fields["password"] = req.Password
	}
	return s.repo.UpdateServer(req.ID, fields)
}

func (s *DatabaseService) DeleteServer(id uint) error {
	_ = s.repo.DeleteInstanceByServerID(id)
	return s.repo.DeleteServer(id)
}

func (s *DatabaseService) SearchServer(req dto.DatabaseServerSearch) (int64, []dto.DatabaseServerInfo, error) {
	opts := []repo.DBOption{repo.WithServerType(req.Type)}
	if req.Info != "" {
		opts = append(opts, repo.WithLikeName(req.Info))
	}
	total, servers, err := s.repo.PageServer(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}
	var items []dto.DatabaseServerInfo
	for _, sv := range servers {
		items = append(items, dto.DatabaseServerInfo{
			ID: sv.ID, CreatedAt: sv.CreatedAt, Name: sv.Name,
			Type: sv.Type, From: sv.From, Address: sv.Address,
			Port: sv.Port, Username: sv.Username,
		})
	}
	return total, items, nil
}

func (s *DatabaseService) GetServer(id uint) (*model.DatabaseServer, error) {
	return s.repo.GetServer(id)
}

func (s *DatabaseService) TestConnection(id uint) error {
	server, err := s.repo.GetServer(id)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	return testDBConnection(server)
}

func (s *DatabaseService) CreateInstance(req dto.DatabaseInstanceCreate) error {
	server, err := s.repo.GetServer(req.ServerID)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}

	switch server.Type {
	case "mysql":
		client, err := dbUtil.NewMysqlClient(server.Address, server.Port, server.Username, server.Password)
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		defer client.Close()
		username := req.Username
		if username == "" {
			username = req.Name
		}
		permission := req.Permission
		if permission == "" {
			permission = "%"
		}
		if req.Password == "" {
			return buserr.WithDetail(constant.ErrInternalServer, "password is required", fmt.Errorf("password is required"))
		}
		if err := client.CreateDatabaseWithUser(req.Name, req.Charset, username, req.Password, permission); err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		req.Username = username
		req.Permission = permission
	case "postgresql":
		client, err := dbUtil.NewPostgresClient(server.Address, server.Port, server.Username, server.Password)
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		defer client.Close()
		username := req.Username
		if username == "" {
			username = req.Owner
		}
		if username == "" {
			username = req.Name
		}
		if req.Password == "" {
			return buserr.WithDetail(constant.ErrInternalServer, "password is required", fmt.Errorf("password is required"))
		}
		if err := client.CreateDatabaseWithUser(req.Name, username, req.Password, req.SuperUser); err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		req.Owner = username
		req.Username = username
	}

	instance := &model.DatabaseInstance{
		ServerID:   req.ServerID,
		Name:       req.Name,
		Charset:    req.Charset,
		Owner:      req.Owner,
		Username:   req.Username,
		Password:   req.Password,
		Permission: req.Permission,
		SuperUser:  req.SuperUser,
	}
	return s.repo.CreateInstance(instance)
}

func (s *DatabaseService) DeleteInstance(id uint) error {
	instance, err := s.repo.GetInstance(id)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	server, err := s.repo.GetServer(instance.ServerID)
	if err == nil {
		switch server.Type {
		case "mysql":
			client, err := dbUtil.NewMysqlClient(server.Address, server.Port, server.Username, server.Password)
			if err == nil {
				defer client.Close()
				_ = client.DeleteDatabase(instance.Name)
				username := instance.Username
				if username == "" {
					username = instance.Name
				}
				_ = client.DeleteUser(username, instance.Permission)
			}
		case "postgresql":
			client, err := dbUtil.NewPostgresClient(server.Address, server.Port, server.Username, server.Password)
			if err == nil {
				defer client.Close()
				_ = client.DeleteDatabase(instance.Name)
				if instance.Username != "" && instance.Password != "" {
					_ = client.DeleteUser(instance.Username)
				}
			}
		}
	}
	return s.repo.DeleteInstance(id)
}

func (s *DatabaseService) SearchInstance(req dto.DatabaseInstanceSearch) (int64, []dto.DatabaseInstanceInfo, error) {
	opts := []repo.DBOption{repo.WithServerID(req.ServerID)}
	if req.Info != "" {
		opts = append(opts, repo.WithLikeName(req.Info))
	}
	total, instances, err := s.repo.PageInstance(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}
	var items []dto.DatabaseInstanceInfo
	for _, inst := range instances {
		items = append(items, dto.DatabaseInstanceInfo{
			ID: inst.ID, CreatedAt: inst.CreatedAt, ServerID: inst.ServerID,
			Name: inst.Name, Charset: inst.Charset, Owner: inst.Owner,
			Username: inst.Username, Permission: inst.Permission,
			SuperUser: inst.SuperUser,
		})
	}
	return total, items, nil
}

func (s *DatabaseService) SyncInstances(serverID uint) error {
	server, err := s.repo.GetServer(serverID)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}

	var remoteDBs []dbUtil.DBInfo
	switch server.Type {
	case "mysql":
		client, err := dbUtil.NewMysqlClient(server.Address, server.Port, server.Username, server.Password)
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		defer client.Close()
		remoteDBs, err = client.ListDatabasesWithInfo()
		if err != nil {
			return err
		}
	case "postgresql":
		client, err := dbUtil.NewPostgresClient(server.Address, server.Port, server.Username, server.Password)
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		defer client.Close()
		remoteDBs, err = client.ListDatabasesWithInfo()
		if err != nil {
			return err
		}
	}

	existingInstances, err := s.repo.ListInstancesByServerID(serverID)
	if err != nil {
		return err
	}
	existingMap := make(map[string]model.DatabaseInstance, len(existingInstances))
	for _, inst := range existingInstances {
		if prev, dup := existingMap[inst.Name]; dup {
			_ = s.repo.DeleteInstance(prev.ID)
		}
		existingMap[inst.Name] = inst
	}

	remoteSet := make(map[string]struct{}, len(remoteDBs))
	for _, info := range remoteDBs {
		remoteSet[info.Name] = struct{}{}
		if existing, found := existingMap[info.Name]; found {
			fields := map[string]interface{}{
				"charset":    info.Charset,
				"owner":      info.Owner,
				"permission": info.Permission,
			}
			if existing.Username == "" {
				if info.Username != "" {
					fields["username"] = info.Username
				} else {
					fields["username"] = info.Owner
				}
			}
			_ = s.repo.UpdateInstance(existing.ID, fields)
		} else {
			username := info.Username
			if username == "" {
				username = info.Owner
			}
			_ = s.repo.CreateInstance(&model.DatabaseInstance{
				ServerID:   serverID,
				Name:       info.Name,
				Charset:    info.Charset,
				Owner:      info.Owner,
				Username:   username,
				Permission: info.Permission,
			})
		}
	}

	for name, inst := range existingMap {
		if _, found := remoteSet[name]; !found {
			_ = s.repo.DeleteInstance(inst.ID)
		}
	}

	return nil
}

func (s *DatabaseService) ChangeInstancePassword(req dto.DatabaseInstanceChangePassword) error {
	instance, err := s.repo.GetInstance(req.ID)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	server, err := s.repo.GetServer(instance.ServerID)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	switch server.Type {
	case "mysql":
		client, err := dbUtil.NewMysqlClient(server.Address, server.Port, server.Username, server.Password)
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		defer client.Close()
		userName := instance.Username
		if userName == "" {
			userName = instance.Name
		}
		if err := client.ChangePassword(userName, req.Password, instance.Permission); err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		_ = s.repo.UpdateInstance(instance.ID, map[string]interface{}{"password": req.Password})
	case "postgresql":
		client, err := dbUtil.NewPostgresClient(server.Address, server.Port, server.Username, server.Password)
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		defer client.Close()
		userName := instance.Owner
		if instance.Username != "" {
			userName = instance.Username
		}
		if userName == "" {
			userName = instance.Name
		}
		if err := client.ChangePassword(userName, req.Password); err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		_ = s.repo.UpdateInstance(instance.ID, map[string]interface{}{"password": req.Password})
	}
	return nil
}

func (s *DatabaseService) ChangeInstancePrivileges(req dto.DatabaseInstanceChangePrivileges) error {
	instance, err := s.repo.GetInstance(req.ID)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	server, err := s.repo.GetServer(instance.ServerID)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if server.Type != "postgresql" {
		return buserr.WithDetail(constant.ErrInternalServer, "privileges are only supported for PostgreSQL", fmt.Errorf("privileges are only supported for PostgreSQL"))
	}
	client, err := dbUtil.NewPostgresClient(server.Address, server.Port, server.Username, server.Password)
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	defer client.Close()
	userName := instance.Username
	if userName == "" {
		userName = instance.Owner
	}
	if userName == "" {
		return buserr.WithDetail(constant.ErrInternalServer, "database user is empty", fmt.Errorf("database user is empty"))
	}
	if err := client.ChangePrivileges(userName, req.SuperUser); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	return s.repo.UpdateInstance(instance.ID, map[string]interface{}{"super_user": req.SuperUser})
}

func (s *DatabaseService) BackupInstance(id uint) (string, error) {
	instance, err := s.repo.GetInstance(id)
	if err != nil {
		return "", buserr.New(constant.ErrRecordNotFound)
	}
	server, err := s.repo.GetServer(instance.ServerID)
	if err != nil {
		return "", buserr.New(constant.ErrRecordNotFound)
	}

	backupDir := global.CONF.System.DataDir + "/backup/database"
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return "", buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	timestamp := time.Now().Format("20060102150405")
	var outFile string

	switch server.Type {
	case "mysql":
		outFile = fmt.Sprintf("%s/%s_%s.sql", backupDir, instance.Name, timestamp)
		client, err := dbUtil.NewMysqlClient(server.Address, server.Port, server.Username, server.Password)
		if err != nil {
			return "", buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		defer client.Close()
		if err := client.Backup(instance.Name, outFile); err != nil {
			return "", buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
	case "postgresql":
		outFile = fmt.Sprintf("%s/%s_%s.dump", backupDir, instance.Name, timestamp)
		client, err := dbUtil.NewPostgresClient(server.Address, server.Port, server.Username, server.Password)
		if err != nil {
			return "", buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		defer client.Close()
		if err := client.Backup(instance.Name, outFile); err != nil {
			return "", buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
	}
	return outFile, nil
}

func (s *DatabaseService) BackupInstanceAsync(id uint) (*FileTaskStatus, error) {
	instance, err := s.repo.GetInstance(id)
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}
	taskName := fmt.Sprintf("备份数据库 %s", instance.Name)
	var outFile string
	task := StartFileTaskWithNotification("database_backup", taskName, FileTaskNotification{
		Source:       "database",
		TargetURL:    "/database",
		SuccessTitle: fmt.Sprintf("数据库「%s」备份完成", instance.Name),
		SuccessContentFunc: func() string {
			if outFile == "" {
				return "数据库备份任务已完成"
			}
			return fmt.Sprintf("备份文件已保存到：%s", outFile)
		},
		FailedTitle: fmt.Sprintf("数据库「%s」备份失败", instance.Name),
	}, func() error {
		file, err := s.BackupInstance(id)
		if err != nil {
			_ = NewIBackupService().CreateRecordForFile("database", instance.Name, 0, 0, "", 0, constant.StatusFailed, err.Error())
			return err
		}
		outFile = file
		_ = NewIBackupService().CreateRecordForFile("database", instance.Name, 0, 0, file, 0, constant.StatusSuccess, file)
		return nil
	})
	return task, nil
}

func (s *DatabaseService) RestoreInstance(req dto.DatabaseInstanceRestore) error {
	inFile := req.File
	cleanup := func() {}
	if req.BackupRecordID > 0 {
		prepared, release, err := NewIBackupService().PrepareRecordFile(req.BackupRecordID)
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		inFile = prepared
		cleanup = release
	}
	defer cleanup()
	if inFile == "" {
		return buserr.WithDetail(constant.ErrInternalServer, "backup file is required", fmt.Errorf("backup file is required"))
	}

	instance, err := s.repo.GetInstance(req.ID)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	server, err := s.repo.GetServer(instance.ServerID)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}

	switch server.Type {
	case "mysql":
		client, err := dbUtil.NewMysqlClient(server.Address, server.Port, server.Username, server.Password)
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		defer client.Close()
		if err := client.Restore(instance.Name, inFile); err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
	case "postgresql":
		client, err := dbUtil.NewPostgresClient(server.Address, server.Port, server.Username, server.Password)
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		defer client.Close()
		if err := client.Restore(instance.Name, inFile); err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
	}
	return nil
}

func (s *DatabaseService) RestoreInstanceAsync(req dto.DatabaseInstanceRestore) (*FileTaskStatus, error) {
	instance, err := s.repo.GetInstance(req.ID)
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}
	taskName := fmt.Sprintf("恢复数据库 %s", instance.Name)
	task := StartFileTaskWithNotification("database_restore", taskName, FileTaskNotification{
		Source:         "database",
		TargetURL:      "/database",
		SuccessTitle:   fmt.Sprintf("数据库「%s」恢复完成", instance.Name),
		SuccessContent: "数据库恢复任务已完成",
		FailedTitle:    fmt.Sprintf("数据库「%s」恢复失败", instance.Name),
	}, func() error {
		return s.RestoreInstance(req)
	})
	return task, nil
}

func testDBConnection(server *model.DatabaseServer) error {
	switch server.Type {
	case "mysql":
		client, err := dbUtil.NewMysqlClient(server.Address, server.Port, server.Username, server.Password)
		if err != nil {
			return err
		}
		client.Close()
	case "postgresql":
		client, err := dbUtil.NewPostgresClient(server.Address, server.Port, server.Username, server.Password)
		if err != nil {
			return err
		}
		client.Close()
	}
	return nil
}
