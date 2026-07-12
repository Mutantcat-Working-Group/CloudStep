package alert

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/entity"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
)

// TestEmit_ChannelBuffered 验证 emit 进入 channel + 满缓冲 drop 后只剩 10 条。
func TestEmit_ChannelBuffered(t *testing.T) {
	drain(t)
	for i := 0; i < 15; i++ { // > cap 10, drop 5
		Emit(Event{Id: i, Path: "p", Kind: KindDown, Attempts: i})
	}
	if len(EventCh) != 10 {
		t.Fatalf("channel len=%d want 10", len(EventCh))
	}
	drain(t)
}

// TestProcessEvent_DisabledSkips 告警总开关=0 时, 处理事件不写库(LastAlertAt 保持 NULL)。
func TestProcessEvent_DisabledSkips(t *testing.T) {
	drain(t)
	id := seedAlertUrl(t, "alert-disabled", "http://127.0.0.1:0/none")
	defer cleanupAlertUrl(t, id)

	restore := snapshotAlertConfig(t)
	disableAlert(t) // 总开关关, ding+mail 也关
	processEvent(Event{Id: id, Path: "http://127.0.0.1:0/none", Kind: KindDown, Attempts: 3})
	restore()

	var u entity.Url
	has, _ := dao.PublicEngine.ID(id).Get(&u)
	if !has {
		t.Fatalf("url id=%d missing", id)
	}
	if u.LastAlertAt != nil {
		t.Fatalf("disabled config should not update LastAlertAt, got %v", *u.LastAlertAt)
	}
}

// TestProcessEvent_EnabledDownWrites 总开关=1 + ding 通道开(指向本地测试服务器),
// 处理一条 DOWN 事件应落库: LastAlertAt 非空 / isDown=true / failCount=1,
// 且钉钉测试服务器收到一条 markdown 且含标题/内容。
func TestProcessEvent_EnabledDownWrites(t *testing.T) {
	drain(t)
	id := seedAlertUrl(t, "alert-enabled", "http://127.0.0.1:0/none")
	defer cleanupAlertUrl(t, id)

	srv := newDingRecorder(t)
	defer srv.Close()

	restore := snapshotAlertConfig(t)
	enableAlertDing(t, srv.URL, 600)

	processEvent(Event{Id: id, Path: "http://127.0.0.1:0/none", Kind: KindDown, Attempts: 4})
	restore()

	var u entity.Url
	if _, err := dao.PublicEngine.ID(id).Get(&u); err != nil {
		t.Fatalf("read url: %v", err)
	}
	if u.LastAlertAt == nil {
		t.Fatalf("LastAlertAt should be set after enabled DOWN event")
	}
	if !u.LastAlertIsDown {
		t.Fatalf("LastAlertIsDown=true expected after DOWN event")
	}
	if u.LastAlertFailCount != 1 {
		t.Fatalf("failCount=%d want 1", u.LastAlertFailCount)
	}

	got := srv.body()
	if got.Msgtype != "markdown" {
		t.Fatalf("ding msgtype=%q want markdown", got.Msgtype)
	}
	if got.Markdown.Title == "" || got.Markdown.Text == "" {
		t.Fatalf("ding markdown empty: %+v", got.Markdown)
	}
}

// TestProcessEvent_DebounceSkips 同一 URL+kind 在窗口内再次 emit, 防抖生效不漏库。
func TestProcessEvent_DebounceSkips(t *testing.T) {
	drain(t)
	id := seedAlertUrl(t, "alert-debounce", "http://127.0.0.1:0/none")
	defer cleanupAlertUrl(t, id)

	srv := newDingRecorder(t)
	defer srv.Close()

	restore := snapshotAlertConfig(t)
	enableAlertDing(t, srv.URL, 600)

	processEvent(Event{Id: id, Path: "p", Kind: KindDown, Attempts: 1}) // 落库 failCount=1
	processEvent(Event{Id: id, Path: "p", Kind: KindDown, Attempts: 2}) // 窗口内同 kind → 防抖跳过, 不发
	restore()

	var u entity.Url
	if _, err := dao.PublicEngine.ID(id).Get(&u); err != nil {
		t.Fatalf("read: %v", err)
	}
	if u.LastAlertFailCount != 1 {
		t.Fatalf("debounce window should suppress second DOWN: failCount=%d want 1", u.LastAlertFailCount)
	}
	if !u.LastAlertIsDown || u.LastAlertAt == nil {
		t.Fatalf("expected first alert recorded: isDown=%v at=%v", u.LastAlertIsDown, u.LastAlertAt)
	}
	// 钉钉服务器只应收到 1 条(第二次被防抖拦截)。
	if srv.count() != 1 {
		t.Fatalf("ding should fire once, got %d", srv.count())
	}
}

// ---- 测试服务器与配置 helpers ----

// dingRecorder 是捕获钉钉请求体的 httptest 服务器。
type dingRecorder struct {
	*httptest.Server
	mu    sync.Mutex
	calls []dingPayload
}

type dingPayload struct {
	Msgtype  string `json:"msgtype"`
	Markdown struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	} `json:"markdown"`
}

func newDingRecorder(t *testing.T) *dingRecorder {
	t.Helper()
	dr := &dingRecorder{}
	dr.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var p dingPayload
		_ = json.Unmarshal(raw, &p)
		dr.mu.Lock()
		dr.calls = append(dr.calls, p)
		dr.mu.Unlock()
		w.WriteHeader(200)
	}))
	return dr
}

func (d *dingRecorder) body() dingPayload {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.calls) == 0 {
		return dingPayload{}
	}
	return d.calls[len(d.calls)-1]
}

func (d *dingRecorder) count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.calls)
}

// enableAlertDing 开启总开关+钉钉通道(指向 webhook), 设 debounce 秒。
func enableAlertDing(t *testing.T, webhook string, debounce int) {
	t.Helper()
	if !dao.UpdateAlertConfig(entity.SystemConfig{
		AlertEnabled:     true,
		AlertDingEnabled:  true,
		AlertDingWebhook: webhook,
		AlertMailEnabled: false,
		AlertDebounceSec: debounce,
	}) {
		t.Fatalf("enable alert+ding config")
	}
}

// disableAlert 总开关 + 通道全关(用于 DisabledSkips 配置状态)。
// 走 UpdateAlertConfig 的零值保留写法会错误地保留前一个测试留下的真值,
// 这里直写 DB 把三个 bool 强制置 0。
func disableAlert(t *testing.T) {
	t.Helper()
	if _, err := dao.PublicEngine.ID(1).Cols("alert_enabled", "alert_ding_enabled", "alert_mail_enabled").
		Update(&entity.SystemConfig{AlertEnabled: false, AlertDingEnabled: false, AlertMailEnabled: false}); err != nil {
		t.Fatalf("disable alert config: %v", err)
	}
}

// snapshotAlertConfig / restore 在当前测试前后保存并回写完整告警配置。
func snapshotAlertConfig(t *testing.T) func() {
	t.Helper()
	orig := dao.GetSystemConfig()
	return func() {
		_ = dao.UpdateAlertConfig(orig)
	}
}

// drain 清空全局 EventCh(仅测试用)。
func drain(t *testing.T) {
	t.Helper()
	for {
		select {
		case <-EventCh:
			continue
		default:
			return
		}
	}
}

func seedAlertUrl(t *testing.T, parent, path string) int {
	t.Helper()
	var maxId interface{}
	_, _ = dao.PublicEngine.Table("url").Select("IFNULL(MAX(id),0)").Get(&maxId)
	newId := 1
	switch v := maxId.(type) {
	case int64:
		newId = int(v) + 1
	case []byte:
		n, _ := strconv.Atoi(string(v))
		newId = n + 1
	}
	_, err := dao.PublicEngine.Insert(&entity.Url{Id: newId, Parent: parent, Path: path, Alive: true, Retry: 0})
	if err != nil {
		t.Fatalf("seed insert: %v", err)
	}
	collection.MWorkCllection.Lock()
	collection.WorkCllection[parent] = append(collection.WorkCllection[parent], entity.Url{Id: newId, Parent: parent, Path: path})
	collection.MWorkCllection.Unlock()
	return newId
}

func cleanupAlertUrl(t *testing.T, id int) {
	t.Helper()
	dao.PublicEngine.Where("id = ?", id).Delete(&entity.Url{})
	collection.MWorkCllection.Lock()
	for coll, urls := range collection.WorkCllection {
		filtered := urls[:0]
		for _, u := range urls {
			if u.Id != id {
				filtered = append(filtered, u)
			}
		}
		collection.WorkCllection[coll] = filtered
	}
	collection.MWorkCllection.Unlock()
}
