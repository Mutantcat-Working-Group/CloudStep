package collection

import (
	"com.mutantcat.cloud_step/entity"
	"testing"
)

// seedSelfHelp seeds a self-help way -> collection with the given urls and a
// polling Mode so selection is deterministic across runs.
func seedSelfHelp(way, coll string, mode string, urls []entity.Url, index int) {
	MWorkCllection.Lock()
	WorkCllection[coll] = urls
	MWorkCllection.Unlock()

	MSelfHelpMode.Lock()
	SelfHelpMode[way] = entity.SelfHelp{Way: way, Point: coll, Mode: mode, Index: index}
	MSelfHelpMode.Unlock()
}

// seedProxy seeds a proxy way -> collection with the given urls and a
// polling Mode so selection is deterministic across runs.
func seedProxy(way, coll string, mode string, urls []entity.Url, index int) {
	MWorkCllection.Lock()
	WorkCllection[coll] = urls
	MWorkCllection.Unlock()

	MProxyMode.Lock()
	ProxyMode[way] = entity.Proxy{Way: way, Point: coll, Mode: mode, Index: index}
	MProxyMode.Unlock()
}

func TestGetPath_SkipsDeadUrls(t *testing.T) {
	resetCaches()
	const (
		way  = "w"
		coll = "c"
	)
	urls := []entity.Url{
		{Id: 1, Parent: coll, Path: "http://dead", Alive: false, Retry: 3},
		{Id: 2, Parent: coll, Path: "http://live", Alive: true, Retry: 0},
	}
	seedSelfHelp(way, coll, "polling", urls, 0)

	for i := 0; i < 50; i++ {
		got := GetPath(way)
		if got != "http://live" {
			t.Fatalf("iteration %d: GetPath = %q; want http://live (never http://dead)", i, got)
		}
	}
}

func TestGetPath_AllDeadReturnsEmpty(t *testing.T) {
	resetCaches()
	const (
		way  = "w"
		coll = "c"
	)
	urls := []entity.Url{
		{Id: 1, Parent: coll, Path: "http://dead1", Alive: false, Retry: 3},
		{Id: 2, Parent: coll, Path: "http://dead2", Alive: false, Retry: 1},
	}
	seedSelfHelp(way, coll, "polling", urls, 0)

	if got := GetPath(way); got != "" {
		t.Fatalf("GetPath = %q; want empty when all urls are dead", got)
	}
}

func TestGetProxyPath_SkipsDeadUrls(t *testing.T) {
	resetCaches()
	const (
		way  = "wp"
		coll = "cp"
	)
	urls := []entity.Url{
		{Id: 1, Parent: coll, Path: "http://dead", Alive: false, Retry: 3},
		{Id: 2, Parent: coll, Path: "http://live", Alive: true, Retry: 0},
	}
	seedProxy(way, coll, "polling", urls, 0)

	for i := 0; i < 50; i++ {
		got := GetProxyPath(way)
		if got != "http://live" {
			t.Fatalf("iteration %d: GetProxyPath = %q; want http://live (never http://dead)", i, got)
		}
	}
}

func TestGetProxyPath_AllDeadReturnsEmpty(t *testing.T) {
	resetCaches()
	const (
		way  = "wp"
		coll = "cp"
	)
	urls := []entity.Url{
		{Id: 1, Parent: coll, Path: "http://dead1", Alive: false, Retry: 3},
		{Id: 2, Parent: coll, Path: "http://dead2", Alive: false, Retry: 1},
	}
	seedProxy(way, coll, "polling", urls, 0)

	if got := GetProxyPath(way); got != "" {
		t.Fatalf("GetProxyPath = %q; want empty when all urls are dead", got)
	}
}
