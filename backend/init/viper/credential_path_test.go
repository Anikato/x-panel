package viper

import "testing"

func TestResolveCredentialKeyPath(t *testing.T) {
	tests := []struct {
		name       string
		dataDir    string
		configured string
		want       string
	}{
		{
			name:    "legacy config follows custom install root",
			dataDir: "/srv/custom/data",
			want:    "/srv/custom/secrets/credential-keyring.json",
		},
		{
			name:       "absolute configured path",
			dataDir:    "/srv/custom/data",
			configured: "/run/secrets/xpanel-keyring.json",
			want:       "/run/secrets/xpanel-keyring.json",
		},
		{
			name:       "relative configured path follows install root",
			dataDir:    "/srv/custom/data",
			configured: "private/keyring.json",
			want:       "/srv/custom/private/keyring.json",
		},
		{
			name:    "top-level data directory remains its own install root",
			dataDir: "/data",
			want:    "/data/secrets/credential-keyring.json",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := resolveCredentialKeyPath(test.dataDir, test.configured); got != test.want {
				t.Fatalf("resolveCredentialKeyPath(%q, %q) = %q, want %q",
					test.dataDir, test.configured, got, test.want)
			}
		})
	}
}
