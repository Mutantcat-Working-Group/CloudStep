// 心跳 sweep 端到端测试:
//   TestBeat_HealthyUrlResetsRetry — 探活成功, retry 清零。
//   TestBeat_FailingUrlDownsAfterThreshold — 连续 failThreshold 次失败 → Alive=false。
// 直接调 beat([]entity.Url) 幂等函数, 不触 production URL 集。

package scheduler

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/entity"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

var seededId int

func TestBeat_HealthyUrlResetsRetry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	host, portStr := u.Hostname(), u.Port()
	if host == "" || portStr == "" {
		t.Fatalf("cannot parse test server URL %q", srv.URL)
	}
	_ = portStr

	seedUrl(t, "hb-ok", "http://"+host+":"+portStr, 2)
	beat([]entity.Url{{Id: seededId, Parent: "hb-ok", Path: srv.URL, Alive: true, Retry: 2}})

	got := fetchSeeded(t)
	if got.Retry != 0 {
		t.Fatalf("after healthy beat, retry = %d; want 0", got.Retry)
	}
	if !got.Alive {
		t.Fatalf("healthy beat must not flip Alive to false")
	}
}

func TestBeat_FailingUrlDownsAfterThreshold(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	u, _ := url.Parse(srv.URL)
	host, portStr := u.Hostname(), u.Port()
	srv.Close() // 立刻关, 探活必 timeout

	seedUrl(t, "hb-fail", "http://"+host+":"+portStr, 0)
	path := "http://" + host + ":" + portStr
	for i := 0; i < failThreshold; i++ {
		beat([]entity.Url{{Id: seededId, Parent: "hb-fail", Path: path, Alive: true, Retry: 0}})
	}
	got := fetchSeeded(t)
	if got.Alive {
		t.Fatalf("after %d failing beats, url should be down; still alive", failThreshold)
	}
	if got.Retry != failThreshold {
		t.Fatalf("expected retry = %d; got %d", failThreshold, got.Retry)
	}
}

// seedUrl 分配新 id (max(url.id)+1), 插一条测试 URL + cache, cleanup 时一并删。
func seedUrl(t *testing.T, parent, path string, retry int) {
	t.Helper()
	var maxId interface{}
	_, _ = dao.PublicEngine.Table("url").Select("IFNULL(MAX(id),0)").Get(&maxId)
	switch v := maxId.(type) {
	case int64:
		seededId = int(v) + 1
	case []byte:
		n, _ := strconv.Atoi(string(v))
		seededId = n + 1
	default:
		seededId = 1
	}
	if _, err := dao.PublicEngine.Insert(&entity.Url{Id: seededId, Parent: parent, Path: path, Alive: true, Retry: retry}); err != nil {
		t.Fatalf("seed insert: %v", err)
	}
	// 同步 cache
	collection.MWorkCllection.Lock()
	collection.WorkCllection[parent] = append(collection.WorkCllection[parent], entity.Url{Id: seededId, Parent: parent, Path: path, Alive: true, Retry: retry})
	collection.MWorkCllection.Unlock()

	t.Cleanup(func() {
		dao.PublicEngine.Where("id = ?", seededId).Delete(&entity.Url{})
		collection.MWorkCllection.Lock()
		for coll, urls := range collection.WorkCllection {
			filtered := urls[:0]
			for _, u := range urls {
				if u.Id != seededId {
					filtered = append(filtered, u)
				}
			}
			collection.WorkCllection[coll] = filtered
		}
		collection.MWorkCllection.Unlock()
	})
}

func fetchSeeded(t *testing.T) entity.Url {
	t.Helper()
	var u entity.Url
	has, err := dao.PublicEngine.ID(seededId).Get(&u)
	if err != nil || !has {
		t.Fatalf("seed read id=%d: has=%v err=%v", seededId, has, err)
	}
	return u
}
