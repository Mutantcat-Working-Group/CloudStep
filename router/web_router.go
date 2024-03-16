package router

import "github.com/gin-gonic/gin"

type WebRouter struct {
}

func (router *WebRouter) PrepareRouter() error {
	return nil
}

func (router *WebRouter) InitRouter(context *gin.Engine) error {
	context.Static("/web", "./web/cloud-step-web-1g/dist")
	context.Static("/assets", "./web/cloud-step-web-1g/dist/assets")
	return nil
}

func (router *WebRouter) DestroyRouter() error {
	return nil
}
