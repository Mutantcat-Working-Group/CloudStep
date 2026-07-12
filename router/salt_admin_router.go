package router

import (
	"com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/util"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SaltAdminRouter 提供 salted-mode 的查询/轮换/验证三个管理端点,均需登录(LoginHandler)。
type SaltAdminRouter struct{}

func (router *SaltAdminRouter) PrepareRouter() error { return nil }

func (router *SaltAdminRouter) InitRouter(context *gin.Engine) error {
	context.GET("/salts", LoginHandler(), getSalt)
	context.POST("/salts/rotate", LoginHandler(), rotateSalt)
	context.POST("/salts/verify", LoginHandler(), verifySalt)
	return nil
}

func (router *SaltAdminRouter) DestroyRouter() error { return nil }

// getSalt GET /salts?way=abc —— 查看当前 salt 与所属模式;若为空则自动生成并持久化。
func getSalt(c *gin.Context) {
	way := c.Query("way")
	if way == "" {
		c.JSON(200, gin.H{"code": 1, "msg": "way is required"})
		return
	}
	salt, mode, found := dao.GetSaltForWay(way)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "way not found"})
		return
	}
	if salt == "" {
		// 首次查看时懒生成
		salt, _ = dao.RotateSalt(way, mode)
	}
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{"salt": salt, "mode": mode},
	})
}

// rotateSalt POST /salts/rotate {way,mode} —— 重新生成 salt。
func rotateSalt(c *gin.Context) {
	type body struct {
		Way  string `json:"way" binding:"required"`
		Mode string `json:"mode" binding:"required"`
	}
	var b body
	if c.ShouldBindJSON(&b) != nil || b.Way == "" || b.Mode == "" {
		c.JSON(200, gin.H{"code": 1, "msg": "invalid param"})
		return
	}
	if b.Mode != "self" && b.Mode != "proxy" {
		c.JSON(200, gin.H{"code": 1, "msg": "mode must be self or proxy"})
		return
	}
	newSalt, ok := dao.RotateSalt(b.Way, b.Mode)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "way not found"})
		return
	}
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{"salt": newSalt},
	})
}

// verifySalt POST /salts/verify {way,mode,path,salt} —— 客户端在发真实请求前先校验签出的 hex。
func verifySalt(c *gin.Context) {
	type body struct {
		Way  string `json:"way"`
		Mode string `json:"mode"`
		Path string `json:"path"`
		Salt string `json:"salt"`
	}
	var b body
	if c.ShouldBindJSON(&b) != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "invalid param"})
		return
	}
	salt, _, found := dao.GetSaltForWay(b.Way)
	valid := false
	if found && salt != "" {
		expected := util.HMACSHA256Hex(salt, b.Path)
		valid = strings.EqualFold(expected, b.Salt)
	}
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{"valid": valid},
	})
}
