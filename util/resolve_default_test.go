package util

import "testing"

// fakeResolver 返回 mimicked 的 id→name 映射,用于单元测试不依赖 DB。
func fakeResolver(m map[int]string) func(int) string {
	return func(id int) string { return m[id] }
}

func TestResolveWayCollection(t *testing.T) {
	// 捕获并还原全局 resolver,避免污染其他 test(与 SetSystemConfigFromDao 一致)。
	before := defaultCollectionResolver
	defer func() { defaultCollectionResolver = before }()

	db := map[int]string{1: "演示集合", 12: "动漫"}

	// 未注入 resolver: 所有未配/误配场景一律 miss,不 panic。
	defaultCollectionResolver = nil
	if coll, ok := ResolveWayCollection("explicitWay", 3); coll != "" || ok {
		t.Fatalf("[nil resolver] explicit way should bypass default; got (%q,%v)", coll, ok)
	}
	if coll, ok := ResolveWayCollection("", 0); coll != "" || ok {
		t.Fatalf("[nil resolver] zero id should miss; got (%q,%v)", coll, ok)
	}

	// 注入 fake resolver 后走真实判定逻辑。
	defaultCollectionResolver = fakeResolver(db)

	// way 非空 → 不读 default, 返回 ("", false)
	if coll, ok := ResolveWayCollection("explicitWay", 3); coll != "" || ok {
		t.Fatalf("explicit way should bypass default; got (%q,%v)", coll, ok)
	}
	// defaultId == 0 → ("", false): 未配
	if coll, ok := ResolveWayCollection("", 0); coll != "" || ok {
		t.Fatalf("zero id should return miss; got (%q,%v)", coll, ok)
	}
	// defaultId 指向 miss 的 collection → ("", false)
	if coll, ok := ResolveWayCollection("", 999999); coll != "" || ok {
		t.Fatalf("nonexistent collection should miss; got (%q,%v)", coll, ok)
	}
	// defaultId 指向存在的 collection → (name, true)
	if coll, ok := ResolveWayCollection("", 1); !ok || coll != "演示集合" {
		t.Fatalf("existing default coll id=1: got (%q,%v); want (演示集合,true)", coll, ok)
	}
	if coll, ok := ResolveWayCollection("", 12); !ok || coll != "动漫" {
		t.Fatalf("existing default coll id=12: got (%q,%v); want (动漫,true)", coll, ok)
	}
}
