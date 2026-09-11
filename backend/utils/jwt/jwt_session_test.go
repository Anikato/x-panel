package jwt

import (
	"testing"

	"xpanel/global"
)

func TestParseTokenKeepsSessionID(t *testing.T) {
	previous := global.CONF
	global.CONF.System.JwtSecret = "test-secret"
	t.Cleanup(func() { global.CONF = previous })

	token, err := GenerateSessionToken("admin", "session-a", 60)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.SessionID != "session-a" {
		t.Fatalf("session = %q", claims.SessionID)
	}
}
