package dao

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/entity"
	"testing"
	"time"
)

// maxUrlIdReturn a deterministic fresh id (max existing + 1) so tests
// don't collide across repeated runs; return 1 when table empty.
func freshTestId() int {
	var max struct{ Id int }
	_, err := PublicEngine.SQL("SELECT MAX(id) AS id FROM url").Get(&max)
	if err != nil {
		return 1
	}
	return max.Id + 1
}

func TestGenerateAndSaveUrlKey_FirstReadSeeds(t *testing.T) {
	resetUrlCaches()
	id := freshTestId()
	PublicEngine.ID(id).Delete(&entity.Url{})
	u := entity.Url{Id: id, Parent: "sdcoll", Path: "http://seed", Alive: true, Retry: 0}
	if _, err := PublicEngine.Insert(&u); err != nil {
		t.Fatalf("insert: %v", err)
	}
	defer PublicEngine.ID(id).Delete(&entity.Url{})

	// cache 预置(空 key), 模拟首次读走 seed 路径
	collection.MWorkCllection.Lock()
	collection.WorkCllection["sdcoll"] = []entity.Url{{Id: id, Parent: "sdcoll", Path: "http://seed", Alive: true, Retry: 0}}
	collection.MWorkCllection.Unlock()

	k, err := GenerateAndSaveUrlKey(id)
	if err != nil {
		t.Fatalf("GenerateAndSaveUrlKey: %v", err)
	}
	if len(k) == 0 {
		t.Fatalf("key empty")
	}
	// DB 持久化
	var got entity.Url
	has, _ := PublicEngine.ID(id).Get(&got)
	if !has || got.SelfDeactivateKey != k {
		t.Fatalf("DB key mismatch: has=%v got=%q", has, got.SelfDeactivateKey)
	}
	// cache 同步(锁内扫描)
	collection.MWorkCllection.Lock()
	cached := ""
	for _, urls := range collection.WorkCllection {
		for _, u := range urls {
			if u.Id == id {
				cached = u.SelfDeactivateKey
			}
		}
	}
	collection.MWorkCllection.Unlock()
	if cached != k {
		t.Fatalf("cache key mismatch: %q", cached)
	}
	// 同一 url 再次读取返回同一个 key(不走 re-seed)
	k2, err := GenerateAndSaveUrlKey(id)
	if err != nil || k2 != k {
		t.Fatalf("second read non-idempotent: k2=%q err=%v", k2, err)
	}
}

func TestGetUrl_RoundTrip(t *testing.T) {
	resetUrlCaches()
	id := freshTestId()
	PublicEngine.ID(id).Delete(&entity.Url{})
	u := entity.Url{Id: id, Parent: "rtcoll", Path: "http://rt", Alive: true, Retry: 2}
	if _, err := PublicEngine.Insert(&u); err != nil {
		t.Fatalf("insert: %v", err)
	}
	defer PublicEngine.ID(id).Delete(&entity.Url{})

	got, ok := GetUrl(id)
	if !ok || got.Id != id || got.Path != "http://rt" || got.Alive != true || got.Retry != 2 {
		t.Fatalf("GetUrl mismatch: %+v ok=%v", got, ok)
	}
	if _, ok := GetUrl(9999999); ok {
		t.Fatalf("GetUrl(miss) should be false")
	}
}

func TestSetUrlSelfDeactivate_WritesUntilAndAttempts(t *testing.T) {
	resetUrlCaches()
	id := freshTestId()
	PublicEngine.ID(id).Delete(&entity.Url{})
	u := entity.Url{Id: id, Parent: "scoll", Path: "http://sc", Alive: true, Retry: 0}
	if _, err := PublicEngine.Insert(&u); err != nil {
		t.Fatalf("insert: %v", err)
	}
	defer PublicEngine.ID(id).Delete(&entity.Url{})
	until := time.Now().Add(10 * time.Minute).UTC()

	if !SetUrlSelfDeactivate(id, until, 2) {
		t.Fatalf("SetUrlSelfDeactivate = false")
	}
	var got entity.Url
	has, _ := PublicEngine.ID(id).Get(&got)
	if !has || got.SelfDeactivateAttempts != 2 || got.SelfDeactivateUntil == nil {
		t.Fatalf("DB after set: until=%v attempts=%d", got.SelfDeactivateUntil, got.SelfDeactivateAttempts)
	}
	if got.SelfDeactivateUntil.Sub(until) > time.Second || got.SelfDeactivateUntil.Sub(until) < -time.Second {
		t.Fatalf("until mismatch: got=%v want=%v", got.SelfDeactivateUntil, until)
	}
	// miss → false
	if SetUrlSelfDeactivate(9999999, until, 0) {
		t.Fatalf("SetUrlSelfDeactivate(miss) should be false")
	}
}

func TestClearUrlSelfDeactivate_NullifiesAndResets(t *testing.T) {
	resetUrlCaches()
	id := freshTestId()
	PublicEngine.ID(id).Delete(&entity.Url{})
	u := entity.Url{Id: id, Parent: "ccoll", Path: "http://cc", Alive: true, Retry: 0}
	if _, err := PublicEngine.Insert(&u); err != nil {
		t.Fatalf("insert: %v", err)
	}
	defer PublicEngine.ID(id).Delete(&entity.Url{})
	until := time.Now().UTC()
	if !SetUrlSelfDeactivate(id, until, 4) {
		t.Fatalf("prereq SetUrlSelfDeactivate = false")
	}

	if !ClearUrlSelfDeactivate(id) {
		t.Fatalf("ClearUrlSelfDeactivate = false")
	}
	var got entity.Url
	has, _ := PublicEngine.ID(id).Get(&got)
	if !has || got.SelfDeactivateUntil != nil || got.SelfDeactivateAttempts != 0 {
		t.Fatalf("DB after clear: until=%v attempts=%d; want nil,0", got.SelfDeactivateUntil, got.SelfDeactivateAttempts)
	}
	if ClearUrlSelfDeactivate(9999999) {
		t.Fatalf("ClearUrlSelfDeactivate(miss) should be false")
	}
}
