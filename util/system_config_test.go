package util

import (
	"com.mutantcat.cloud_step/entity"
	"testing"
)

func TestGetSysConfigMirror_InitDefault(t *testing.T) {
	// 捕获并后续还原,防测试污染后续 test; 用带锁辅助与 SetSystemConfigFromDao 一致,避免 data race。
	before := getSysCfg()
	defer func() { setSysCfg(before) }()

	SetSystemConfigFromDao(entity.SystemConfig{AllowIntranetProxy: true, SelfDefaultCollectionId: 3, AgentDefaultCollectionId: 5})
	got := GetSysConfigMirror()

	if got.AllowIntranetProxy != true || got.SelfDefaultCollectionId != 3 || got.AgentDefaultCollectionId != 5 {
		t.Fatalf("mirror = %+v; want {AllowIntranetProxy:true Self:3 Agent:5}", got)
	}
	if !AllowIntranet() {
		t.Fatalf("AllowIntranet() = false; want true")
	}
}

// getSysCfg / setSysCfg 带锁读/写镜像, 与 SetSystemConfigFromDao 的锁风格一致,
// 让测试在 -race 下也不会报告 data race。
func getSysCfg() entity.SystemConfig {
	sysCfgMu.RLock()
	defer sysCfgMu.RUnlock()
	return sysCfg
}

func setSysCfg(c entity.SystemConfig) {
	sysCfgMu.Lock()
	defer sysCfgMu.Unlock()
	sysCfg = c
}
