package service

import (
	"fmt"
	"net"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"

	"golang.org/x/crypto/ssh"
)

type IHostService interface {
	Create(req dto.HostCreate) error
	Update(req dto.HostUpdate) error
	Delete(id uint) error
	GetByID(id uint) (*dto.HostInfo, error)
	SearchWithPage(req dto.SearchHostReq) (int64, []dto.HostInfo, error)
	GetTree() ([]dto.HostTree, error)
	TestByID(id uint) bool
	ConnSSH(id uint) (*ssh.Client, error)
}

type HostService struct {
	hostRepo  repo.IHostRepo
	groupRepo repo.IGroupRepo
}

func NewIHostService() IHostService {
	return &HostService{
		hostRepo:  repo.NewIHostRepo(),
		groupRepo: repo.NewIGroupRepo(),
	}
}

func (s *HostService) Create(req dto.HostCreate) error {
	host := model.Host{
		GroupID:     req.GroupID,
		Name:        req.Name,
		Addr:        req.Addr,
		Port:        req.Port,
		User:        req.User,
		AuthMode:    req.AuthMode,
		Password:    req.Password,
		PrivateKey:  req.PrivateKey,
		PassPhrase:  req.PassPhrase,
		Description: req.Description,
	}
	if err := s.hostRepo.Create(&host); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	global.LOG.Infof("Host created: %s (%s:%d)", req.Name, req.Addr, req.Port)
	return nil
}

func (s *HostService) Update(req dto.HostUpdate) error {
	updates := map[string]interface{}{
		"group_id":    req.GroupID,
		"name":        req.Name,
		"addr":        req.Addr,
		"port":        req.Port,
		"user":        req.User,
		"auth_mode":   req.AuthMode,
		"password":    req.Password,
		"private_key": req.PrivateKey,
		"pass_phrase": req.PassPhrase,
		"description": req.Description,
	}
	if err := s.hostRepo.Update(req.ID, updates); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	return nil
}

func (s *HostService) Delete(id uint) error {
	return s.hostRepo.Delete(repo.WithByID(id))
}

func (s *HostService) GetByID(id uint) (*dto.HostInfo, error) {
	host, err := s.hostRepo.Get(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}
	info := s.toHostInfo(host)
	return &info, nil
}

func (s *HostService) SearchWithPage(req dto.SearchHostReq) (int64, []dto.HostInfo, error) {
	var opts []repo.DBOption
	if req.Info != "" {
		opts = append(opts, repo.WithLikeName(req.Info))
	}
	if req.GroupID > 0 {
		opts = append(opts, repo.WithByGroupID(req.GroupID))
	}
	total, hosts, err := s.hostRepo.Page(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}
	var items []dto.HostInfo
	for _, h := range hosts {
		items = append(items, s.toHostInfo(h))
	}
	return total, items, nil
}

func (s *HostService) GetTree() ([]dto.HostTree, error) {
	groups, _ := s.groupRepo.GetList(repo.WithByType("host"))
	hosts, err := s.hostRepo.GetList()
	if err != nil {
		return nil, err
	}

	groupMap := make(map[uint]string)
	for _, g := range groups {
		groupMap[g.ID] = g.Name
	}

	treeMap := make(map[uint][]dto.HostTree)
	for _, h := range hosts {
		treeMap[h.GroupID] = append(treeMap[h.GroupID], dto.HostTree{
			ID:    h.ID,
			Label: fmt.Sprintf("%s (%s)", h.Name, h.Addr),
		})
	}

	var tree []dto.HostTree
	// Default group
	if items, ok := treeMap[0]; ok {
		tree = append(tree, dto.HostTree{Label: "Default", Children: items})
	}
	for _, g := range groups {
		if items, ok := treeMap[g.ID]; ok {
			tree = append(tree, dto.HostTree{ID: g.ID, Label: g.Name, Children: items})
		}
	}
	return tree, nil
}

func (s *HostService) TestByID(id uint) bool {
	client, err := s.ConnSSH(id)
	if err != nil {
		global.LOG.Warnf("SSH test failed for host %d: %v", id, err)
		return false
	}
	client.Close()
	return true
}

func (s *HostService) ConnSSH(id uint) (*ssh.Client, error) {
	host, err := s.hostRepo.Get(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}

	config := &ssh.ClientConfig{
		User:            host.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	switch host.AuthMode {
	case "password":
		config.Auth = []ssh.AuthMethod{ssh.Password(host.Password)}
	case "key":
		var signer ssh.Signer
		if host.PassPhrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(host.PrivateKey), []byte(host.PassPhrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(host.PrivateKey))
		}
		if err != nil {
			return nil, fmt.Errorf("parse private key failed: %v", err)
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	default:
		return nil, fmt.Errorf("unsupported auth mode: %s", host.AuthMode)
	}

	addr := fmt.Sprintf("%s:%d", host.Addr, host.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("ssh dial failed: %v", err)
	}
	return client, nil
}

func (s *HostService) toHostInfo(h model.Host) dto.HostInfo {
	info := dto.HostInfo{
		ID:          h.ID,
		GroupID:     h.GroupID,
		Name:        h.Name,
		Addr:        h.Addr,
		Port:        h.Port,
		User:        h.User,
		AuthMode:    h.AuthMode,
		Description: h.Description,
	}
	if h.GroupID > 0 {
		if g, err := s.groupRepo.Get(repo.WithByID(h.GroupID)); err == nil {
			info.GroupName = g.Name
		}
	}
	return info
}

// TestHostConn 测试主机直连（不保存）
func TestHostConn(addr string, port int, user, authMode, password, privateKey, passPhrase string) error {
	config := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	switch authMode {
	case "password":
		config.Auth = []ssh.AuthMethod{ssh.Password(password)}
	case "key":
		var signer ssh.Signer
		var err error
		if passPhrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(privateKey), []byte(passPhrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(privateKey))
		}
		if err != nil {
			return err
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	}
	target := fmt.Sprintf("%s:%d", addr, port)
	conn, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	c, chans, reqs, err := ssh.NewClientConn(conn, target, config)
	if err != nil {
		return err
	}
	client := ssh.NewClient(c, chans, reqs)
	client.Close()
	return nil
}
