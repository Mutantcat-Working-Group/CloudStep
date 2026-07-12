package util

import (
	"sync"
	"testing"
	"time"
)

// TestCouldLogin_AllowsUnderThreshold 清洁状态 + 过期窗口(=now-60min, 触发重启) → 首次调用初始化窗口放行。
func TestCouldLogin_AllowsUnderThreshold(t *testing.T) {
	setupTestInjector()
	window := time.Now().Add(-1 * time.Hour)
	testState.mu.Lock()
	testState.cfg.MaxFail = 10
	testState.cfg.WindowSec = 180
	testState.cfg.FailCount = 0
	testState.cfg.WindowStart = &window
	testState.mu.Unlock()

	if !CouldLogin() {
		t.Fatalf("CouldLogin should return true on clean state")
	}
}

// TestCouldLogin_LocksAfterThreshold 窗口内 count >= max 时锁定。
func TestCouldLogin_LocksAfterThreshold(t *testing.T) {
	setupTestInjector()
	maxFail := 3
	now := time.Now()
	testState.mu.Lock()
	testState.cfg.MaxFail = maxFail
	testState.cfg.WindowSec = 180
	testState.cfg.FailCount = maxFail
	testState.cfg.WindowStart = &now
	testState.mu.Unlock()

	if CouldLogin() {
		t.Fatalf("CouldLogin should be locked when count(%d) >= max(%d)", testState.cfg.FailCount, maxFail)
	}
}

// TestCouldLogin_ExpiredWindow 超窗口自动重启 → count=0 放行。
func TestCouldLogin_ExpiredWindow(t *testing.T) {
	setupTestInjector()
	testState.mu.Lock()
	testState.cfg.MaxFail = 3
	testState.cfg.WindowSec = 1
	testState.cfg.FailCount = 3
	start := time.Now().Add(-2 * time.Second)
	testState.cfg.WindowStart = &start
	testState.mu.Unlock()

	if !CouldLogin() {
		t.Fatalf("expired window should restart and allow")
	}
}

// TestClearHotNum_ResetsCount 清零 login_fail_count。
func TestClearHotNum_ResetsCount(t *testing.T) {
	setupTestInjector()
	testState.mu.Lock()
	testState.cfg.FailCount = 5
	testState.mu.Unlock()
	ClearHotNum()
	testState.mu.Lock()
	defer testState.mu.Unlock()
	if testState.cfg.FailCount != 0 {
		t.Fatalf("after ClearHotNum: count=%d want 0", testState.cfg.FailCount)
	}
}

// TestRecordLoginPersistsViaRecord failCount 计数值通过 record 写入, 供外部 DAO 持久化。
func TestRecordLoginPersistsViaRecord(t *testing.T) {
	setupTestInjector()
	testState.mu.Lock()
	testState.cfg.FailCount = 6
	testState.mu.Unlock()
	// 直接调 record(外部注入), 验证 write 被调。
	before := testState.writeCalled
	// 触发一次 CouldLogin 确保注入路径被调用
	_ = CouldLogin()
	after := testState.writeCalled
	if after <= before {
		t.Fatalf("record should have been called by CouldLogin invocation")
	}
}

// ---- 测试注入 infra ----

// testState 是 util 测试模拟持久化的 in-memory 状态(避免 import cycle)。
var testState = struct {
	mu          sync.Mutex
	cfg         loginFailSnapshot
	writeCalled int
}{}

// setupTestInjector 注册测试用 read/write 闭包(每个测试调用一次)。
func setupTestInjector() {
	SetLoginTimesInjector(
		func() loginFailSnapshot {
			testState.mu.Lock()
			defer testState.mu.Unlock()
			return testState.cfg
		},
		func(count int, windowStart *time.Time) bool {
			testState.mu.Lock()
			defer testState.mu.Unlock()
			testState.cfg.FailCount = count
			testState.cfg.WindowStart = windowStart
			testState.writeCalled++
			return true
		},
	)
}

var _ = time.Now
