package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
)

type INodeService interface {
	Create(req dto.NodeCreate) error
	Update(req dto.NodeUpdate) error
	Delete(id uint) error
	List() ([]dto.NodeInfo, error)
	Get(id uint) (*model.Node, error)
	TestConnection(id uint) error
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
	node := &model.Node{
		Name:    req.Name,
		Address: req.Address,
		Token:   req.Token,
		GroupID: req.GroupID,
		Status:  "offline",
	}
	return s.repo.Create(node)
}

func (s *NodeService) Update(req dto.NodeUpdate) error {
	fields := map[string]interface{}{
		"name":     req.Name,
		"address":  req.Address,
		"group_id": req.GroupID,
	}
	if req.Token != "" {
		fields["token"] = req.Token
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
