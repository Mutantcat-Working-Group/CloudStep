package router

import "github.com/gin-gonic/gin"

type LoginRouter struct {
}

func (router *LoginRouter) PrepareRouter() error {
	return nil
}

func (router *LoginRouter) InitRouter(context *gin.Engine) error {

	return nil
}

func (router *LoginRouter) DestroyRouter() error {
	return nil
}
