package collection

import (
	"com.mutantcat.cloud_step/entity"
	"sync"
)

// 所有映射集
var WorkCllection = make(map[string][]entity.Url)
var MWorkCllection sync.Mutex

// 自助模式（指向映射集中的某几项）
var SelfHelpMode = make(map[string]entity.SelfHelp)
var MSelfHelpMode sync.Mutex

// 代理模式（指向映射集中的某几项）
var ProxyMode = make(map[string]entity.Proxy)
var MProxyMode sync.Mutex
