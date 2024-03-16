package main

import (
	_ "com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/lifecycle"
	"com.mutantcat.cloud_step/router"
	_ "com.mutantcat.cloud_step/scheduler"
)

func main() {
	gin := lifecycle.InitGin()
	lifecycle.RegisterRouter(gin, &router.WebRouter{})
	lifecycle.StartGin(gin, "9091")
}
