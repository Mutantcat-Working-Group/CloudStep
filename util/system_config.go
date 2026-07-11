package util

import "sync"

// systemConfig 保存运行时的系统级开关,作为 DB 的内存镜像以便热更新时读取。
type systemConfig struct {
	// AllowIntranetProxy 是否允许反代理目标为私有/内网/回环/链路本地地址。
	// 部署于公网时建议关闭,默认 true 以保留内网智能家居等场景。
	AllowIntranetProxy bool
}

var (
	sysCfg     = systemConfig{AllowIntranetProxy: true}
	sysCfgMu   sync.RWMutex
)

// SetAllowIntranetProxy 更新反代内网允许开关。
func SetAllowIntranetProxy(allow bool) {
	sysCfgMu.Lock()
	defer sysCfgMu.Unlock()
	sysCfg.AllowIntranetProxy = allow
}

// AllowIntranet 查询当前是否允许代理到内网/私有地址。
func AllowIntranet() bool {
	sysCfgMu.RLock()
	defer sysCfgMu.RUnlock()
	return sysCfg.AllowIntranetProxy
}
