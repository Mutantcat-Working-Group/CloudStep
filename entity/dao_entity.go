package entity

import "time"

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
	Salt      string `xorm:"varchar(200) notnull default('')" json:"salt"` // 请求层 HMAC 密钥(空表示不校验)
}

type SelfHelp struct {
	Id       int    `xorm:"pk autoincr" json:"id"`
	Name     string `xorm:"varchar(200) notnull" json:"name"`
	Way      string `xorm:"varchar(200) notnull" json:"way"`
	Point    string `xorm:"varchar(200) notnull" json:"point"`
	Mode     string `json:"mode"`     // 当前模式: 轮询、随机
	Index    int    `json:"index"`    // 当前映射集的索引(用于轮询模式)
	AliveNum int    `json:"aliveNum"` // 存活的路径数
	Salt     string `xorm:"varchar(200) notnull default('')" json:"salt"` // 请求层 HMAC 密钥(空表示不校验)
}

// 路径
type Url struct {
	Id     int    `xorm:"pk autoincr" json:"id"` // id
	Parent string `xorm:"varchar(200) notnull" json:"parent"`
	Path   string `xorm:"varchar(200) notnull" json:"address"` // 路径
	Alive  bool   `xorm:"notnull" json:"alive"`                // 是否存活
	Retry  int    `xorm:"notnull" json:"retry"`                // 重试次数

	// LastAlertAt 最近一次真正发出了告警消息的时间(NULL=从未告警)。
	// LastAlertIsDown 1=最近一次告警是 DOWN(安全默认: 现有行视为"已 DOWN 过", 防启动风暴)。
	// LastAlertFailCount: DOWN 路径累计次数, UP 路径清零。
	LastAlertAt        *time.Time `json:"lastAlertAt"`
	LastAlertIsDown    bool      `xorm:"notnull default(1)" json:"lastAlertIsDown"`
	LastAlertFailCount int       `xorm:"notnull default(0)" json:"lastAlertFailCount"`

	// 服务器携密钥自申请停用(可指定时间): F1-F6 扩展列
	SelfDeactivateKey      string     `xorm:"varchar(200) notnull default('')" json:"selfDeactivateKey"`
	SelfDeactivateUntil    *time.Time `json:"selfDeactivateUntil"` // 到期时间, NULL = 无在用自申请(xorm 无 notnull → NULLable 列)
	SelfDeactivateAttempts int        `xorm:"notnull default(0)" json:"selfDeactivateAttempts"`
}

type User struct {
	Username string `xorm:"varchar(200) notnull"`
	Password string `xorm:"varchar(200) notnull"`
}

// SystemConfig 系统级运行开关表,只有一行(id=1)。
type SystemConfig struct {
	Id int `xorm:"pk" json:"id"`

	// AllowIntranetProxy 是否允许反代理目标为私有/内网/回环/链路本地地址。
	AllowIntranetProxy bool `xorm:"notnull" json:"allowIntranetProxy"`

	// SelfDefaultCollectionId / AgentDefaultCollectionId 配置空 way= 时的
	// 自助 / 代理模式默认映射集; 0 表示未配置。
	SelfDefaultCollectionId  int `xorm:"notnull default(0)" json:"selfDefaultCollectionId"`
	AgentDefaultCollectionId int `xorm:"notnull default(0)" json:"agentDefaultCollectionId"`

	// ---- 地址失效告警(host / mail 双通道, 防抖) ----

	// AlertEnabled 告警总开关(0=关闭, 关闭时 dispatcher 直接丢弃事件)。
	AlertEnabled bool `xorm:"notnull default(0)" json:"alertEnabled"`

	// AlertDingEnabled / AlertDingWebhook / AlertDingSecret 钉钉群机器人通道配置。
	// Secret 为空表示 plain webhook(不加签); 非空走加签( timestamp + "\n" + secret, HMAC-SHA256 → base64 → urlencode)。
	AlertDingEnabled bool   `xorm:"notnull default(0)" json:"alertDingEnabled"`
	AlertDingWebhook string `xorm:"varchar(500) notnull default('')" json:"alertDingWebhook"`
	AlertDingSecret  string `xorm:"varchar(200) notnull default('')" json:"alertDingSecret"`

	// AlertMailEnabled / AlertSMTP* 邮件(SMTP) 通道配置。
	AlertMailEnabled  bool   `xorm:"notnull default(0)" json:"alertMailEnabled"`
	AlertSMTPHost     string `xorm:"'alert_smtp_host' varchar(200) notnull default('')" json:"alertSmtpHost"`
	AlertSMTPPort     int    `xorm:"'alert_smtp_port' notnull default(25)" json:"alertSmtpPort"`
	AlertSMTPUser     string `xorm:"'alert_smtp_user' varchar(200) notnull default('')" json:"alertSmtpUser"`
	AlertSMTPPassword string `xorm:"'alert_smtp_password' varchar(200) notnull default('')" json:"alertSmtpPassword"`
	AlertSMTPFrom     string `xorm:"'alert_smtp_from' varchar(200) notnull default('')" json:"alertSmtpFrom"`
	AlertSMTPTo       string `xorm:"'alert_smtp_to' varchar(500) notnull default('')" json:"alertSmtpTo"` // 逗号分隔

	// AlertDebounceSec 同一 URL 同一 kind(UP/DOWN)的告警防抖窗口(秒), 默认 600。
	AlertDebounceSec int `xorm:"notnull default(600)" json:"alertDebounceSec"`

	// ---- SSL 证书支持(管理员配置 cert/key 路径起 HTTPS 入口) ----

	// SSL 总开关: 启用后 StartGin 走 HTTPS-only(SslPort), 未启用行为与改造前一致。
	SslEnabled bool `xorm:"notnull default(0)" json:"sslEnabled"`
	// SSL 证书/私钥在宿主机的文件路径(由管理员配置, 容器内通过映射可达)。
	SslCertPath string `xorm:"varchar(500) notnull default('')" json:"sslCertPath"`
	SslKeyPath  string `xorm:"varchar(500) notnull default('')" json:"sslKeyPath"`
	// HTTPS 监听端口, 默认 9443(与 HTTP 9091 避让)。
	SslPort int `xorm:"notnull default(9443)" json:"sslPort"`
}
