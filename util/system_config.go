package util

import (
	"com.mutantcat.cloud_step/entity"
	"sync"
)

// sysCfg 是 entity.SystemConfig 的内存镜像,热路径读取绕过 DB。
var (
	sysCfg   = entity.SystemConfig{AllowIntranetProxy: true}
	sysCfgMu sync.RWMutex
)

// SetSystemConfigFromDao 由 InitSystemConfig 与 updateSysConfig 成功后调用,
// 把最新 DB 值整块替换进镜像。
func SetSystemConfigFromDao(c entity.SystemConfig) {
	sysCfgMu.Lock()
	defer sysCfgMu.Unlock()
	sysCfg = c
}

// GetSysConfigMirror 只读返回当前镜像(RLock 保护)。热路径用。
func GetSysConfigMirror() entity.SystemConfig {
	sysCfgMu.RLock()
	defer sysCfgMu.RUnlock()
	return sysCfg
}

// AllowIntranet 是否允许代理到内网(向后兼容入口)。
func AllowIntranet() bool {
	return GetSysConfigMirror().AllowIntranetProxy
}
