package util

import "sync"

var HotNum = 0
var mHotNum sync.Mutex

// 清空热点数
func ClearHotNum() {
	mHotNum.Lock()
	defer mHotNum.Unlock()
	HotNum = 0
}

// 是否可以登录
func CouldLogin() bool {
	mHotNum.Lock()
	defer mHotNum.Unlock()
	// 当一定时间内登录失败次数超过10次时，不允许登录
	if HotNum >= 10 {
		return false
	}
	HotNum++
	return true
}
