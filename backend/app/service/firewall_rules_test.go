package service

import (
	"strings"
	"testing"
)

func TestParseUFWRulesSeparatesAllowFromAndPortRules(t *testing.T) {
	output := `
Status: active

     To                         Action      From
     --                         ------      ----
[ 1] 22/tcp                     ALLOW IN    Anywhere
[ 2] 1.2.3.4                    ALLOW IN    Anywhere
[ 3] Anywhere                   DENY IN     10.0.0.8
[ 4] 443                        ALLOW IN    10.1.0.0/16
[ 5] 22/tcp (v6)                ALLOW IN    Anywhere (v6)
`
	rules := parseUFWRules(output)
	if len(rules) != 5 {
		t.Fatalf("rules = %d, want 5", len(rules))
	}
	if !rules[0].IsPort || rules[0].Number != 1 || rules[0].Port != "22" || rules[0].Protocol != "tcp" {
		t.Fatalf("port rule = %#v", rules[0])
	}
	if rules[1].IsPort || rules[1].Number != 2 || rules[1].Address != "1.2.3.4" || rules[1].Strategy != "allow" {
		t.Fatalf("allow-from = %#v", rules[1])
	}
	if rules[2].IsPort || rules[2].Address != "10.0.0.8" || rules[2].Strategy != "deny" {
		t.Fatalf("deny-from = %#v", rules[2])
	}
	if !rules[3].IsPort || rules[3].From != "10.1.0.0/16" {
		t.Fatalf("restricted port = %#v", rules[3])
	}
	if !rules[4].IsPort || rules[4].Number != 5 {
		t.Fatalf("v6 port = %#v", rules[4])
	}
}

func TestDeleteUFWByNumberUsesForce(t *testing.T) {
	previous := execCommand
	t.Cleanup(func() { execCommand = previous })
	var got []string
	execCommand = func(name string, args ...string) (string, error) {
		got = append([]string{name}, args...)
		return "", nil
	}
	if err := deleteUFWByNumber(2); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "ufw --force delete 2" {
		t.Fatalf("command = %q", strings.Join(got, " "))
	}
}

func TestFirewallSpecValidation(t *testing.T) {
	if !validPortSpec("80") || !validPortSpec("8000:8100") || validPortSpec("0") || validPortSpec("22;reboot") {
		t.Fatal("port validation mismatch")
	}
	if !validIPSpec("1.2.3.4") || !validIPSpec("10.0.0.0/8") || validIPSpec("Anywhere") || validIPSpec("1.2.3.4\nroot") {
		t.Fatal("address validation mismatch")
	}
}
