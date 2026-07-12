package router

import (
	"com.mutantcat.cloud_step/dao"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const maxSelfDeactivateDuration = 7200 // 2h, spec F5

// SelfDeactivateRouter 公共端点:服务器携密钥自申请停用/提前恢复;
// 管理端点:查看/rotate 自申请密钥(LoginHandler).
type SelfDeactivateRouter struct{}

func (router *SelfDeactivateRouter) PrepareRouter() error { return nil }

func (router *SelfDeactivateRouter) InitRouter(context *gin.Engine) error {
	context.POST("/self-deactivate", selfDeactivateHandler)
	context.POST("/self-activate", selfActivateHandler)
	context.GET("/self-deactivate/key", LoginHandler(), getKeyHandler)
	context.POST("/self-deactivate/key/rotate", LoginHandler(), rotateKeyHandler)
	return nil
}

func (router *SelfDeactivateRouter) DestroyRouter() error { return nil }

// selfDeactivateHandler POST /self-deactivate {id,key,durationSec}. key-gate.
func selfDeactivateHandler(c *gin.Context) {
	type body struct {
		Id          int    `json:"id"`
		Key         string `json:"key"`
		DurationSec int    `json:"durationSec"`
	}
	var b body
	if err := c.ShouldBindJSON(&b); err != nil || b.Id <= 0 || b.Key == "" {
		c.JSON(200, gin.H{"code": 1, "msg": "invalid param"})
		return
	}
	if b.DurationSec <= 0 || b.DurationSec > maxSelfDeactivateDuration {
		c.JSON(200, gin.H{"code": 1, "msg": "duration must be >0 and <=7200"})
		return
	}
	url, ok := dao.GetUrl(b.Id)
	if !ok {
		c.JSON(200, gin.H{"code": 1, "msg": "invalid key or url"})
		return
	}
	// key 匹配(平等文案防 id 枚举, admin-disabled 也走同一路径)
	if url.SelfDeactivateKey != b.Key {
		c.JSON(200, gin.H{"code": 1, "msg": "invalid key or url"})
		return
	}
	// admin 手动禁用(alive=false 且 until=NULL)拒收自申请停用(equal-text, 403)
	if !url.Alive && url.SelfDeactivateUntil == nil {
		c.JSON(403, gin.H{"code": 1, "msg": "url is administratively disabled; contact admin"})
		return
	}
	until := time.Now().Add(time.Duration(b.DurationSec) * time.Second)
	if !dao.UpdateUrlAlive(b.Id, false) {
		c.JSON(200, gin.H{"code": 1, "msg": "error"})
		return
	}
	if !dao.SetUrlSelfDeactivate(b.Id, until, 0) {
		c.JSON(200, gin.H{"code": 1, "msg": "error"})
		return
	}
	c.JSON(200, gin.H{
		"code":            0,
		"msg":             "ok",
		"deactivateUntil": until.Format(time.RFC3339),
	})
}

// selfActivateHandler POST /self-activate {id,key}. key-gate.
func selfActivateHandler(c *gin.Context) {
	type body struct {
		Id  int    `json:"id"`
		Key string `json:"key"`
	}
	var b body
	if err := c.ShouldBindJSON(&b); err != nil || b.Id <= 0 || b.Key == "" {
		c.JSON(200, gin.H{"code": 1, "msg": "invalid param"})
		return
	}
	url, ok := dao.GetUrl(b.Id)
	if !ok {
		c.JSON(200, gin.H{"code": 1, "msg": "invalid key or url"})
		return
	}
	if url.SelfDeactivateKey != b.Key {
		c.JSON(200, gin.H{"code": 1, "msg": "invalid key or url"})
		return
	}
	// 没有活跃自申请窗口 → 400
	if url.SelfDeactivateUntil == nil {
		c.JSON(400, gin.H{"code": 1, "msg": "url not self-deactivated"})
		return
	}
	if !dao.UpdateUrlAlive(b.Id, true) {
		c.JSON(200, gin.H{"code": 1, "msg": "error"})
		return
	}
	if !dao.ClearUrlSelfDeactivate(b.Id) {
		c.JSON(200, gin.H{"code": 1, "msg": "error"})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok"})
}

// getKeyHandler GET /self-deactivate/key?id=. Login-admin, 按需 seed 密钥.
func getKeyHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil || id <= 0 {
		c.JSON(200, gin.H{"code": 1, "msg": "invalid id"})
		return
	}
	url, ok := dao.GetUrl(id)
	if !ok {
		c.JSON(404, gin.H{"code": 1, "msg": "url not found"})
		return
	}
	seeded := false
	if url.SelfDeactivateKey == "" {
		if _, gerr := dao.GenerateAndSaveUrlKey(id); gerr != nil {
			c.JSON(200, gin.H{"code": 1, "msg": "error"})
			return
		}
		seeded = true
		// 读取 key/url 已 seed 后最新状态
		if url, ok = dao.GetUrl(id); !ok {
			c.JSON(200, gin.H{"code": 1, "msg": "url not found"})
			return
		}
	}
	c.JSON(200, gin.H{
		"code":   0,
		"seeded": seeded,
		"data": gin.H{
			"key":      url.SelfDeactivateKey,
			"attempts": url.SelfDeactivateAttempts,
			"url": gin.H{
				"id":              url.Id,
				"path":            url.Path,
				"alive":           url.Alive,
				"deactivateUntil": url.SelfDeactivateUntil,
			},
		},
	})
}

// rotateKeyHandler POST /self-deactivate/key/rotate?id=. Login-admin, 强制 regen.
func rotateKeyHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil || id <= 0 {
		c.JSON(200, gin.H{"code": 1, "msg": "invalid id"})
		return
	}
	if _, ok := dao.GetUrl(id); !ok {
		c.JSON(404, gin.H{"code": 1, "msg": "url not found"})
		return
	}
	newKey, gerr := dao.RotateUrlKey(id)
	if gerr != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "error"})
		return
	}
	c.JSON(200, gin.H{"code": 0, "data": gin.H{"key": newKey}})
}
