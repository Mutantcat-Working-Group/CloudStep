package dao

import (
	"com.mutantcat.cloud_step/entity"
	"testing"
)

// TestUpdateUrlRetry_Increment 心跳功能 TDD 测试。
func TestUpdateUrlRetry_Increment(t *testing.T) {
	resetUrlCaches()

	// 选 max(url.id)+1 避免与演示数据冲突。
	var maxId int64
	_, _ = PublicEngine.Table("url").Select("IFNULL(MAX(id),0)").Get(&maxId)
	id := int(maxId) + 1

	if _, err := PublicEngine.Insert(&entity.Url{Id: id, Parent: "test-coll", Path: "probe://hb", Alive: true, Retry: 0}); err != nil {
		t.Fatalf("insert seed url: %v", err)
	}
	defer PublicEngine.Where("id = ?", id).Delete(&entity.Url{})

	// 1) retry=0 → 累加到 3
	if !UpdateUrlRetry(id, 3) {
		t.Fatalf("UpdateUrlRetry(id, 3) = false; want true")
	}
	got := fetchUrl(t, id)
	if got.Retry != 3 {
		t.Fatalf("db retry after inc = %d; want 3", got.Retry)
	}

	// 2) retry=3 → 清零
	if !UpdateUrlRetry(id, 0) {
		t.Fatalf("UpdateUrlRetry(id, 0) = false; want true")
	}
	got = fetchUrl(t, id)
	if got.Retry != 0 {
		t.Fatalf("db retry after reset = %d; want 0", got.Retry)
	}

	// 3) already equal should be no-op success (idempotent)
	if !UpdateUrlRetry(id, 0) {
		t.Fatalf("UpdateUrlRetry(id, 0) idempotency = false; want true")
	}

	// 4) 一条不存在的 id → false
	if UpdateUrlRetry(999999, 1) {
		t.Fatalf("UpdateUrlRetry on missing id = true; want false")
	}
}

func fetchUrl(t *testing.T, id int) entity.Url {
	t.Helper()
	var u entity.Url
	has, err := PublicEngine.ID(id).Get(&u)
	if err != nil || !has {
		t.Fatalf("db read id=%d: has=%v err=%v", id, has, err)
	}
	return u
}
