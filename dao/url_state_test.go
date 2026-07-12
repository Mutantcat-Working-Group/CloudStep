package dao

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/entity"
	"testing"
)

func resetUrlCaches() {
	collection.MWorkCllection.Lock()
	collection.WorkCllection = make(map[string][]entity.Url)
	collection.MWorkCllection.Unlock()

	collection.MSelfHelpMode.Lock()
	collection.SelfHelpMode = make(map[string]entity.SelfHelp)
	collection.MSelfHelpMode.Unlock()

	collection.MProxyMode.Lock()
	collection.ProxyMode = make(map[string]entity.Proxy)
	collection.MProxyMode.Unlock()
}

func insertUrl(id int, coll, path string, alive bool, retry int) {
	u := entity.Url{Id: id, Parent: coll, Path: path, Alive: alive, Retry: retry}
	if _, err := PublicEngine.Insert(&u); err != nil {
		panic(err)
	}
}

func TestUpdateUrlAlive_Enable(t *testing.T) {
	resetUrlCaches()
	const ( id = 1; coll = "coll"; path = "http://u" )
	insertUrl(id, coll, path, false, 3)
	collection.MWorkCllection.Lock()
	collection.WorkCllection[coll] = []entity.Url{{Id: id, Parent: coll, Path: path, Alive: false, Retry: 3}}
	collection.MWorkCllection.Unlock()

	if !UpdateUrlAlive(id, true) {
		t.Fatalf("UpdateUrlAlive(enable) = false; want true")
	}
	var got entity.Url
	has, err := PublicEngine.ID(id).Get(&got)
	if err != nil || !has {
		t.Fatalf("DB fetch after enable: has=%v err=%v", has, err)
	}
	if !got.Alive || got.Retry != 0 {
		t.Fatalf("DB after enable: got Alive=%v Retry=%d; want true,0", got.Alive, got.Retry)
	}
}

func TestUpdateUrlAlive_Disable(t *testing.T) {
	resetUrlCaches()
	const ( id = 2; coll = "coll2"; path = "http://v" )
	insertUrl(id, coll, path, true, 7)
	collection.MWorkCllection.Lock()
	collection.WorkCllection[coll] = []entity.Url{{Id: id, Parent: coll, Path: path, Alive: true, Retry: 7}}
	collection.MWorkCllection.Unlock()

	if !UpdateUrlAlive(id, false) {
		t.Fatalf("UpdateUrlAlive(disable) = false; want true")
	}
	var got entity.Url
	has, err := PublicEngine.ID(id).Get(&got)
	if err != nil || !has {
		t.Fatalf("DB fetch after disable: has=%v err=%v", has, err)
	}
	if got.Alive || got.Retry != 7 {
		t.Fatalf("DB after disable: got Alive=%v Retry=%d; want false,7", got.Alive, got.Retry)
	}
}

func TestUpdateUrlAlive_NotFound(t *testing.T) {
	resetUrlCaches()
	if UpdateUrlAlive(999999, true) || UpdateUrlAlive(999999, false) {
		t.Fatalf("UpdateUrlAlive(nonexistent id) should return false")
	}
}
