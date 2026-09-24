package auth

import (
	"path/filepath"
	"testing"
	"time"
)

func TestFailureCounterSurvivesRestartAndCountsAcrossAddresses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "login-failures.json")
	tracker := NewIPTracker()
	if err := tracker.UseFile(path); err != nil {
		t.Fatal(err)
	}
	tracker.IncrementFail("10.0.0.1")
	tracker.IncrementFail("10.0.0.2")
	tracker.IncrementFail("10.0.0.3")
	if !tracker.NeedCaptcha("10.9.9.9") {
		t.Fatal("distributed failures should require captcha")
	}

	reloaded := NewIPTracker()
	if err := reloaded.UseFile(path); err != nil {
		t.Fatal(err)
	}
	if !reloaded.NeedCaptcha("192.0.2.10") {
		t.Fatal("captcha requirement did not survive restart")
	}
	reloaded.Clear("192.0.2.10")
	if reloaded.NeedCaptcha("192.0.2.10") {
		t.Fatal("successful login should clear the captcha requirement")
	}

	expired := NewIPTracker()
	expired.globalCount = CaptchaThreshold
	expired.globalLast = time.Now().Add(-ExpireDuration - time.Second)
	if expired.NeedCaptcha("10.0.0.8") {
		t.Fatal("expired failures should not require captcha")
	}
}
