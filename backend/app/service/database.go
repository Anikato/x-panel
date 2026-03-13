package service

import (
	"fmt"
	"os/exec"
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
		if err := client.CreateDatabase(req.Name, req.Charset); err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		if req.Password != "" {
			_ = client.CreateUser(req.Name, req.Password, req.Name)
		}
	case "postgresql":
		client, err := dbUtil.NewPostgresClient(server.Address, server.Port, server.Username, server.Password)
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		defer client.Close()
		owner := req.Owner
		if owner == "" {
			owner = server.Username
		}
		if err := client.CreateDatabase(req.Name, owner); err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
	}

	instance := &model.DatabaseInstance{
		ServerID: req.ServerID,
		Name:     req.Name,
		Charset:  req.Charset,
		Owner:    req.Owner,
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
				_ = client.DeleteUser(instance.Name)
			}
		case "postgresql":
			client, err := dbUtil.NewPostgresClient(server.Address, server.Port, server.Username, server.Password)
			if err == nil {
				defer client.Close()
				_ = client.DeleteDatabase(instance.Name)
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
			_ = s.repo.UpdateInstance(existing.ID, map[string]interface{}{
				"charset": info.Charset,
				"owner":   info.Owner,
			})
		} else {
			_ = s.repo.CreateInstance(&model.DatabaseInstance{
				ServerID: serverID,
				Name:     info.Name,
				Charset:  info.Charset,
				Owner:    info.Owner,
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
		if err := client.ChangePassword(instance.Name, req.Password); err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
	case "postgresql":
		client, err := dbUtil.NewPostgresClient(server.Address, server.Port, server.Username, server.Password)
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		defer client.Close()
		userName := instance.Owner
		if userName == "" {
			userName = instance.Name
		}
		if err := client.ChangePassword(userName, req.Password); err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
	}
	return nil
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
	_ = exec.Command("mkdir", "-p", backupDir).Run()
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

