package router

import (
	"com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/entity"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newSelfTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	eng := gin.New()
	// 注册全部 handler; 管理端点直接接入(绕开 LoginHandler, 与 spec §4 一致测试).
	eng.POST("/self-deactivate", selfDeactivateHandler)
	eng.POST("/self-activate", selfActivateHandler)
	eng.GET("/self-deactivate/key", getKeyHandler)
	eng.POST("/self-deactivate/key/rotate", rotateKeyHandler)
	return eng
}

func freshTestId() int {
	var max struct{ Id int }
	_, err := dao.PublicEngine.SQL("SELECT MAX(id) AS id FROM url").Get(&max)
	if err != nil {
		return 1
	}
	return max.Id + 1
}

func insertUrlForHandler(id int) {
	dao.PublicEngine.ID(id).Delete(&entity.Url{})
	u := entity.Url{Id: id, Parent: "rhcoll", Path: "http://rh", Alive: true, Retry: 0}
	if _, err := dao.PublicEngine.Insert(&u); err != nil {
		panic(err)
	}
}

// 预置一条 admin-disabled url(alive=false, until=NULL), key 已 seed.
func insertAdminDisabledUrl(t *testing.T) (int, string) {
	t.Helper()
	id := freshTestId()
	insertUrlForHandler(id)
	if !dao.UpdateUrlAlive(id, false) {
		t.Fatalf("prereq admin-disable = false")
	}
	k, err := dao.RotateUrlKey(id)
	if err != nil {
		t.Fatalf("prereq rotate: %v", err)
	}
	// seed 同步 cache(走 heartbeat 不设 cache, handler 借助 GetUrl/)
	collection.MWorkCllection.Lock()
	collection.WorkCllection["rhcoll"] = append(collection.WorkCllection["rhcoll"], entity.Url{Id: id, Parent: "rhcoll", Path: "http://rh", Alive: false, SelfDeactivateKey: k, Retry: 0})
	collection.MWorkCllection.Unlock()
	return id, k
}

func seedKey(t *testing.T, id int) string {
	t.Helper()
	k, err := dao.RotateUrlKey(id)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	return k
}

func postJSON(t *testing.T, eng *gin.Engine, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	eng.ServeHTTP(w, req)
	return w
}

func respCode(w *httptest.ResponseRecorder) int {
	var m struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	return m.Code
}

func TestSelfDeactivate_HappyPath(t *testing.T) {
	eng := newSelfTestEngine(t)
	id := freshTestId()
	insertUrlForHandler(id)
	k := seedKey(t, id)

	w := postJSON(t, eng, "/self-deactivate", `{"id":`+itoa(id)+`,"key":"`+k+`","durationSec":60}`)
	if w.Code != http.StatusOK || respCode(w) != 0 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	// DB alive=false + until within ~60s
	u, ok := dao.GetUrl(id)
	if !ok || u.Alive {
		t.Fatalf("DB alive should be false: %+v ok=%v", u, ok)
	}
	if u.SelfDeactivateUntil == nil {
		t.Fatalf("DB until nil")
	}
	d := u.SelfDeactivateUntil.Sub(time.Now())
	if d < 50*time.Second || d > 61*time.Second {
		t.Fatalf("until ~60s out of range: %v", d)
	}
}

func TestSelfDeactivate_KeyMismatch(t *testing.T) {
	eng := newSelfTestEngine(t)
	id := freshTestId()
	insertUrlForHandler(id)

	w := postJSON(t, eng, "/self-deactivate", `{"id":`+itoa(id)+`,"key":"wrongkey","durationSec":60}`)
	if w.Code != http.StatusOK || respCode(w) != 1 {
		t.Fatalf("want code:1 resp, body=%s", w.Body.String())
	}
}

func TestSelfDeactivate_DurationOutOfRange(t *testing.T) {
	eng := newSelfTestEngine(t)
	id := freshTestId()
	insertUrlForHandler(id)
	k := seedKey(t, id)

	wZero := postJSON(t, eng, "/self-deactivate", `{"id":`+itoa(id)+`,"key":"`+k+`","durationSec":0}`)
	if respCode(wZero) != 1 {
		t.Fatalf("duration=0 should code:1, body=%s", wZero.Body.String())
	}
	wBig := postJSON(t, eng, "/self-deactivate", `{"id":`+itoa(id)+`,"key":"`+k+`","durationSec":8000}`)
	if respCode(wBig) != 1 {
		t.Fatalf("duration>7200 should code:1, body=%s", wBig.Body.String())
	}
}

func TestSelfDeactivate_AdminDisabled_403(t *testing.T) {
	eng := newSelfTestEngine(t)
	id, k := insertAdminDisabledUrl(t)

	w := postJSON(t, eng, "/self-deactivate", `{"id":`+itoa(id)+`,"key":"`+k+`","durationSec":60}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d body=%s", w.Code, w.Body.String())
	}
	if respCode(w) != 1 {
		t.Fatalf("403 should carry code:1, body=%s", w.Body.String())
	}
}

func TestSelfActivate_HappyPath(t *testing.T) {
	eng := newSelfTestEngine(t)
	id := freshTestId()
	insertUrlForHandler(id)
	k := seedKey(t, id)
	// self-deactivate 再 activate
	w := postJSON(t, eng, "/self-deactivate", `{"id":`+itoa(id)+`,"key":"`+k+`","durationSec":60}`)
	if respCode(w) != 0 {
		t.Fatalf("prereq deactivate failed: %s", w.Body.String())
	}
	act := postJSON(t, eng, "/self-activate", `{"id":`+itoa(id)+`,"key":"`+k+`"}`)
	if act.Code != http.StatusOK || respCode(act) != 0 {
		t.Fatalf("self-activate fail status=%d body=%s", act.Code, act.Body.String())
	}
	u, _ := dao.GetUrl(id)
	if !u.Alive || u.SelfDeactivateUntil != nil {
		t.Fatalf("after activate alive until nil: %+v", u)
	}
}

func TestSelfActivate_NotSelfDeactivated_400(t *testing.T) {
	eng := newSelfTestEngine(t)
	id := freshTestId()
	insertUrlForHandler(id)
	k := seedKey(t, id)

	w := postJSON(t, eng, "/self-activate", `{"id":`+itoa(id)+`,"key":"`+k+`"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", w.Code, w.Body.String())
	}
	if respCode(w) != 1 {
		t.Fatalf("400 should carry code:1, body=%s", w.Body.String())
	}
}

func TestSelfActivate_AdminDisabled_400(t *testing.T) {
	eng := newSelfTestEngine(t)
	id, k := insertAdminDisabledUrl(t)

	// admin-disabled 且 until==nil → 走 SelfDeactivateUntil==nil 路径 → 400
	w := postJSON(t, eng, "/self-activate", `{"id":`+itoa(id)+`,"key":"`+k+`"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("admin-disabled self-activate want 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestGetKey_NotFound_404(t *testing.T) {
	eng := newSelfTestEngine(t)
	req := httptest.NewRequest(http.MethodGet, "/self-deactivate/key?id=9999999", nil)
	w := httptest.NewRecorder()
	eng.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("getKey miss want 404, got %d", w.Code)
	}
}

func TestRotateKey_ReturnsDifferentKey(t *testing.T) {
	eng := newSelfTestEngine(t)
	id := freshTestId()
	insertUrlForHandler(id)
	k1, _ := dao.GenerateAndSaveUrlKey(id)

	req := httptest.NewRequest(http.MethodPost, "/self-deactivate/key/rotate?id="+itoa(id), nil)
	w := httptest.NewRecorder()
	eng.ServeHTTP(w, req)
	if w.Code != http.StatusOK || respCode(w) != 0 {
		t.Fatalf("rotate fail status=%d body=%s", w.Code, w.Body.String())
	}
	var m struct {
		Data struct {
			Key string `json:"key"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	if m.Data.Key == "" || m.Data.Key == k1 {
		t.Fatalf("rotate should return new key, old=%q new=%q", k1, m.Data.Key)
	}
	u, _ := dao.GetUrl(id)
	if u.SelfDeactivateKey != m.Data.Key {
		t.Fatalf("DB key should reflect rotated: %q vs %q", u.SelfDeactivateKey, m.Data.Key)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
