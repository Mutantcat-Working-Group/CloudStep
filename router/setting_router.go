package router

import (
	"com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/entity"
	"github.com/gin-gonic/gin"
	"strconv"
)

type SettingRouter struct {
}

func (router *SettingRouter) PrepareRouter() error {
	return nil
}

func (router *SettingRouter) InitRouter(context *gin.Engine) error {
	// 集合的操作
	context.GET("/collection/getall", LoginHandler(), getAllCollection)
	context.GET("/collection/geturls", LoginHandler(), getAllCollectionUrls)
	context.POST("/collection/add", LoginHandler(), addCollection)
	context.POST("/collection/update", LoginHandler(), updateCollection)
	context.GET("/collection/delete", LoginHandler(), deleteCollection)
	// 链接的操作
	context.POST("/url/add", LoginHandler(), updateCollection)
	context.POST("/url/update", LoginHandler(), updateCollection)
	context.GET("/url/delete", LoginHandler(), updateCollection)
	// 自助的操作
	context.POST("/selfhelp/add", LoginHandler(), updateCollection)
	context.POST("/selfhelp/update", LoginHandler(), updateCollection)
	context.GET("/selfhelp/delete", LoginHandler(), updateCollection)
	// 代理的操作
	context.POST("/proxy/add", LoginHandler(), updateCollection)
	context.POST("/proxy/update", LoginHandler(), updateCollection)
	context.GET("/proxy/delete", LoginHandler(), updateCollection)
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
	id := c.Query("id")
	if id == "" {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
	urls := dao.GetUrlsByParentId(idInt)
	if urls == nil {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error or nil",
		})
		return
	}
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "success",
		"data": urls,
	})
}

func addCollection(c *gin.Context) {
	type collection struct {
		Name string       `json:"name"`
		Urls []entity.Url `json:"urls"`
	}
	var col collection
	err := c.ShouldBindJSON(&col)
	if err != nil {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
	if col.Name == "" {
		c.JSON(200, gin.H{
			"code": 2,
			"msg":  "colname is empty",
		})
		return
	}
	if dao.CheckCollectionNameExist(col.Name) {
		c.JSON(200, gin.H{
			"code": 3,
			"msg":  "colname is exist",
		})
		return
	}
	// 添加成功的入口
	if dao.AddCollection(col.Name, col.Urls) {
		c.JSON(200, gin.H{
			"code": 0,
			"msg":  "success",
		})
		return
	}
	c.JSON(200, gin.H{
		"code": 1,
		"msg":  "error",
	})
}

func deleteCollection(c *gin.Context) {

}

func updateCollection(c *gin.Context) {

}
