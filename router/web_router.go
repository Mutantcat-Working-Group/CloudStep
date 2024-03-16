package router

import (
	"github.com/gin-gonic/gin"
	"github.com/thinkerou/favicon"
)

type WebRouter struct {
}

func (router *WebRouter) PrepareRouter() error {
	return nil
}

func (router *WebRouter) InitRouter(context *gin.Engine) error {
	context.Use(favicon.New("./web/favicon.jpg"))
	context.Static("/web", "./web/cloud-step-web-1g/dist")
	context.Static("/assets", "./web/cloud-step-web-1g/dist/assets")
	return nil
}

func (router *WebRouter) DestroyRouter() error {
	return nil
}
