package router

import (
	"com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/entity"
	"com.mutantcat.cloud_step/util"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newSaltAdminTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	eng := gin.New()
	eng.POST("/login", login)
	sr := &SaltAdminRouter{}
	if err := sr.InitRouter(eng); err != nil {
		t.Fatalf("init salt admin router: %v", err)
	}
	return eng
}

// loginAndGetToken 走 /login 拿 token,作为后续请求的 Token header。
func loginAndGetToken(t *testing.T, eng *gin.Engine) string {
	t.Helper()
	w := postJSON(t, eng, "/login", `{"username":"admin96","password":"admin96"}`)
	if w.Code != http.StatusOK || respCode(w) != 0 {
		t.Fatalf("login failed: status=%d body=%s", w.Code, w.Body.String())
	}
	var m struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	if m.Token == "" {
		t.Fatalf("login returned empty token: %s", w.Body.String())
	}
	return m.Token
}

type saltResp struct {
	Data struct {
		Salt string `json:"salt"`
		Mode string `json:"mode"`
	} `json:"data"`
}

func TestSaltAdmin_RoundTrip(t *testing.T) {
	eng := newSaltAdminTestEngine(t)
	tok := loginAndGetToken(t, eng)

	way := "adm_" + util.RandToken(6)
	if _, err := dao.PublicEngine.Insert(&entity.SelfHelp{Name: way, Way: way, Point: "x"}); err != nil {
		t.Fatalf("seed self way: %v", err)
	}

	// GET /salts 首次查看: 应懒生成非空 salt, mode=self。
	w := getWithToken(t, eng, "/salts?way="+way, tok)
	if w.Code != http.StatusOK || respCode(w) != 0 {
		t.Fatalf("getSalt: status=%d body=%s", w.Code, w.Body.String())
	}
	var got saltResp
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Data.Salt == "" || got.Data.Mode != "self" {
		t.Fatalf("getSalt lazy-gen unexpected: %+v body=%s", got, w.Body.String())
	}
	firstSalt := got.Data.Salt

	// POST /salts/rotate: 新 salt 与旧不同。
	w = postJSONWithToken(t, eng, "/salts/rotate", `{"way":"`+way+`","mode":"self"}`, tok)
	if w.Code != http.StatusOK || respCode(w) != 0 {
		t.Fatalf("rotate: status=%d body=%s", w.Code, w.Body.String())
	}
	var rot saltResp
	_ = json.Unmarshal(w.Body.Bytes(), &rot)
	if rot.Data.Salt == "" || rot.Data.Salt == firstSalt {
		t.Fatalf("rotate should yield new salt: old=%q new=%q", firstSalt, rot.Data.Salt)
	}
	newSalt := rot.Data.Salt

	// GET 反射旋转后的值。
	w = getWithToken(t, eng, "/salts?way="+way, tok)
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Data.Salt != newSalt {
		t.Fatalf("get after rotate mismatch: want=%q got=%q", newSalt, got.Data.Salt)
	}

	// POST /salts/verify: 正确 path 签名 → valid=true。
	valid := util.HMACSHA256Hex(newSalt, "hello")
	w = postJSONWithToken(t, eng, "/salts/verify", `{"way":"`+way+`","path":"hello","salt":"`+strings.ToLower(valid)+`"}`, tok)
	if w.Code != http.StatusOK || respCode(w) != 0 {
		t.Fatalf("verify: status=%d body=%s", w.Code, w.Body.String())
	}
	var ver struct {
		Data struct {
			Valid bool `json:"valid"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &ver)
	if !ver.Data.Valid {
		t.Fatalf("verify valid sig should be true: body=%s", w.Body.String())
	}

	// verify: 错误 salt → valid=false。
	w = postJSONWithToken(t, eng, "/salts/verify", `{"way":"`+way+`","path":"hello","salt":"00"}`, tok)
	_ = json.Unmarshal(w.Body.Bytes(), &ver)
	if ver.Data.Valid {
		t.Fatalf("verify wrong sig should be false: body=%s", w.Body.String())
	}
}

func TestSaltAdmin_GetUnknownWay_404(t *testing.T) {
	eng := newSaltAdminTestEngine(t)
	tok := loginAndGetToken(t, eng)

	req := httptest.NewRequest(http.MethodGet, "/salts?way="+util.RandToken(8), nil)
	req.Header.Set("Token", tok)
	w := httptest.NewRecorder()
	eng.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown way want 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestSaltAdmin_Unauthorized(t *testing.T) {
	eng := newSaltAdminTestEngine(t)
	req := httptest.NewRequest(http.MethodGet, "/salts?way=whatever", nil)
	w := httptest.NewRecorder()
	eng.ServeHTTP(w, req)
	if respCode(w) != 1 {
		t.Fatalf("no-token want body code 1, got %d (%s)", respCode(w), w.Body.String())
	}
}

func getWithToken(t *testing.T, eng *gin.Engine, path, tok string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Token", tok)
	w := httptest.NewRecorder()
	eng.ServeHTTP(w, req)
	return w
}

func postJSONWithToken(t *testing.T, eng *gin.Engine, path, body, tok string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", tok)
	w := httptest.NewRecorder()
	eng.ServeHTTP(w, req)
	return w
}
