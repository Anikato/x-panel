package ssl

import (
	"errors"
	"testing"
	"time"
)

func TestAliDNSZoneForDomain(t *testing.T) {
	zones := []string{"Hiny.cn", "example.com"}
	if zone, ok := aliDNSZoneForDomain("*.hiny.cn", zones); !ok || zone != "hiny.cn" {
		t.Fatalf("wildcard = %q %v", zone, ok)
	}
	if zone, ok := aliDNSZoneForDomain("www.hiny.cn", zones); !ok || zone != "hiny.cn" {
		t.Fatalf("www = %q %v", zone, ok)
	}
	if _, ok := aliDNSZoneForDomain("hiny.com", zones); ok {
		t.Fatal("hiny.com should not match the Aliyun account")
	}
}

func TestCertificateObtainRetryDelay(t *testing.T) {
	if certificateObtainRetryDelay(errors.New("propagation timeout")) != 0 {
		t.Fatal("ordinary failure should not retry")
	}
	if certificateObtainRetryDelay(errors.New("DomainRecordDuplicate")) != 45*time.Second {
		t.Fatal("duplicate TXT should wait before retry")
	}
}
