// CloudStep — 由异猫工作群（mutantcat.org）发行
// GitHub: https://github.com/Mutantcat-Working-Group
package main

import (
	_ "com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/lifecycle"
	"com.mutantcat.cloud_step/router"
	_ "com.mutantcat.cloud_step/scheduler"
)

// version 为当前发布版本号，release 构建可用 -ldflags -X main.version 覆盖。
var version = "1.0.20260920"

func main() {
	gin := lifecycle.InitGin()
	lifecycle.RegisterRouter(gin, &router.WebRouter{},
		&router.LoginRouter{},
		&router.SelfHelpRouter{},
		&router.SelfDeactivateRouter{},
		&router.ProxyRouter{},
		&router.PingRouter{},
		&router.SettingRouter{},
		&router.SelfHelpListRouter{},
		&router.SaltAdminRouter{},
		&router.AlertAdminRouter{},
		&router.SslAdminRouter{},
	)
	lifecycle.StartGin(gin, "9091")
}
