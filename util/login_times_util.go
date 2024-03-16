package util

import "sync"

var HotNum = 0
var hDcode sync.Mutex

// 清空热点数
func ClearHotNum() {
	hDcode.Lock()
	defer hDcode.Unlock()
	HotNum = 0
}

// 是否可以登录
func CouldLogin() bool {
	hDcode.Lock()
	defer hDcode.Unlock()
	// 当一定时间内登录失败次数超过10次时，不允许登录
	if HotNum >= 10 {
		return false
	}
	HotNum++
	return true
}
