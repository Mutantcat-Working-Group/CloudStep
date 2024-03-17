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
	context.Any("/proxy", proxy)
	context.Any("/proxy/:way", proxy)
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

	url := collection.GetPath(way)
	if url == "" {
		c.JSON(200, gin.H{
			"code": 404,
		})
		return
	}

	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "success",
		"way":  way,
		"url":  url,
	})

}
