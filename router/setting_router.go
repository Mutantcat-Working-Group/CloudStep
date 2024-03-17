package router

import (
	"com.mutantcat.cloud_step/dao"
	"github.com/gin-gonic/gin"
)

type SettingRouter struct {
}

func (router *SettingRouter) PrepareRouter() error {
	return nil
}

func (router *SettingRouter) InitRouter(context *gin.Engine) error {
	context.GET("/collection/getall", LoginHandler(), getAllCollection)
	context.GET("/collection/geturls", LoginHandler(), getAllCollectionUrls)
	context.GET("/collection/add", LoginHandler(), addCollection)
	context.GET("/collection/delete", LoginHandler(), deleteCollection)
	context.GET("/collection/update", LoginHandler(), updateCollection)
	return nil
}

func (router *SettingRouter) DestroyRouter() error {
	return nil
}

func getAllCollection(c *gin.Context) {
	collections := dao.GetAllCollections()
	if collections == nil {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error or nil",
		})
		return
	}
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "success",
		"data": collections,
	})
}

func getAllCollectionUrls(c *gin.Context) {
	collection := c.Query("id")
	if collection == "" {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
}

func addCollection(c *gin.Context) {

}

func deleteCollection(c *gin.Context) {

}

func updateCollection(c *gin.Context) {

}
