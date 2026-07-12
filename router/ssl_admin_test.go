package router

import (
	"com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/entity"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// newSslAdminTestEngine 起一台仅注册 /login + SslAdminRouter 的引擎。
func newSslAdminTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	eng := gin.New()
	eng.POST("/login", login)
	sr := &SslAdminRouter{}
	if err := sr.InitRouter(eng); err != nil {
		t.Fatalf("init ssl admin router: %v", err)
	}
	return eng
}

// genTempCert 在 dir 下生成一条 ECDSA P-256 自签证书 + PKCS#8 私钥, 返回 (certPath, keyPath)。
// 仅用于测试, 无任何外部依赖。
func genTempCert(t *testing.T, dir string) (certPath, keyPath string) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "cloudstep-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")
	cf, err := os.Create(certPath)
	if err != nil {
		t.Fatalf("create cert pem file: %v", err)
	}
	if err := pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		cf.Close()
		t.Fatalf("pem encode cert: %v", err)
	}
	cf.Close()

	kb, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	kf, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("create key pem file: %v", err)
	}
	if err := pem.Encode(kf, &pem.Block{Type: "PRIVATE KEY", Bytes: kb}); err != nil {
		kf.Close()
		t.Fatalf("pem encode key: %v", err)
	}
	kf.Close()
	return certPath, keyPath
}

// resetSslCols 把 DB + util 镜像里的 SSL 4 列清零, 测试隔离。
func resetSslCols(t *testing.T) {
	t.Helper()
	if _, err := dao.PublicEngine.ID(1).
		Cols("ssl_enabled", "ssl_cert_path", "ssl_key_path", "ssl_port").
		Update(&entity.SystemConfig{}); err != nil {
		t.Fatalf("clear ssl cols: %v", err)
	}
	_ = dao.UpdateSystemConfig(entity.SystemConfig{})
}

func TestSSLAdmin_HappyPath(t *testing.T) {
	eng := newSslAdminTestEngine(t)
	tok := loginAndGetToken(t, eng)
	dir := t.TempDir()
	certPath, keyPath := genTempCert(t, dir)
	resetSslCols(t)

	certEsc := strings.ReplaceAll(certPath, `\`, `\\`)
	keyEsc := strings.ReplaceAll(keyPath, `\`, `\\`)
	payload := `{"enabled":true,"certPath":"` + certEsc + `","keyPath":"` + keyEsc + `","port":19443}`
	w := postJSONWithToken(t, eng, "/sslcerts/update", payload, tok)
	if w.Code != http.StatusOK || respCode(w) != 0 {
		t.Fatalf("update: status=%d code=%d body=%s", w.Code, respCode(w), w.Body.String())
	}

	cfg := dao.GetSystemConfig()
	if !cfg.SslEnabled {
		t.Fatalf("SslEnabled should be true")
	}
	if cfg.SslCertPath != certPath {
		t.Fatalf("certPath=%q want %q", cfg.SslCertPath, certPath)
	}
	if cfg.SslKeyPath != keyPath {
		t.Fatalf("keyPath=%q want %q", cfg.SslKeyPath, keyPath)
	}
	if cfg.SslPort != 19443 {
		t.Fatalf("sslPort=%d want 19443", cfg.SslPort)
	}

	// GET 反射落库值。
	w = getWithToken(t, eng, "/sslcerts", tok)
	var body struct {
		Data struct {
			Enabled  bool   `json:"enabled"`
			CertPath string `json:"certPath"`
			KeyPath  string `json:"keyPath"`
			Port     int    `json:"port"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if !body.Data.Enabled || body.Data.CertPath != certPath || body.Data.KeyPath != keyPath || body.Data.Port != 19443 {
		t.Fatalf("GET reflect mismatch: %+v body=%s", body.Data, w.Body.String())
	}
}

func TestSSLAdmin_MissingFile(t *testing.T) {
	eng := newSslAdminTestEngine(t)
	tok := loginAndGetToken(t, eng)
	resetSslCols(t)

	w := postJSONWithToken(t, eng, "/sslcerts/update", `{"enabled":true,"certPath":"/nope/cert.pem","keyPath":"/nope/key.pem","port":19443}`, tok)
	if w.Code != http.StatusOK || respCode(w) != 1 {
		t.Fatalf("missing-file: want code 1, got status=%d code=%d body=%s", w.Code, respCode(w), w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "cert or key file not found") {
		t.Fatalf("missing-file: msg should contain reason, got %s", w.Body.String())
	}
	// 不应落库。
	cfg := dao.GetSystemConfig()
	if cfg.SslEnabled {
		t.Fatalf("SslEnabled should remain false on failed update")
	}
}

func TestSSLAdmin_DisallowedEmpty(t *testing.T) {
	eng := newSslAdminTestEngine(t)
	tok := loginAndGetToken(t, eng)
	resetSslCols(t)

	// enabled=true 但路径空 → code:1。
	w := postJSONWithToken(t, eng, "/sslcerts/update", `{"enabled":true,"certPath":"","keyPath":"","port":19443}`, tok)
	if respCode(w) != 1 {
		t.Fatalf("empty-enabled want code 1, got %d (%s)", respCode(w), w.Body.String())
	}
}

func TestSSLAdmin_DisabledAllowedEmptyPaths(t *testing.T) {
	eng := newSslAdminTestEngine(t)
	tok := loginAndGetToken(t, eng)
	resetSslCols(t)

	// enabled=false 时空路径允许, 不落文件校验。
	w := postJSONWithToken(t, eng, "/sslcerts/update", `{"enabled":false,"certPath":"","keyPath":"","port":9443}`, tok)
	if w.Code != http.StatusOK || respCode(w) != 0 {
		t.Fatalf("disabled: status=%d code=%d body=%s", w.Code, respCode(w), w.Body.String())
	}
	cfg := dao.GetSystemConfig()
	if cfg.SslEnabled {
		t.Fatalf("sslEnabled should be false")
	}
}

func TestSSLAdmin_Unauthenticated(t *testing.T) {
	eng := newSslAdminTestEngine(t)

	req := httptest.NewRequest(http.MethodGet, "/sslcerts", nil)
	w := httptest.NewRecorder()
	eng.ServeHTTP(w, req)
	if respCode(w) != 1 {
		t.Fatalf("GET no-token want body code 1, got %d (%s)", respCode(w), w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/sslcerts/update", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	eng.ServeHTTP(w, req)
	if respCode(w) != 1 {
		t.Fatalf("POST no-token want body code 1, got %d (%s)", respCode(w), w.Body.String())
	}
}
