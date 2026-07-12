package util

import (
	"com.mutantcat.cloud_step/entity"
	"testing"
)

func TestGetSysConfigMirror_InitDefault(t *testing.T) {
	// 捕获并后续还原,防测试污染后续 test
	before := sysCfg
	defer func() { sysCfg = before }()

	SetSystemConfigFromDao(entity.SystemConfig{AllowIntranetProxy: true, SelfDefaultCollectionId: 3, AgentDefaultCollectionId: 5})
	got := GetSysConfigMirror()

	if got.AllowIntranetProxy != true || got.SelfDefaultCollectionId != 3 || got.AgentDefaultCollectionId != 5 {
		t.Fatalf("mirror = %+v; want {AllowIntranetProxy:true Self:3 Agent:5}", got)
	}
	if !AllowIntnet() { // 故意拼写错误 AllowIntnet 触发 red(compile error)
		t.Fatalf("AllowIntranet() = false; want true")
	}
}
