package router

import (
	"com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/entity"
	"com.mutantcat.cloud_step/util"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// insertSelfWay 向 self_help_mode 表插入一条 way 记录。
func insertSelfWay(t *testing.T, way string) {
	t.Helper()
	if _, err := dao.PublicEngine.Insert(&entity.SelfHelp{Name: way, Way: way, Point: "x"}); err != nil {
		t.Fatalf("insert self way %q: %v", way, err)
	}
}

// setSelfSalt 按 way 把 self_help_mode.salt 写为指定值(直接改 DB,与 SaltInjector 读路径一致)。
func setSelfSalt(t *testing.T, way string, salt string) {
	t.Helper()
	if _, err := dao.PublicEngine.Table(&entity.SelfHelp{}).
		Where("way = ?", way).Update(map[string]interface{}{"salt": salt}); err != nil {
		t.Fatalf("set salt for %q: %v", way, err)
	}
}

func newSelfHelpTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	eng := gin.New()
	eng.Any("/self", selfhelp)
	eng.Any("/self/:way", selfhelp)
	eng.Any("/agent/*name", proxy)
	return eng
}

func doSelf(t *testing.T, eng *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	eng.ServeHTTP(w, req)
	return w
}

func TestSaltInjector_Table(t *testing.T) {
	eng := newSelfHelpTestEngine(t)

	openWay := "open_" + util.RandToken(6)
	setWay := "set_" + util.RandToken(6)
	insertSelfWay(t, openWay)
	insertSelfWay(t, setWay)
	setSelfSalt(t, setWay, "the-key")

	// 注: self 表单走 query 参数(?way=...)而不是路径段(/self/<way>),
	// 前者由 util.GetwayParam 提前 short-circuit 避免 gin 在 BindJSON 空 body 时
	// 预写 400 状态码(见 slist_router.go 的同类说明)。这也是 spec 的主要示例形态。
	validHMAC := util.HMACSHA256Hex("the-key", "")

	cases := []struct {
		name       string
		path       string
		wantHalted bool
	}{
		{"empty-salt-no-param-pass", "/self?way=" + openWay, false},
		{"empty-salt-any-param-pass", "/self?way=" + openWay + "&salt=whatever", false},
		{"salt-set-no-param-403", "/self?way=" + setWay, true},
		{"salt-set-wrong-param-403", "/self?way=" + setWay + "&salt=deadbeef", true},
		{"salt-set-correct-pass", "/self?way=" + setWay + "&salt=" + strings.ToLower(validHMAC), false},
		{"salt-set-correct-upper-pass", "/self?way=" + setWay + "&salt=" + strings.ToUpper(validHMAC), false},
		{"unknown-way-pass", "/self?way=" + "nope_" + util.RandToken(6), false},
	}
	for _, tc := range cases {
		w := doSelf(t, eng, tc.path)
		if tc.wantHalted {
			if w.Code != http.StatusForbidden {
				t.Fatalf("%s: want HTTP 403, got %d body=%s", tc.name, w.Code, w.Body.String())
			}
			if respCode(w) != 1 {
				t.Fatalf("%s: want body code 1, got %d (%s)", tc.name, respCode(w), w.Body.String())
			}
			continue
		}
		if w.Code == http.StatusForbidden {
			t.Fatalf("%s: unexpected halt(403) body=%s", tc.name, w.Body.String())
		}
	}
}

func TestSaltInjector_AgentPathSigning(t *testing.T) {
	eng := newSelfHelpTestEngine(t)
	way := "agent_" + util.RandToken(6)
	insertSelfWay(t, way)
	setSelfSalt(t, way, "agent-key")
	valid := util.HMACSHA256Hex("agent-key", "seg")

	w := doSelf(t, eng, "/agent/seg?way="+way+"&salt="+strings.ToLower(valid))
	if w.Code == http.StatusForbidden {
		t.Fatalf("agent valid sig halted: body=%s", w.Body.String())
	}

	w = doSelf(t, eng, "/agent/seg?way="+way+"&salt=badhex00")
	if w.Code != http.StatusForbidden {
		t.Fatalf("agent wrong sig want 403, got %d body=%s", w.Code, w.Body.String())
	}
}
