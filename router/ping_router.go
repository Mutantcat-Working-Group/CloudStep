package router

import (
	"com.mutantcat.cloud_step/util"
	"github.com/gin-gonic/gin"
)

type PingRouter struct {
}

func (router *PingRouter) PrepareRouter() error {
	return nil
}

func (router *PingRouter) InitRouter(context *gin.Engine) error {
	context.POST("/ping", ping)
	return nil
}

func (router *PingRouter) DestroyRouter() error {
	return nil
}

func ping(c *gin.Context) {
	type url struct {
		Url string `json:"url"`
	}
	var u url
	err := c.ShouldBindJSON(&u)
	if err != nil {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
	if u.Url == "" {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}

	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "pong",
		"ms":   util.GetTCPSpeed(u.Url),
	})
}
