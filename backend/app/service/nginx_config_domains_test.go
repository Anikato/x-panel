package service

import (
	"reflect"
	"testing"

	"xpanel/app/model"
)

func TestCollectDomainsSplitsCommonSeparators(t *testing.T) {
	got := NewNginxConfigGenerator().collectDomains(model.Website{
		PrimaryDomain: "example.com",
		Domains:       "www.example.com，api.example.com\ncdn.example.com; static.example.com example.com",
	})
	want := []string{"example.com", "www.example.com", "api.example.com", "cdn.example.com", "static.example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("domains = %#v, want %#v", got, want)
	}
	if normalizeExtraDomains("example.com", got[1]) != "www.example.com" {
		t.Fatal("single extra domain was not preserved")
	}
	stored := normalizeExtraDomains("example.com", "www.example.com，api.example.com example.com")
	if stored != "www.example.com,api.example.com" {
		t.Fatalf("stored = %q", stored)
	}
}
