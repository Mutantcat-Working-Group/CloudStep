// 登录限速持久化+账户临时锁定。
//
// 原实现把 HotNum 仅存在内存,重启即重置 + 不继承窗口语义。
// 现将持久化状态落到 system_config 表(login_max_fail/login_fail_window_sec/
// login_fail_count/login_fail_window_start)。
//
// 为防 util → dao 形成 import cycle, 持久化调用由 lifecycle 通过
// SetLoginTimesInjector(...) 注入(仅包级变量, 无锁竞争)。

package util

import (
	"sync"
	"time"
)

// loginTimesInjector 登录限速持久化外部注入点。
// lifecycle/gin_service.go::InitGin 阶段调用 SetLoginTimesInjector 赋。
//
// readCfg: 立即返回 system_config 镜像(读 util.GetSysConfigMirror 的薄封装)。
// record(count, windowStart): 写 DB 并同步镜像(受所属 DAO 封装)。
type loginTimesInjector struct {
	readCfg  func() loginFailSnapshot
	record   func(count int, windowStart *time.Time) bool
}

// loginFailSnapshot 持久化状态的镜像视图(读取方无需依赖 entity 完整定义)。
type loginFailSnapshot struct {
	MaxFail     int
	WindowSec   int
	FailCount   int
	WindowStart *time.Time
}

var (
	ltInst     *loginTimesInjector
	ltInstOnce sync.Once
)

// SetLoginTimesInjector 设置持久化注入(仅一次, lifecycle 冷启动时调用)。
func SetLoginTimesInjector(read func() loginFailSnapshot, write func(count int, windowStart *time.Time) bool) {
	ltInstOnce.Do(func() {
		ltInst = &loginTimesInjector{readCfg: read, record: write}
	})
}

// ClearHotNum 重置登录失败计数(登录成功/管理员手动调)。
func ClearHotNum() {
	if ltInst == nil {
		logSkip("[login] injector not set, skip clear")
		return
	}
	if !ltInst.record(0, nil) {
		logSkip("[login] reset count failed")
	}
}

// CouldLogin 当前请求是否允许登录:
//   - cfg.LoginFailCount >= MaxFail 时在窗口内 → 锁定拒绝。
//   - 超窗口自动清零重启新窗口。
//   - 未锁定 → FailCount 计 1(沿用原逻辑: check 时 pre-increment)。
func CouldLogin() bool {
	if ltInst == nil {
		logSkip("[login] injector not set, allow by default")
		return true
	}

	cfg := ltInst.readCfg()
	now := time.Now()

	if cfg.WindowStart == nil {
		if !ltInst.record(1, &now) {
			return true
		}
		return true
	}

	if now.Sub(*cfg.WindowStart) >= time.Duration(cfg.WindowSec)*time.Second {
		if !ltInst.record(1, &now) {
			return true
		}
		return true
	}

	if cfg.FailCount >= cfg.MaxFail {
		return false
	}

	if !ltInst.record(cfg.FailCount+1, nil) {
		return true
	}
	return true
}

func logSkip(msg string) {
	println(msg)
}

var _ = sync.Once{}
