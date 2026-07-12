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
	context.POST("/url/enable", LoginHandler(), enableUrl)
	context.POST("/url/disable", LoginHandler(), disableUrl)
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
	// 系统开关(内网代理是否放行)
	context.GET("/sysconfig/get", LoginHandler(), getSysConfig)
	context.POST("/sysconfig/update", LoginHandler(), updateSysConfig)
	return nil
}

func (router *SettingRouter) DestroyRouter() error {
	return nil
}

func getSysConfig(c *gin.Context) {
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "success",
		"data": dao.GetSystemConfig(),
	})
}

func updateSysConfig(c *gin.Context) {
	type body struct {
		AllowIntranetProxy      bool `json:"allowIntranetProxy"`
		SelfDefaultCollectionId  int  `json:"selfDefaultCollectionId"`
		AgentDefaultCollectionId int  `json:"agentDefaultCollectionId"`
	}
	var b body
	if c.ShouldBindJSON(&b) != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "参数错误"})
		return
	}
	// 校验 default id 指向存在的 collection(0 表示"清除默认", 合法)。
	if b.SelfDefaultCollectionId > 0 && dao.GetCollectionNameById(b.SelfDefaultCollectionId) == "" {
		c.JSON(200, gin.H{"code": 1, "msg": "参数错误", "desc": "selfDefaultCollectionId not found"})
		return
	}
	if b.AgentDefaultCollectionId > 0 && dao.GetCollectionNameById(b.AgentDefaultCollectionId) == "" {
		c.JSON(200, gin.H{"code": 1, "msg": "参数错误", "desc": "agentDefaultCollectionId not found"})
		return
	}
	// 透传完整三个字段, 避免全量 SystemConfig 镜像把另外两个 default id 零擦。
	if dao.UpdateSystemConfig(entity.SystemConfig{
		AllowIntranetProxy:      b.AllowIntranetProxy,
		SelfDefaultCollectionId:  b.SelfDefaultCollectionId,
		AgentDefaultCollectionId: b.AgentDefaultCollectionId,
	}) {
		c.JSON(200, gin.H{"code": 0, "msg": "success"})
		return
	}
	c.JSON(200, gin.H{"code": 1, "msg": "error"})
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

func enableUrl(c *gin.Context) {
	id, ok := bindUrlId(c)
	if !ok {
		return
	}
	if dao.UpdateUrlAlive(id, true) {
		// spec §5 admin-enable invariant: 管理员 enable 一出, 立刻销毁自申请窗口
		dao.ClearUrlSelfDeactivate(id)
		c.JSON(200, gin.H{"code": 0, "msg": "success"})
		return
	}
	c.JSON(200, gin.H{"code": 1, "msg": "error"})
}

func disableUrl(c *gin.Context) {
	id, ok := bindUrlId(c)
	if !ok {
		return
	}
	if dao.UpdateUrlAlive(id, false) {
		c.JSON(200, gin.H{"code": 0, "msg": "success"})
		return
	}
	c.JSON(200, gin.H{"code": 1, "msg": "error"})
}

// bindUrlId 校验 JSON body 中的 id>0; 校验失败写 {"code":1} 并返回 (0, false).
func bindUrlId(c *gin.Context) (int, bool) {
	type body struct {
		Id int `json:"id"`
	}
	var b body
	if err := c.ShouldBindJSON(&b); err != nil || b.Id <= 0 {
		c.JSON(200, gin.H{"code": 1, "msg": "error"})
		return 0, false
	}
	return b.Id, true
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
	if dao.CheckWayExistExceptId(selfHelp.Way, selfHelp.Id) {
		c.JSON(200, gin.H{
			"code": 2,
			"msg":  "way is exist",
		})
		return
	}

	// 需要判断一下其他的自助是否有相同的name
	if dao.CheckNameExistExceptId(selfHelp.Name, selfHelp.Id) {
		c.JSON(200, gin.H{
			"code": 3,
			"msg":  "name is exist",
		})
		return
	}

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
	if dao.CheckProxyWayExistExceptId(proxy.Way, proxy.Id) {
		c.JSON(200, gin.H{
			"code": 2,
			"msg":  "way is exist",
		})
		return
	}

	// 需要判断一下其他的代理是否有相同的name
	if dao.CheckProxyNameExistExceptId(proxy.Name, proxy.Id) {
		c.JSON(200, gin.H{
			"code": 3,
			"msg":  "name is exist",
		})
		return
	}
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
