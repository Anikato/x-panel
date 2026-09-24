package service

import "testing"

func TestCertServerTokenSuffixLeavesSecretHidden(t *testing.T) {
	if certServerTokenSuffix("") != "" || certServerTokenSuffix("abc") != "" {
		t.Fatal("short tokens should not expose a suffix")
	}
	if certServerTokenSuffix("certificate-server-secret") != "cret" {
		t.Fatal("suffix should be the last four characters")
	}
}
