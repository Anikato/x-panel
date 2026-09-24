package ssl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/dara"
	alidns "github.com/go-acme/alidns-20150109/v4/client"
)

func EnsureAliDNSZones(authJSON string, domains []string) error {
	var param DNSParam
	if err := json.Unmarshal([]byte(authJSON), &param); err != nil {
		return fmt.Errorf("阿里云 DNS 参数无法解析: %v", err)
	}
	names, err := listAliDNSDomainNames(param.AccessKey, param.SecretKey)
	if err != nil {
		return err
	}
	var missing []string
	for _, domain := range domains {
		if _, ok := aliDNSZoneForDomain(domain, names); !ok {
			missing = append(missing, strings.TrimPrefix(strings.TrimSpace(domain), "*."))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("阿里云账号下没有这些域名的解析：%s", strings.Join(missing, ", "))
	}
	return nil
}

func listAliDNSDomainNames(accessKey, secretKey string) ([]string, error) {
	if strings.TrimSpace(accessKey) == "" || strings.TrimSpace(secretKey) == "" {
		return nil, fmt.Errorf("阿里云 AccessKey 或 SecretKey 为空")
	}
	cfg := new(openapi.Config).
		SetRegionId("cn-hangzhou").
		SetAccessKeyId(strings.TrimSpace(accessKey)).
		SetAccessKeySecret(strings.TrimSpace(secretKey)).
		SetReadTimeout(10000)
	client, err := alidns.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("阿里云 DNS 客户端创建失败: %v", err)
	}
	var names []string
	var page int64 = 1
	for {
		request := new(alidns.DescribeDomainsRequest).SetPageNumber(page).SetPageSize(100)
		response, err := alidns.DescribeDomainsWithContext(context.Background(), client, request, &dara.RuntimeOptions{})
		if err != nil {
			return nil, fmt.Errorf("阿里云 DNS 接口调用失败: %v", err)
		}
		if response == nil || response.Body == nil || response.Body.Domains == nil {
			break
		}
		for _, zone := range response.Body.Domains.Domain {
			if zone == nil {
				continue
			}
			if zone.DomainName == nil {
				continue
			}
			if name := strings.TrimSpace(*zone.DomainName); name != "" {
				names = append(names, name)
			}
		}
		pageNumber := derefInt64(response.Body.PageNumber)
		pageSize := derefInt64(response.Body.PageSize)
		total := derefInt64(response.Body.TotalCount)
		if pageNumber*pageSize >= total {
			break
		}
		page++
	}
	return names, nil
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func aliDNSZoneForDomain(domain string, accountZones []string) (string, bool) {
	domain = strings.TrimSuffix(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(domain)), "*."), ".")
	if domain == "" || !strings.Contains(domain, ".") {
		return "", false
	}
	zones := make(map[string]string, len(accountZones))
	for _, zone := range accountZones {
		zone = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(zone)), ".")
		if zone != "" {
			zones[zone] = zone
		}
	}
	labels := strings.Split(domain, ".")
	for i := 0; i < len(labels)-1; i++ {
		candidate := strings.Join(labels[i:], ".")
		if zone, ok := zones[candidate]; ok {
			return zone, true
		}
	}
	return "", false
}
