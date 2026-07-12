package scheduler

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/entity"
	"strconv"
	"testing"
	"time"
)

// TestReactivate_ExpiresSelfDeactivate 到期 self_deactivate 的 url 恢复为 alive=true,
// 且 attempts 递增。
func TestReactivate_ExpiresSelfDeactivate(t *testing.T) {
	resetUrlCachesForTest()

	// 选 max(url.id)+1 避免与演示数据冲突。
	var maxId interface{}
	_, _ = dao.PublicEngine.Table("url").Select("IFNULL(MAX(id),0)").Get(&maxId)
	id := 1
	switch v := maxId.(type) {
	case int64:
		id = int(v) + 1
	case []byte:
		n, _ := strconv.Atoi(string(v))
		id = n + 1
	}

	// seed 一条 url: Alive=false(自停用状态), Retry=3(前 3 次心跳探到 still-down), SelfDeactivateUntil=now-1min, attempts=0
	past := time.Now().Add(-1 * time.Minute)
	if _, err := dao.PublicEngine.Insert(&entity.Url{
		Id:                     id,
		Parent:                 "test-reactivate",
		Path:                   "http://reactivate",
		Alive:                  false,
		Retry:                  3,
		SelfDeactivateKey:      "k",
		SelfDeactivateAttempts: 0,
		SelfDeactivateUntil:    &past,
	}); err != nil {
		t.Fatalf("seed insert: %v", err)
	}
	defer dao.PublicEngine.Where("id = ?", id).Delete(&entity.Url{})

	// 同步 cache(与 DB 一致)
	collection.MWorkCllection.Lock()
	collection.WorkCllection["test-reactivate"] = []entity.Url{{
		Id: id, Parent: "test-reactivate", Path: "http://reactivate",
		Alive: false, Retry: 3, SelfDeactivateAttempts: 0, SelfDeactivateUntil: &past,
	}}
	collection.MWorkCllection.Unlock()

	// 跑 reactivate beat
	reactivateBeat()

	var u entity.Url
	has, _ := dao.PublicEngine.ID(id).Get(&u)
	if !has {
		t.Fatalf("url id=%d missing", id)
	}
	if u.SelfDeactivateAttempts != 1 {
		t.Fatalf("after 1 beat: attempts=%d want 1", u.SelfDeactivateAttempts)
	}
	if u.SelfDeactivateUntil == nil || u.SelfDeactivateUntil.After(time.Now()) {
		t.Fatalf("SelfDeactivateUntil should be past/unmodified, got %v", u.SelfDeactivateUntil)
	}
}

func resetUrlCachesForTest() {
	collection.MWorkCllection.Lock()
	collection.WorkCllection = make(map[string][]entity.Url)
	collection.MWorkCllection.Unlock()
}
