package router

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/util"
	"github.com/gin-gonic/gin"
)

type SelfHelpRouter struct {
}

func (router *SelfHelpRouter) PrepareRouter() error {
	return nil
}

func (router *SelfHelpRouter) InitRouter(context *gin.Engine) error {
	context.Any("/self", selfhelp)
	context.Any("/self/:way", selfhelp)
	return nil
}

func (router *SelfHelpRouter) DestroyRouter() error {
	return nil
}

func selfhelp(c *gin.Context) {
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
