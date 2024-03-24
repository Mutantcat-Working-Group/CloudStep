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
	context.GET("/collection/delete", LoginHandler(), deleteCollection)
	// 链接的操作
	context.POST("/url/add", LoginHandler(), addUrl)
	context.POST("/url/update", LoginHandler(), updateUrl)
	context.GET("/url/delete", LoginHandler(), deleteUrl)
	// 自助的操作
	context.GET("/selfhelp/get", LoginHandler(), getSelfHelp)
	context.POST("/selfhelp/add", LoginHandler(), addSelfHelp)
	context.POST("/selfhelp/update", LoginHandler(), updateSelfHelp)
	context.GET("/selfhelp/delete", LoginHandler(), deleteSelfHelp)
	// 代理的操作
	context.GET("/proxy/get", LoginHandler(), getProxy)
	context.POST("/proxy/add", LoginHandler(), addProxy)
	context.POST("/proxy/update", LoginHandler(), updateProxy)
	context.GET("/proxy/delete", LoginHandler(), deleteProxy)
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
	b, i := dao.AddCollection(col.Name, col.Urls)
	if b {
		c.JSON(200, gin.H{
			"code": 0,
			"msg":  "success",
			"id":   i,
		})
		return
	}
	c.JSON(200, gin.H{
		"code": 1,
		"msg":  "error",
	})
}

func deleteCollection(c *gin.Context) {
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
	if dao.CheckCollectionDepend(idInt) {
		c.JSON(200, gin.H{
			"code": 2,
			"msg":  "depend",
		})
		return
	}
	b := dao.DeleteCollectionById(idInt)
	if b {
		c.JSON(200, gin.H{
			"code": 0,
			"msg":  "success",
		})
		return
	} else {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
	}
}

func addUrl(c *gin.Context) {
	type Url struct {
		Parent int    `json:"parent"`
		Path   string `json:"address"`
	}
	var url Url
	err := c.ShouldBindJSON(&url)
	if err != nil {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
			"data": Url{Path: "null", Parent: 0},
		})
		return
	}
	name := dao.GetCollectionNameById(url.Parent)
	if !dao.CheckCollectionNameExist(name) {
		c.JSON(200, gin.H{
			"code": 2,
			"msg":  "parent not exist",
		})
		return
	}
	if url.Path == "" {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
	b := dao.AddUrl(name, url.Path)
	if b {
		c.JSON(200, gin.H{
			"code": 0,
			"msg":  "success",
		})
		return
	} else {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}

}

func updateUrl(c *gin.Context) {
	var url entity.Url
	err := c.ShouldBindJSON(&url)
	if err != nil {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
	if url.Path == "" {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
	b := dao.UpdateUrlById(url.Id, url.Path)
	if b {
		c.JSON(200, gin.H{
			"code": 0,
			"msg":  "success",
		})
		return
	} else {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
}

func deleteUrl(c *gin.Context) {
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
	b := dao.DeleteUrlById(idInt)
	if b {
		c.JSON(200, gin.H{
			"code": 0,
			"msg":  "success",
		})
		return
	} else {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
	}
}

func addSelfHelp(c *gin.Context) {
	var selfHelp entity.SelfHelp
	err := c.ShouldBindJSON(&selfHelp)
	if err != nil {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
	if selfHelp.Point == "" {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
	// 检查名称是否存在
	if dao.CheckNameExist(selfHelp.Name) {
		c.JSON(200, gin.H{
			"code": 2,
			"msg":  "name is exist",
		})
		return
	}

	// 检查way是否存在
	if dao.CheckWayExist(selfHelp.Way) {
		c.JSON(200, gin.H{
			"code": 3,
			"msg":  "way is exist",
		})
		return
	}
	b := dao.AddSelfHelp(selfHelp)
	if b {
		c.JSON(200, gin.H{
			"code": 0,
			"msg":  "success",
		})
		return
	} else {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
}

func updateSelfHelp(c *gin.Context) {
	var selfHelp entity.SelfHelp
	err := c.ShouldBindJSON(&selfHelp)
	if err != nil {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
	if selfHelp.Id == 0 {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}

	// 需要判断一下其他的自助是否有相同的way

	b := dao.UpdateSelfHelpById(selfHelp)
	if b {
		c.JSON(200, gin.H{
			"code": 0,
			"msg":  "success",
		})
		return
	} else {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
}

func deleteSelfHelp(c *gin.Context) {
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
	b := dao.DeleteSelfHelpById(idInt)
	if b {
		c.JSON(200, gin.H{
			"code": 0,
			"msg":  "success",
		})
		return
	} else {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
	}
}

func getSelfHelp(c *gin.Context) {
	selfHelps := dao.GetAllSelfHelps()
	if selfHelps == nil {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error or nil",
		})
		return
	}
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "success",
		"data": selfHelps,
	})

}

func getProxy(c *gin.Context) {
	proxys := dao.GetAllProxies()
	if proxys == nil {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error or nil",
		})
		return
	}
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "success",
		"data": proxys,
	})
}

func addProxy(c *gin.Context) {
	var proxy entity.Proxy
	err := c.ShouldBindJSON(&proxy)
	if err != nil {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
	if proxy.Name == "" {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
	if dao.CheckProxyNameExist(proxy.Name) {
		c.JSON(200, gin.H{
			"code": 2,
			"msg":  "name is exist",
		})
		return
	}
	if dao.CheckProxyWayExist(proxy.Way) {
		c.JSON(200, gin.H{
			"code": 3,
			"msg":  "way is exist",
		})
		return
	}
	b := dao.AddProxy(proxy)
	if b {
		c.JSON(200, gin.H{
			"code": 0,
			"msg":  "success",
		})
		return
	} else {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
}

func updateProxy(c *gin.Context) {
	var proxy entity.Proxy
	err := c.ShouldBindJSON(&proxy)
	if err != nil {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
	if proxy.Id == 0 {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}

	// 需要判断一下其他的代理是否有相同的way

	b := dao.UpdateProxyById(proxy)
	if b {
		c.JSON(200, gin.H{
			"code": 0,
			"msg":  "success",
		})
		return
	} else {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
		return
	}
}

func deleteProxy(c *gin.Context) {
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
	b := dao.DeleteProxyById(idInt)
	if b {
		c.JSON(200, gin.H{
			"code": 0,
			"msg":  "success",
		})
		return
	} else {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "error",
		})
	}
}
