package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"xpanel/app/dto"
	"xpanel/buserr"
	"xpanel/constant"

	"github.com/gin-gonic/gin"
)

// fakeNezhaAgentService is a test double for the handler-facing service interface.
// Tests must not run in parallel: they swap the package-level factory.
type fakeNezhaAgentService struct {
	status    *dto.NezhaAgentStatus
	statusErr error

	configureReq *dto.NezhaAgentConfigUpdate
	configureErr error
	configureN   int

	operateOp  string
	operateErr error
	operateN   int

	installErr error
	installN   int
}

func (f *fakeNezhaAgentService) Status() (*dto.NezhaAgentStatus, error) {
	if f.statusErr != nil {
		return nil, f.statusErr
	}
	return f.status, nil
}

func (f *fakeNezhaAgentService) Configure(req dto.NezhaAgentConfigUpdate) error {
	f.configureN++
	cp := req
	f.configureReq = &cp
	return f.configureErr
}

func (f *fakeNezhaAgentService) Operate(operation string) error {
	f.operateN++
	f.operateOp = operation
	return f.operateErr
}

func (f *fakeNezhaAgentService) Install() error {
	f.installN++
	return f.installErr
}

func installFakeNezhaAgentService(t *testing.T, fake nezhaAgentService) {
	t.Helper()
	prev := newNezhaAgentService
	newNezhaAgentService = func() nezhaAgentService { return fake }
	t.Cleanup(func() {
		newNezhaAgentService = prev
	})
}

func performNezhaHandler(method, path, body string, h gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	c.Request = httptest.NewRequest(method, path, reader)
	if body != "" {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	h(c)
	return rec
}

func assertNoSecretLeak(t *testing.T, body, secret string) {
	t.Helper()
	if secret != "" && strings.Contains(body, secret) {
		t.Fatal("response leaked a secret value")
	}
}

func TestNezhaAgentStatusReturnsSafeFieldsWithoutSecrets(t *testing.T) {
	const persistedSecret = "persisted-nezha-secret-value"
	const sentinelSecret = "sentinel-status-secret-never-emit"

	fake := &fakeNezhaAgentService{
		status: &dto.NezhaAgentStatus{
			ComponentAvailable:      true,
			Configured:              true,
			ConfigHealthy:           true,
			Active:                  true,
			ServiceState:            "active",
			Enabled:                 true,
			DesiredEnabled:          true,
			Drift:                   false,
			Version:                 "1.2.3",
			UUID:                    "uuid-safe",
			DashboardURL:            "https://dash.example.com",
			Server:                  "dash.example.com:443",
			TLS:                     true,
			SecretConfigured:        true,
			RemoteOperationsEnabled: true,
			Conflicts:               []dto.NezhaAgentConflict{},
		},
	}
	installFakeNezhaAgentService(t, fake)

	rec := performNezhaHandler(http.MethodGet, "/api/v1/nezha-agent/status", "", (&NezhaAgentAPI{}).GetNezhaAgentStatus)
	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	assertNoSecretLeak(t, body, persistedSecret)
	assertNoSecretLeak(t, body, sentinelSecret)

	var resp dto.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response envelope: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("response code = %d, want 0", resp.Code)
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	dataStr := string(data)
	assertNoSecretLeak(t, dataStr, persistedSecret)
	assertNoSecretLeak(t, dataStr, sentinelSecret)

	// Safe status fields must be present.
	for _, want := range []string{
		`"componentAvailable":true`,
		`"configured":true`,
		`"configHealthy":true`,
		`"active":true`,
		`"serviceState":"active"`,
		`"enabled":true`,
		`"desiredEnabled":true`,
		`"secretConfigured":true`,
		`"dashboardUrl":"https://dash.example.com"`,
		`"remoteOperationsEnabled":true`,
	} {
		if !strings.Contains(dataStr, want) {
			t.Fatalf("status data missing safe field fragment %s", want)
		}
	}
	// Must never expose secret keys in the JSON payload.
	for _, banned := range []string{
		`"clientSecret"`,
		`"client_secret"`,
		`"AgentSecret"`,
		`"agentSecret"`,
		persistedSecret,
		sentinelSecret,
	} {
		if strings.Contains(dataStr, banned) {
			t.Fatal("status data contained a forbidden secret field or value")
		}
	}
}

func TestNezhaAgentInstallHasNoRequestBodyOrSecret(t *testing.T) {
	fake := &fakeNezhaAgentService{}
	installFakeNezhaAgentService(t, fake)
	rec := performNezhaHandler(http.MethodPost, "/api/v1/nezha-agent/install", "", (&NezhaAgentAPI{}).InstallNezhaAgent)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if fake.installN != 1 || fake.configureN != 0 || fake.operateN != 0 {
		t.Fatalf("calls install=%d configure=%d operate=%d", fake.installN, fake.configureN, fake.operateN)
	}
}

func TestNezhaAgentConfigPassesRequestWithoutEchoingSecret(t *testing.T) {
	const submittedSecret = "submitted-config-secret-xyz"

	fake := &fakeNezhaAgentService{}
	installFakeNezhaAgentService(t, fake)

	dash := "https://dash.example.com"
	body := `{"dashboardUrl":"https://dash.example.com","clientSecret":"` + submittedSecret + `","enableAndStart":true}`
	rec := performNezhaHandler(http.MethodPut, "/api/v1/nezha-agent/config", body, (&NezhaAgentAPI{}).UpdateNezhaAgentConfig)
	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
	assertNoSecretLeak(t, rec.Body.String(), submittedSecret)

	var resp dto.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("response code = %d, want 0", resp.Code)
	}
	if resp.Data != nil {
		t.Fatal("config success response must not include data")
	}
	if fake.configureN != 1 {
		t.Fatalf("Configure calls = %d, want 1", fake.configureN)
	}
	if fake.configureReq == nil {
		t.Fatal("Configure was not given a request")
	}
	if fake.configureReq.DashboardURL == nil || *fake.configureReq.DashboardURL != dash {
		t.Fatal("Configure did not receive dashboardUrl")
	}
	if fake.configureReq.ClientSecret == nil || *fake.configureReq.ClientSecret != submittedSecret {
		t.Fatal("Configure did not receive clientSecret")
	}
	if !fake.configureReq.EnableAndStart {
		t.Fatal("Configure did not receive enableAndStart=true")
	}
	assertNoSecretLeak(t, rec.Body.String(), submittedSecret)
	if strings.Contains(rec.Body.String(), `"clientSecret"`) {
		t.Fatal("config response echoed clientSecret field")
	}
}

func TestNezhaAgentConfigMalformedJSONDoesNotCallConfigure(t *testing.T) {
	const bodySentinel = "malformed-body-secret-sentinel-abc"

	fake := &fakeNezhaAgentService{}
	installFakeNezhaAgentService(t, fake)

	// Invalid JSON that still embeds a sentinel secret-like string.
	body := `{"dashboardUrl":"https://dash.example.com","clientSecret":"` + bodySentinel
	rec := performNezhaHandler(http.MethodPut, "/api/v1/nezha-agent/config", body, (&NezhaAgentAPI{}).UpdateNezhaAgentConfig)
	if fake.configureN != 0 {
		t.Fatalf("Configure calls = %d, want 0 for malformed JSON", fake.configureN)
	}
	assertNoSecretLeak(t, rec.Body.String(), bodySentinel)
	if strings.Contains(rec.Body.String(), bodySentinel) {
		t.Fatal("malformed config response contained body sentinel")
	}
	var resp dto.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code == 0 {
		t.Fatal("malformed JSON should not succeed")
	}
}

func TestNezhaAgentOperateStartPassesToService(t *testing.T) {
	fake := &fakeNezhaAgentService{}
	installFakeNezhaAgentService(t, fake)

	rec := performNezhaHandler(
		http.MethodPost,
		"/api/v1/nezha-agent/operate",
		`{"operation":"start"}`,
		(&NezhaAgentAPI{}).OperateNezhaAgent,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
	var resp dto.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("response code = %d, want 0", resp.Code)
	}
	if resp.Data != nil {
		t.Fatal("operate success response must not include data")
	}
	if fake.operateN != 1 {
		t.Fatalf("Operate calls = %d, want 1", fake.operateN)
	}
	if fake.operateOp != "start" {
		t.Fatalf("Operate op = %q, want start", fake.operateOp)
	}
}

func TestNezhaAgentOperateRejectsEmptyBeforeService(t *testing.T) {
	fake := &fakeNezhaAgentService{}
	installFakeNezhaAgentService(t, fake)

	rec := performNezhaHandler(
		http.MethodPost,
		"/api/v1/nezha-agent/operate",
		`{"operation":""}`,
		(&NezhaAgentAPI{}).OperateNezhaAgent,
	)
	if fake.operateN != 0 {
		t.Fatalf("Operate calls = %d, want 0 for empty operation", fake.operateN)
	}
	var resp dto.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code == 0 {
		t.Fatal("empty operation should be rejected")
	}
}

func TestNezhaAgentOperateRejectsUnsupportedBeforeService(t *testing.T) {
	fake := &fakeNezhaAgentService{}
	installFakeNezhaAgentService(t, fake)

	rec := performNezhaHandler(
		http.MethodPost,
		"/api/v1/nezha-agent/operate",
		`{"operation":"reload"}`,
		(&NezhaAgentAPI{}).OperateNezhaAgent,
	)
	if fake.operateN != 0 {
		t.Fatalf("Operate calls = %d, want 0 for unsupported operation", fake.operateN)
	}
	var resp dto.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code == 0 {
		t.Fatal("unsupported operation should be rejected")
	}
}

func TestNezhaAgentServiceSafeErrorIsHandled(t *testing.T) {
	const secret = "must-not-appear-in-error-response"

	fake := &fakeNezhaAgentService{
		statusErr: buserr.WithDetail(constant.ErrInternalServer, "nezha agent unavailable", nil),
	}
	installFakeNezhaAgentService(t, fake)

	rec := performNezhaHandler(http.MethodGet, "/api/v1/nezha-agent/status", "", (&NezhaAgentAPI{}).GetNezhaAgentStatus)
	body := rec.Body.String()
	assertNoSecretLeak(t, body, secret)

	var resp dto.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code == 0 {
		t.Fatal("service error should not produce success code")
	}
	if resp.Message == "" {
		t.Fatal("service error response missing message")
	}
	assertNoSecretLeak(t, resp.Message, secret)

	// Plain error path (non-BusinessError) also handled safely.
	fake2 := &fakeNezhaAgentService{
		operateErr: errors.New("operate failed safely"),
	}
	installFakeNezhaAgentService(t, fake2)
	rec2 := performNezhaHandler(
		http.MethodPost,
		"/api/v1/nezha-agent/operate",
		`{"operation":"stop"}`,
		(&NezhaAgentAPI{}).OperateNezhaAgent,
	)
	assertNoSecretLeak(t, rec2.Body.String(), secret)
	var resp2 dto.Response
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("unmarshal operate error response: %v", err)
	}
	if resp2.Code == 0 {
		t.Fatal("operate service error should not succeed")
	}
}
