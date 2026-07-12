package router

import (
	"com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/util"
	"strings"

	"github.com/gin-gonic/gin"
)

// SaltInjector 对已解析出 way 的请求做 salted-mode 准入校验。
// 当该 way 坐标未配置 salt(空)或 way 不存在时放行(return false,由既有 handler 决定 404)。
// 当 salt 已配置但请求缺少/错误 salt= 参数时,写入 403 并 Abort,返回 true 告知调用方 halt。
func SaltInjector(c *gin.Context, way string) bool {
	if way == "" {
		return false
	}
	salt, _, found := dao.GetSaltForWay(way)
	if !found || salt == "" {
		// 未配置 salt → 保持向后兼容,走透既有流程(含 404 兜底)
		return false
	}
	// HMAC 消息 = 路由剩余段(/agent/*name 的 catch-all; /self/:way 以 way 自身吸收尾部,剩余为空)
	path := strings.TrimPrefix(c.Param("name"), "/")
	expected := util.HMACSHA256Hex(salt, path)
	supplied := c.Query("salt")
	if !strings.EqualFold(supplied, expected) {
		c.JSON(403, gin.H{"code": 1, "msg": "invalid or missing salt"})
		c.Abort()
		return true
	}
	return false
}
