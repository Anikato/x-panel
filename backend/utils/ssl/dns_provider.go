package ssl

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/providers/dns/alidns"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/providers/dns/dnspod"
	"github.com/go-acme/lego/v4/providers/dns/godaddy"
	"github.com/go-acme/lego/v4/providers/dns/huaweicloud"
	"github.com/go-acme/lego/v4/providers/dns/namesilo"
	"github.com/go-acme/lego/v4/providers/dns/tencentcloud"
)

var (
	propagationTimeout = 30 * time.Minute
	pollingInterval    = 10 * time.Second
	ttl                = 3600
)

// DNSParam DNS 提供商通用参数
type DNSParam struct {
	ID        string `json:"id"`
	Token     string `json:"token"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	Email     string `json:"email"`
	APIKey    string `json:"apiKey"`
	APIUser   string `json:"apiUser"`
	APISecret string `json:"apiSecret"`
	SecretID  string `json:"secretID"`
	Region    string `json:"region"`
}

// SupportedDNSProviders 返回支持的 DNS 提供商列表
func SupportedDNSProviders() []map[string]string {
	return []map[string]string{
		{"value": "CloudFlare", "label": "Cloudflare", "fields": "email,apiKey"},
		{"value": "AliYun", "label": "阿里云 DNS", "fields": "accessKey,secretKey"},
		{"value": "DnsPod", "label": "DNSPod", "fields": "id,token"},
		{"value": "TencentCloud", "label": "腾讯云 DNS", "fields": "secretID,secretKey"},
		{"value": "HuaweiCloud", "label": "华为云 DNS", "fields": "accessKey,secretKey,region"},
		{"value": "NameSilo", "label": "NameSilo", "fields": "apiKey"},
		{"value": "GoDaddy", "label": "GoDaddy", "fields": "apiKey,apiSecret"},
	}
}

// GetDNSProvider 根据类型和参数创建 DNS 验证提供商
func GetDNSProvider(dnsType, authJSON string) (challenge.Provider, error) {
	var param DNSParam
	if err := json.Unmarshal([]byte(authJSON), &param); err != nil {
		return nil, fmt.Errorf("parse DNS params: %v", err)
	}

	switch dnsType {
	case "CloudFlare":
		config := cloudflare.NewDefaultConfig()
		config.AuthEmail = param.Email
		config.AuthToken = param.APIKey
		config.PropagationTimeout = propagationTimeout
		config.PollingInterval = pollingInterval
		config.TTL = ttl
		return cloudflare.NewDNSProviderConfig(config)

	case "AliYun":
		config := alidns.NewDefaultConfig()
		config.APIKey = param.AccessKey
		config.SecretKey = param.SecretKey
		config.PropagationTimeout = propagationTimeout
		config.PollingInterval = pollingInterval
		config.TTL = ttl
		return alidns.NewDNSProviderConfig(config)

	case "DnsPod":
		config := dnspod.NewDefaultConfig()
		config.LoginToken = param.ID + "," + param.Token
		config.PropagationTimeout = propagationTimeout
		config.PollingInterval = pollingInterval
		config.TTL = ttl
		return dnspod.NewDNSProviderConfig(config)

	case "TencentCloud":
		config := tencentcloud.NewDefaultConfig()
		config.SecretID = param.SecretID
		config.SecretKey = param.SecretKey
		config.PropagationTimeout = propagationTimeout
		config.PollingInterval = pollingInterval
		config.TTL = ttl
		return tencentcloud.NewDNSProviderConfig(config)

	case "HuaweiCloud":
		config := huaweicloud.NewDefaultConfig()
		config.AccessKeyID = param.AccessKey
		config.SecretAccessKey = param.SecretKey
		config.Region = param.Region
		if config.Region == "" {
			config.Region = "cn-north-1"
		}
		config.PropagationTimeout = propagationTimeout
		config.PollingInterval = pollingInterval
		config.TTL = int32(ttl)
		return huaweicloud.NewDNSProviderConfig(config)

	case "NameSilo":
		config := namesilo.NewDefaultConfig()
		config.APIKey = param.APIKey
		config.PropagationTimeout = propagationTimeout
		config.PollingInterval = pollingInterval
		config.TTL = ttl
		return namesilo.NewDNSProviderConfig(config)

	case "GoDaddy":
		config := godaddy.NewDefaultConfig()
		config.APIKey = param.APIKey
		config.APISecret = param.APISecret
		config.PropagationTimeout = propagationTimeout
		config.PollingInterval = pollingInterval
		config.TTL = ttl
		return godaddy.NewDNSProviderConfig(config)

	default:
		return nil, fmt.Errorf("unsupported DNS provider: %s", dnsType)
	}
}
