package collection

import "sync"

// 路径
type Path struct {
	Parent string // 属于的映射集
	Path   string // 自身路径
	Alive  bool   // 是否存活
	Retry  int    // 重试次数
}

type NowWorkCllection struct {
	WorkCllection string // 当前映射集地址
	Mode          string // 当前模式: 轮询、随机
	Index         int    // 当前映射集的索引(用于轮询模式)
	AliveNum      int    // 存活的路径数
}

// 所有映射集
var WorkCllection = make(map[string][]Path)
var mWorkCllection sync.Mutex

// 自助模式（指向映射集中的某几项）
var SelfHelpMode = make(map[string]NowWorkCllection)
var mSelfHelpMode sync.Mutex

// 代理模式（指向映射集中的某几项）
var ProxyMode = make(map[string]NowWorkCllection)
var mProxyMode sync.Mutex
