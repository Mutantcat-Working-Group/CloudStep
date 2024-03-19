package entity

type Collection struct {
	Id   int    `xorm:"pk autoincr"`
	Name string `xorm:"varchar(200) notnull"`
}

type Proxy struct {
	Id       int    `xorm:"pk autoincr"`
	Name     string `xorm:"varchar(200) notnull"`
	Way      string `xorm:"varchar(200) notnull"`
	Point    string `xorm:"varchar(200) notnull"`
	Mode     string // 当前模式: 轮询、随机
	Index    int    // 当前映射集的索引(用于轮询模式)
	AliveNum int    // 存活的路径数
}

type SelfHelp struct {
	Id       int    `xorm:"pk autoincr"`
	Name     string `xorm:"varchar(200) notnull"`
	Way      string `xorm:"varchar(200) notnull"`
	Point    string `xorm:"varchar(200) notnull"`
	Mode     string // 当前模式: 轮询、随机
	Index    int    // 当前映射集的索引(用于轮询模式)
	AliveNum int    // 存活的路径数
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
