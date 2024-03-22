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
	context.Any("/re/*name", proxy)
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
	path := collection.GetProxyPath(way)
	if path == "" {
		c.JSON(200, gin.H{
			"code": 404,
		})
		return
	}
	util.Proxy(path, c)
}
