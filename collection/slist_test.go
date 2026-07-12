package collection

import (
	"com.mutantcat.cloud_step/entity"
	"reflect"
	"testing"
)

// resetCaches clears the three package-level maps in the initModes order so
// tests are hermetic. Tests exercise the cached in-memory maps only.
func resetCaches() {
	MWorkCllection.Lock()
	WorkCllection = make(map[string][]entity.Url)
	MWorkCllection.Unlock()

	MSelfHelpMode.Lock()
	SelfHelpMode = make(map[string]entity.SelfHelp)
	MSelfHelpMode.Unlock()

	MProxyMode.Lock()
	ProxyMode = make(map[string]entity.Proxy)
	MProxyMode.Unlock()
}

func TestGetSelfHelpList_SelfWay(t *testing.T) {
	resetCaches()
	const (
		way  = "alpha-self"
		coll = "alpha-collection"
	)
	urls := []entity.Url{
		{Id: 1, Parent: coll, Path: "http://a", Alive: true, Retry: 0},
		{Id: 2, Parent: coll, Path: "http://b", Alive: false, Retry: 1},
	}
	MWorkCllection.Lock(); WorkCllection[coll] = urls; MWorkCllection.Unlock()
	MSelfHelpMode.Lock(); SelfHelpMode[way] = entity.SelfHelp{Way: way, Point: coll}; MSelfHelpMode.Unlock()

	src, point, got := GetSelfHelpList(way)

	if src != SrcSelf {
		t.Fatalf("src = %v; want SrcSelf", src)
	}
	if point != coll {
		t.Fatalf("point = %q; want %q", point, coll)
	}
	if !reflect.DeepEqual(got, urls) {
		t.Fatalf("urls = %+v; want %+v", got, urls)
	}
}

func TestGetSelfHelpList_ProxyWay(t *testing.T) {
	resetCaches()
	const (
		way  = "beta-proxy"
		coll = "beta-collection"
	)
	urls := []entity.Url{
		{Id: 1, Parent: coll, Path: "http://c", Alive: true, Retry: 0},
	}
	MWorkCllection.Lock(); WorkCllection[coll] = urls; MWorkCllection.Unlock()
	MProxyMode.Lock(); ProxyMode[way] = entity.Proxy{Way: way, Point: coll}; MProxyMode.Unlock()

	src, point, got := GetSelfHelpList(way)

	if src != SrcProxy {
		t.Fatalf("src = %v; want SrcProxy", src)
	}
	if point != coll {
		t.Fatalf("point = %q; want %q", point, coll)
	}
	if !reflect.DeepEqual(got, urls) {
		t.Fatalf("urls = %+v; want %+v", got, urls)
	}
}

func TestGetSelfHelpList_UnknownWay(t *testing.T) {
	resetCaches()

	src, point, got := GetSelfHelpList("no-such-way")

	if src != SrcNone {
		t.Fatalf("src = %v; want SrcNone", src)
	}
	if point != "" {
		t.Fatalf("point = %q; want empty", point)
	}
	if got != nil {
		t.Fatalf("urls = %+v; want nil", got)
	}
}

func TestGetSelfHelpList_EmptyCollection(t *testing.T) {
	resetCaches()
	const way = "gamma"
	MSelfHelpMode.Lock(); SelfHelpMode[way] = entity.SelfHelp{Way: way, Point: "gamma-coll"}; MSelfHelpMode.Unlock()
	// do NOT seed WorkCllection["gamma-coll"]; map miss yields nil slice.

	src, point, got := GetSelfHelpList(way)

	if src != SrcSelf {
		t.Fatalf("src = %v; want SrcSelf", src)
	}
	if point != "gamma-coll" {
		t.Fatalf("point = %q; want gamma-coll", point)
	}
	if got != nil {
		t.Fatalf("urls = %+v; want nil", got)
	}
}

func TestGetSelfHelpList_PrefersSelfOverProxy(t *testing.T) {
	resetCaches()
	const (
		way       = "dup"
		selfColl  = "self-coll"
		proxyColl = "proxy-coll"
	)
	MWorkCllection.Lock()
	WorkCllection[selfColl] = []entity.Url{{Id: 1, Parent: selfColl, Path: "http://s", Alive: true, Retry: 0}}
	WorkCllection[proxyColl] = []entity.Url{{Id: 2, Parent: proxyColl, Path: "http://p", Alive: true, Retry: 0}}
	MWorkCllection.Unlock()
	MSelfHelpMode.Lock(); SelfHelpMode[way] = entity.SelfHelp{Way: way, Point: selfColl}; MSelfHelpMode.Unlock()
	MProxyMode.Lock(); ProxyMode[way] = entity.Proxy{Way: way, Point: proxyColl}; MProxyMode.Unlock()

	src, point, _ := GetSelfHelpList(way)

	if src != SrcSelf {
		t.Fatalf("src = %v; want SrcSelf (self-help must win on collision)", src)
	}
	if point != selfColl {
		t.Fatalf("point = %q; want %q", point, selfColl)
	}
}
