package gost

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps the GOST Web API.
type Client struct {
	BaseURL  string
	Username string
	Password string
	client   *http.Client
}

func NewClient(addr, user, pass string) *Client {
	return &Client{
		BaseURL:  "http://" + addr,
		Username: user,
		Password: pass,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// --- GOST native config types ---

type ServiceConfig struct {
	Name      string            `json:"name" yaml:"name"`
	Addr      string            `json:"addr" yaml:"addr"`
	Handler   HandlerConfig     `json:"handler" yaml:"handler"`
	Listener  ListenerConfig    `json:"listener" yaml:"listener"`
	Forwarder *ForwarderConfig  `json:"forwarder,omitempty" yaml:"forwarder,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Status    *ServiceStatus    `json:"status,omitempty" yaml:"status,omitempty"`
}

type HandlerConfig struct {
	Type     string            `json:"type" yaml:"type"`
	Chain    string            `json:"chain,omitempty" yaml:"chain,omitempty"`
	Auth     *AuthConfig       `json:"auth,omitempty" yaml:"auth,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type ListenerConfig struct {
	Type     string            `json:"type" yaml:"type"`
	TLS      *TLSConfig        `json:"tls,omitempty" yaml:"tls,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type TLSConfig struct {
	CertFile string `json:"certFile,omitempty" yaml:"certFile,omitempty"`
	KeyFile  string `json:"keyFile,omitempty" yaml:"keyFile,omitempty"`
}

type ForwarderConfig struct {
	Nodes []ForwarderNode `json:"nodes" yaml:"nodes"`
}

type ForwarderNode struct {
	Name string `json:"name" yaml:"name"`
	Addr string `json:"addr" yaml:"addr"`
}

type AuthConfig struct {
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
}

type ChainConfig struct {
	Name string    `json:"name" yaml:"name"`
	Hops []HopItem `json:"hops" yaml:"hops"`
}

type HopItem struct {
	Name  string     `json:"name" yaml:"name"`
	Nodes []NodeItem `json:"nodes" yaml:"nodes"`
}

type NodeItem struct {
	Name      string      `json:"name" yaml:"name"`
	Addr      string      `json:"addr" yaml:"addr"`
	Connector Connector   `json:"connector" yaml:"connector"`
	Dialer    Dialer      `json:"dialer" yaml:"dialer"`
}

type Connector struct {
	Type string `json:"type" yaml:"type"`
}

type Dialer struct {
	Type string      `json:"type" yaml:"type"`
	Auth *AuthConfig `json:"auth,omitempty" yaml:"auth,omitempty"`
}

type ServiceStatus struct {
	CreateTime int64  `json:"createTime"`
	State      string `json:"state"`
	Stats      *Stats `json:"stats,omitempty"`
}

type Stats struct {
	TotalConns   uint64 `json:"totalConns"`
	CurrentConns uint64 `json:"currentConns"`
	InputBytes   uint64 `json:"inputBytes"`
	OutputBytes  uint64 `json:"outputBytes"`
	TotalErrs    uint64 `json:"totalErrs"`
}

type FullConfig struct {
	Services []ServiceConfig `json:"services,omitempty"`
	Chains   []ChainConfig   `json:"chains,omitempty"`
}

// --- API methods ---

func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, err
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Username != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GOST API %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

// Ping checks whether the GOST API is reachable.
func (c *Client) Ping() bool {
	_, err := c.doRequest("GET", "/config", nil)
	return err == nil
}

// GetConfig returns the full running config.
func (c *Client) GetConfig() (*FullConfig, error) {
	data, err := c.doRequest("GET", "/config", nil)
	if err != nil {
		return nil, err
	}
	var cfg FullConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveConfig persists the running config to the YAML file.
func (c *Client) SaveConfig() error {
	_, err := c.doRequest("POST", "/config", nil)
	return err
}

// GetServiceStats returns a map of service name -> stats for all services.
func (c *Client) GetServiceStats() (map[string]*Stats, error) {
	cfg, err := c.GetConfig()
	if err != nil {
		return nil, err
	}
	result := make(map[string]*Stats)
	for _, svc := range cfg.Services {
		if svc.Status != nil && svc.Status.Stats != nil {
			result[svc.Name] = svc.Status.Stats
		}
	}
	return result, nil
}

// --- Service CRUD ---

func (c *Client) CreateService(svc ServiceConfig) error {
	_, err := c.doRequest("POST", "/config/services", svc)
	return err
}

func (c *Client) UpdateService(name string, svc ServiceConfig) error {
	_, err := c.doRequest("PUT", "/config/services/"+name, svc)
	return err
}

func (c *Client) DeleteService(name string) error {
	_, err := c.doRequest("DELETE", "/config/services/"+name, nil)
	return err
}

// --- Chain CRUD ---

func (c *Client) CreateChain(chain ChainConfig) error {
	_, err := c.doRequest("POST", "/config/chains", chain)
	return err
}

func (c *Client) UpdateChain(name string, chain ChainConfig) error {
	_, err := c.doRequest("PUT", "/config/chains/"+name, chain)
	return err
}

func (c *Client) DeleteChain(name string) error {
	_, err := c.doRequest("DELETE", "/config/chains/"+name, nil)
	return err
}
