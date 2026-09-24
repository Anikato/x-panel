package service

import "testing"

func TestValidateLinuxAccountRejectsFlagsAndNewlines(t *testing.T) {
	if err := validateLinuxAccount("alice", "secret", "/home/alice", "/bin/bash", ""); err != nil {
		t.Fatal(err)
	}
	if err := validateLinuxAccount("-rf", "secret", "", "", ""); err == nil {
		t.Fatal("username starting with a dash was accepted")
	}
	if err := validateLinuxAccount("alice", "secret\nroot:owned", "", "", ""); err == nil {
		t.Fatal("password newline was accepted")
	}
	if err := validateLinuxAccount("alice", "", "/tmp\n/root", "", ""); err == nil {
		t.Fatal("home newline was accepted")
	}
}
