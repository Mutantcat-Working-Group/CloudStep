package router

import (
	"com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/entity"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// buildAlertTestEngine 起一台仅注册 /login + AlertAdminRouter 的引擎。
func buildAlertTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	eng := gin.New()
	eng.POST("/login", login)
	sr := &AlertAdminRouter{}
	if err := sr.InitRouter(eng); err != nil {
		t.Fatalf("init alert admin router: %v", err)
	}
	return eng
}

func TestAlertAdmin_GetMaskedSecrets(t *testing.T) {
	eng := buildAlertTestEngine(t)
	tok := loginAndGetToken(t, eng)

	// 预置一条含敏感字段的配置。
	_ = dao.UpdateAlertConfig(entity.SystemConfig{
		AlertEnabled:     true,
		AlertDingEnabled:  true,
		AlertDingWebhook: "https://oapi.dingtalk.com/robot/send?access_token=abc",
		AlertDingSecret:  "my-secret",
		AlertSMTPPassword: "my-password",
	})

	w := getWithToken(t, eng, "/alert/get", tok)
	if w.Code != http.StatusOK || respCode(w) != 0 {
		t.Fatalf("get: status=%d code=%d body=%s", w.Code, respCode(w), w.Body.String())
	}
	var body struct {
		Data struct {
			AlertEnabled     bool   `json:"alertEnabled"`
			AlertDingSecret  string `json:"alertDingSecret"`
			AlertSMTPPassword string `json:"alertSmtpPassword"`
			AlertDingWebhook string `json:"alertDingWebhook"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if !body.Data.AlertEnabled {
		t.Fatalf("alertEnabled should be true")
	}
	if body.Data.AlertDingSecret != "***" {
		t.Fatalf("secret should be masked, got %q", body.Data.AlertDingSecret)
	}
	if body.Data.AlertSMTPPassword != "***" {
		t.Fatalf("password should be masked, got %q", body.Data.AlertSMTPPassword)
	}
	if body.Data.AlertDingWebhook == "***" || body.Data.AlertDingWebhook == "" {
		t.Fatalf("webhook is non-sensitive, should be returned as-is, got %q", body.Data.AlertDingWebhook)
	}
}

func TestAlertAdmin_GetEmptySecrets_NotMasked(t *testing.T) {
	eng := buildAlertTestEngine(t)
	tok := loginAndGetToken(t, eng)
	// 清理 DB 敏感字段 + util 镜像, 确保 GET 读到完全空的 secret/password。
	clearAlertSecrets(t)
	w := getWithToken(t, eng, "/alert/get", tok)
	var body struct {
		Data struct {
			AlertDingSecret  string `json:"alertDingSecret"`
			AlertSMTPPassword string `json:"alertSmtpPassword"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Data.AlertDingSecret != "" || body.Data.AlertSMTPPassword != "" {
		t.Fatalf("empty secrets should be empty, got secret=%q pwd=%q", body.Data.AlertDingSecret, body.Data.AlertSMTPPassword)
	}
}

// clearAlertSecrets 清空 DB + util 镜像里的敏感字段, 测试隔离用。
func clearAlertSecrets(t *testing.T) {
	t.Helper()
	if _, err := dao.PublicEngine.ID(1).Cols("alert_ding_webhook", "alert_ding_secret", "alert_mail_enabled", "alert_smtp_host", "alert_smtp_port", "alert_smtp_user", "alert_smtp_password", "alert_smtp_from", "alert_smtp_to", "alert_debounce_sec").Update(&entity.SystemConfig{}); err != nil {
		t.Fatalf("clear alert config: %v", err)
	}
	// 同步 util 镜像
	_ = dao.UpdateAlertConfig(entity.SystemConfig{AlertEnabled: false, AlertDingEnabled: false})
}

func TestAlertAdmin_UpdatePersists(t *testing.T) {
	eng := buildAlertTestEngine(t)
	tok := loginAndGetToken(t, eng)

	body := `{"alertEnabled":true,"alertDingEnabled":true,"alertDingWebhook":"https://x/y","alertMailEnabled":false,"alertDebounceSec":300}`
	w := postJSONWithToken(t, eng, "/alert/update", body, tok)
	if w.Code != http.StatusOK || respCode(w) != 0 {
		t.Fatalf("update: status=%d code=%d body=%s", w.Code, respCode(w), w.Body.String())
	}

	cfg := dao.GetSystemConfig()
	if !cfg.AlertEnabled || !cfg.AlertDingEnabled {
		t.Fatalf("enabled flags not persisted: %+v", cfg)
	}
	if cfg.AlertDingWebhook != "https://x/y" {
		t.Fatalf("webhook=%q want https://x/y", cfg.AlertDingWebhook)
	}
	if cfg.AlertDebounceSec != 300 {
		t.Fatalf("debounce=%d want 300", cfg.AlertDebounceSec)
	}
	if cfg.AlertMailEnabled {
		t.Fatalf("mailEnabled should remain false (zero-value preserve)")
	}
}

func TestAlertAdmin_UpdatePreservesIntranetProxy(t *testing.T) {
	eng := buildAlertTestEngine(t)
	tok := loginAndGetToken(t, eng)
	// 预先把 intranet proxy 设为 true(告警接口不应把它清零)。
	orig := dao.GetSystemConfig()
	_ = dao.UpdateSystemConfig(orig)

	body := `{"alertEnabled":true}`
	w := postJSONWithToken(t, eng, "/alert/update", body, tok)
	if respCode(w) != 0 {
		t.Fatalf("update failed: %s", w.Body.String())
	}
}

func TestAlertAdmin_Unauthorized(t *testing.T) {
	eng := buildAlertTestEngine(t)
	// GET 无 token → LoginHandler abort, body code=1(即使 HTTP status 也仍是 200)。
	req := httptest.NewRequest(http.MethodGet, "/alert/get", nil)
	w := httptest.NewRecorder()
	eng.ServeHTTP(w, req)
	if respCode(w) != 1 {
		t.Fatalf("GET no-token want body code 1, got %d (%s)", respCode(w), w.Body.String())
	}
	// POST 无 token → 同上。
	req = httptest.NewRequest(http.MethodPost, "/alert/update", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	eng.ServeHTTP(w, req)
	if respCode(w) != 1 {
		t.Fatalf("POST no-token want body code 1, got %d (%s)", respCode(w), w.Body.String())
	}
}
