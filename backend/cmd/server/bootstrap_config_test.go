package main

import "testing"

func TestCollectBootstrapSettingsUsesOnlyApprovedEnvironmentVariables(t *testing.T) {
	env := map[string]string{
		"XPANEL_SECURITY_ENTRANCE": "internal-entry",
		"XPANEL_AGENT_TOKEN":       "agent-secret",
		"XPANEL_GITHUB_TOKEN":      "must-not-be-accepted",
	}
	settings, err := collectBootstrapSettings(func(key string) string { return env[key] })
	if err != nil {
		t.Fatalf("collect bootstrap settings: %v", err)
	}
	if len(settings) != 2 ||
		settings["SecurityEntrance"] != "internal-entry" ||
		settings["AgentToken"] != "agent-secret" {
		t.Fatalf("bootstrap settings = %#v", settings)
	}
	if _, exists := settings["GitHubToken"]; exists {
		t.Fatalf("unapproved setting was accepted: %#v", settings)
	}
}

func TestCollectBootstrapSettingsRejectsEmptyInput(t *testing.T) {
	if _, err := collectBootstrapSettings(func(string) string { return "" }); err == nil {
		t.Fatalf("empty bootstrap configuration should fail")
	}
}
