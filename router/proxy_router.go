package router

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/util"
	"github.com/gin-gonic/gin"
)

type ProxyRouter struct {
}

func (router *ProxyRouter) PrepareRouter() error {
	return nil
}

func (router *ProxyRouter) InitRouter(context *gin.Engine) error {
	context.POST("/proxy", LoginHandler(), proxy)
	context.POST("/proxy/:way", LoginHandler(), proxy)
	context.POST("/proxy/:way/:method", LoginHandler(), proxy)
	return nil
}

func (router *ProxyRouter) DestroyRouter() error {
	return nil
}

func proxy(c *gin.Context) {
	way := util.GetWayParam(c)
	if way == "" {
		c.JSON(200, gin.H{
			"code": 404,
		})
		return
	}
	method := util.GetMethodParam(c)
	if method == "" {
		c.JSON(200, gin.H{
			"code": 404,
		})
		return
	}
	path := collection.GetProxyPath(way)
	if path == "" {
		c.JSON(200, gin.H{
			"code": 404,
		})
		return
	}
	err := util.Proxy(path, method, c)
	if err != nil {
		c.JSON(200, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}
}
