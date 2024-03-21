package router

import (
	"com.mutantcat.cloud_step/collection"
	"com.mutantcat.cloud_step/util"
	"github.com/gin-gonic/gin"
	"strings"
)

type ProxyRouter struct {
}

func (router *ProxyRouter) PrepareRouter() error {
	return nil
}

func (router *ProxyRouter) InitRouter(context *gin.Engine) error {
	context.Any("/proxy", proxy)
	context.Any("/proxy/:way", proxy)
	context.Any("/proxy/:way/:method", proxy)
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
	// 获得参数中的请求路径
	method := util.GetMethodParam(c)
	// 变大写
	method = strings.ToUpper(method)
	// 如果没有填写则默认为当前请求的方法
	if method == "" {
		method = c.Request.Method
	}
	err := util.Proxy(path, method, c)
	if err != nil {
		c.JSON(200, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	} else {
		// 返回成功但是不反任何信息
		c.JSON(200, gin.H{})
	}
}
