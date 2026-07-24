package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"xpanel/app/model"
	"xpanel/global"
	"xpanel/security/credentials"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestFleetRegisterUsesAndClearsEnrollmentToken(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	previousDB := global.DB
	previousCredentials := global.CREDENTIALS
	global.DB = db
	manager, _, err := credentials.LoadOrCreate(
		filepath.Join(t.TempDir(), "secrets", "credential-keyring.json"),
		true,
	)
	if err != nil {
		t.Fatalf("create credential manager: %v", err)
	}
	global.CREDENTIALS = manager
	t.Cleanup(func() {
		global.DB = previousDB
		global.CREDENTIALS = previousCredentials
	})
	if err := db.AutoMigrate(&model.Setting{}); err != nil {
		t.Fatalf("migrate settings: %v", err)
	}
	for key, value := range map[string]string{
		"FleetEnrollmentToken": "fenr_test-secret-with-sufficient-entropy",
		"FleetInstanceToken":   "",
	} {
		if err := settingRepo.CreateOrUpdate(key, value); err != nil {
			t.Fatalf("seed %s: %v", key, err)
		}
	}

	const instanceID = "test-instance"
	const instanceToken = "finst_server-issued-token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Fleet-Enrollment-Token"); got != "fenr_test-secret-with-sufficient-entropy" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(fleetResponse{
				Code:    http.StatusUnauthorized,
				Message: "missing enrollment token",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(fleetResponse{
			Code:    http.StatusOK,
			Message: "success",
			Data: json.RawMessage(
				`{"instanceId":"` + instanceID + `","instanceToken":"` + instanceToken + `"}`,
			),
		})
	}))
	defer server.Close()

	reporter := &FleetReporterService{client: server.Client()}
	if err := reporter.register(server.URL, fleetPayload{InstanceID: instanceID}); err != nil {
		t.Fatalf("register: %v", err)
	}

	storedInstanceToken, err := settingRepo.GetValueByKey("FleetInstanceToken")
	if err != nil {
		t.Fatalf("load instance token: %v", err)
	}
	if storedInstanceToken != instanceToken {
		t.Fatalf("instance token = %q, want %q", storedInstanceToken, instanceToken)
	}
	storedEnrollmentToken, err := settingRepo.GetValueByKey("FleetEnrollmentToken")
	if err != nil {
		t.Fatalf("load enrollment token: %v", err)
	}
	if storedEnrollmentToken != "" {
		t.Fatalf("enrollment token was not cleared: %q", storedEnrollmentToken)
	}
}
