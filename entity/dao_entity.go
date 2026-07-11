package entity

type Collection struct {
	Id   int    `xorm:"pk autoincr"`
	Name string `xorm:"varchar(200) notnull"`
}

type Proxy struct {
	Id        int    `xorm:"pk autoincr" json:"id"`
	Name      string `xorm:"varchar(200) notnull" json:"name"`
	Way       string `xorm:"varchar(200) notnull" json:"way"`
	Point     string `xorm:"varchar(200) notnull" json:"point"`
	Mode      string `json:"mode"`      // 当前模式: 轮询、随机
	ProxyMode string `json:"proxyMode"` // 代理模式: 根代理、子代理
	Index     int    `json:"index"`     // 当前映射集的索引(用于轮询模式)
	AliveNum  int    `json:"aliveNum"`  // 存活的路径数
}

type SelfHelp struct {
	Id       int    `xorm:"pk autoincr" json:"id"`
	Name     string `xorm:"varchar(200) notnull" json:"name"`
	Way      string `xorm:"varchar(200) notnull" json:"way"`
	Point    string `xorm:"varchar(200) notnull" json:"point"`
	Mode     string `json:"mode"`     // 当前模式: 轮询、随机
	Index    int    `json:"index"`    // 当前映射集的索引(用于轮询模式)
	AliveNum int    `json:"aliveNum"` // 存活的路径数
}

// 路径
type Url struct {
	Id     int    `xorm:"pk autoincr" json:"id"` // id
	Parent string `xorm:"varchar(200) notnull" json:"parent"`
	Path   string `xorm:"varchar(200) notnull" json:"address"` // 路径
	Alive  bool   `xorm:"notnull" json:"alive"`                // 是否存活
	Retry  int    `xorm:"notnull" json:"retry"`                // 重试次数
}

type User struct {
	Username string `xorm:"varchar(200) notnull"`
	Password string `xorm:"varchar(200) notnull"`
}

// SystemConfig 系统级运行开关表,只有一行(id=1)。
type SystemConfig struct {
	Id int `xorm:"pk" json:"id"`
	// AllowIntranetProxy 是否允许反代理目标为私有/内网/回环/链路本地地址
	AllowIntranetProxy bool `xorm:"notnull" json:"allowIntranetProxy"`
}
