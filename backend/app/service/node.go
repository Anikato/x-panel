package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"

	"golang.org/x/crypto/ssh"
)

type INodeService interface {
	Create(req dto.NodeCreate) error
	Update(req dto.NodeUpdate) error
	Delete(id uint) error
	List() ([]dto.NodeInfo, error)
	Get(id uint) (*model.Node, error)
	TestConnection(id uint) error
	TestSSH(req dto.NodeSSHTest) error
	AgentAction(req dto.NodeAgentAction) (string, error)
	ProxyRequest(nodeID uint, method, path string, body io.Reader) ([]byte, int, error)
	StartHeartbeat()
}

func NewINodeService() INodeService {
	return &NodeService{repo: repo.NewINodeRepo()}
}

type NodeService struct {
	repo repo.INodeRepo
}

func (s *NodeService) Create(req dto.NodeCreate) error {
	port := req.PanelPort
	if port == "" {
		port = "7777"
	}
	sshPort := req.SSHPort
	if sshPort == 0 {
		sshPort = 22
	}
	token := req.AgentToken
	if token == "" {
		token = generateRandomToken()
	}
	node := &model.Node{
		Name:        req.Name,
		Address:     req.SSHHost + ":" + port,
		Token:       token,
		SSHHost:     req.SSHHost,
		SSHPort:     sshPort,
		SSHUser:     req.SSHUser,
		SSHPassword: req.SSHPassword,
		GroupID:     req.GroupID,
		Status:      "offline",
	}
	return s.repo.Create(node)
}

func (s *NodeService) Update(req dto.NodeUpdate) error {
	fields := map[string]interface{}{
		"name":     req.Name,
		"group_id": req.GroupID,
	}
	if req.SSHHost != "" {
		fields["ssh_host"] = req.SSHHost
		port := req.PanelPort
		if port == "" {
			port = "7777"
		}
		fields["address"] = req.SSHHost + ":" + port
	}
	if req.SSHPort > 0 {
		fields["ssh_port"] = req.SSHPort
	}
	if req.SSHUser != "" {
		fields["ssh_user"] = req.SSHUser
	}
	if req.SSHPassword != "" {
		fields["ssh_password"] = req.SSHPassword
	}
	if req.AgentToken != "" {
		fields["token"] = req.AgentToken
	}
	return s.repo.Update(req.ID, fields)
}

func (s *NodeService) Delete(id uint) error {
	return s.repo.Delete(id)
}

func (s *NodeService) List() ([]dto.NodeInfo, error) {
	nodes, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	var items []dto.NodeInfo
	for _, n := range nodes {
		items = append(items, dto.NodeInfo{
			ID: n.ID, CreatedAt: n.CreatedAt, Name: n.Name,
			Address: n.Address, Status: n.Status, GroupID: n.GroupID,
			OS: n.OS, Hostname: n.Hostname, CpuCores: n.CpuCores, MemTotal: n.MemTotal,
			SSHHost: n.SSHHost, SSHPort: n.SSHPort, SSHUser: n.SSHUser,
		})
	}
	return items, nil
}

func (s *NodeService) Get(id uint) (*model.Node, error) {
	return s.repo.Get(id)
}

func (s *NodeService) TestConnection(id uint) error {
	node, err := s.repo.Get(id)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	_, statusCode, err := s.doRequest(node, "GET", "/api/v1/version", nil)
	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}
	if statusCode != 200 {
		return fmt.Errorf("unexpected status: %d", statusCode)
	}
	return nil
}

func (s *NodeService) TestSSH(req dto.NodeSSHTest) error {
	port := req.SSHPort
	if port == 0 {
		port = 22
	}
	client, err := sshConnect(req.SSHHost, port, req.SSHUser, req.SSHPassword)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %v", err)
	}
	client.Close()
	return nil
}

func (s *NodeService) AgentAction(req dto.NodeAgentAction) (string, error) {
	node, err := s.repo.Get(req.ID)
	if err != nil {
		return "", buserr.New(constant.ErrRecordNotFound)
	}
	client, err := sshConnect(node.SSHHost, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		return "", fmt.Errorf("SSH connection failed: %v", err)
	}
	defer client.Close()

	var cmd string
	switch req.Action {
	case "install":
		cmd = fmt.Sprintf(
			"curl -sSL https://raw.githubusercontent.com/Anikato/x-panel/main/scripts/install-online.sh | bash -s -- --agent-token %s",
			node.Token,
		)
	case "uninstall":
		cmd = "curl -sSL https://raw.githubusercontent.com/Anikato/x-panel/main/scripts/install-online.sh | bash -s -- --uninstall --yes"
	case "update":
		cmd = "curl -sSL https://raw.githubusercontent.com/Anikato/x-panel/main/scripts/install-online.sh | bash"
	default:
		return "", fmt.Errorf("unknown action: %s", req.Action)
	}

	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %v\n%s", err, string(output))
	}
	return string(output), nil
}

func (s *NodeService) ProxyRequest(nodeID uint, method, path string, body io.Reader) ([]byte, int, error) {
	node, err := s.repo.Get(nodeID)
	if err != nil {
		return nil, 0, buserr.New(constant.ErrRecordNotFound)
	}
	return s.doRequest(node, method, path, body)
}

func (s *NodeService) doRequest(node *model.Node, method, path string, body io.Reader) ([]byte, int, error) {
	addr := node.Address
	if !strings.HasPrefix(addr, "http") {
		addr = "http://" + addr
	}
	url := addr + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("X-Agent-Token", node.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return data, resp.StatusCode, nil
}

func (s *NodeService) StartHeartbeat() {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.checkAllNodes()
		}
	}()
}

func (s *NodeService) checkAllNodes() {
	nodes, err := s.repo.List()
	if err != nil {
		return
	}
	for _, node := range nodes {
		data, statusCode, err := s.doRequest(&node, "GET", "/api/v1/version", nil)
		status := "offline"
		if err == nil && statusCode == 200 {
			status = "online"
			var info map[string]interface{}
			if json.Unmarshal(data, &info) == nil {
				if d, ok := info["data"].(map[string]interface{}); ok {
					if os, ok := d["os"].(string); ok {
						_ = s.repo.Update(node.ID, map[string]interface{}{"os": os})
					}
				}
			}
		}
		if node.Status != status {
			if err := s.repo.Update(node.ID, map[string]interface{}{"status": status}); err != nil {
				global.LOG.Errorf("update node [%s] status failed: %v", node.Name, err)
			}
		}
	}
}

func sshConnect(host string, port uint, user, password string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}

func generateRandomToken() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	for i := range b {
		val := time.Now().UnixNano() + int64(i)*31
		if val < 0 {
			val = -val
		}
		b[i] = chars[val%int64(len(chars))]
	}
	return string(b)
}
