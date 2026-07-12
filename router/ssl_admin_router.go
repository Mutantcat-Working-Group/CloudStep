package router

import (
	"com.mutantcat.cloud_step/dao"
	"com.mutantcat.cloud_step/entity"
	"github.com/gin-gonic/gin"
	"os"
	"strings"
)

// SslAdminRouter 暴露 SSL 配置查询 / 更新端点, 均需登录(LoginHandler)。
type SslAdminRouter struct{}

func (router *SslAdminRouter) PrepareRouter() error { return nil }

func (router *SslAdminRouter) InitRouter(context *gin.Engine) error {
	context.GET("/sslcerts", LoginHandler(), getSslConfig)
	context.POST("/sslcerts/update", LoginHandler(), updateSslConfig)
	return nil
}

func (router *SslAdminRouter) DestroyRouter() error { return nil }

// sslConfigBody 查询 / 提交的统一字段。
type sslConfigBody struct {
	Enabled  bool   `json:"enabled"`
	CertPath string `json:"certPath"`
	KeyPath  string `json:"keyPath"`
	Port     int    `json:"port"`
}

// getSslConfig GET /sslcerts —— 返回当前 SSL 配置(路径原样透出, 仅 admin 可见)。
func getSslConfig(c *gin.Context) {
	cfg := dao.GetSystemConfig()
	c.JSON(200, gin.H{"code": 0, "msg": "success", "data": sslConfigBody{
		Enabled:  cfg.SslEnabled,
		CertPath: cfg.SslCertPath,
		KeyPath:  cfg.SslKeyPath,
		Port:     cfg.SslPort,
	}})
}

// updateSslConfig POST /sslcerts/update —— 写入 SSL 配置。
// 校验: enabled=true 时 certPath/keyPath 必须均非空且文件在磁盘存在, 否则 code:1 不落库。
func updateSslConfig(c *gin.Context) {
	var b sslConfigBody
	if c.ShouldBindJSON(&b) != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	if b.Enabled {
		if strings.TrimSpace(b.CertPath) == "" || strings.TrimSpace(b.KeyPath) == "" {
			c.JSON(200, gin.H{"code": 1, "msg": "cert or key file not found"})
			return
		}
		if _, err := os.Stat(b.CertPath); err != nil {
			c.JSON(200, gin.H{"code": 1, "msg": "cert or key file not found"})
			return
		}
		if _, err := os.Stat(b.KeyPath); err != nil {
			c.JSON(200, gin.H{"code": 1, "msg": "cert or key file not found"})
			return
		}
		// 端口未填写时回落到 9443；明确 0/负数视为未填写(& explicit 0 亦同)。
		if b.Port <= 0 {
			b.Port = 9443
		}
	}

	in := entity.SystemConfig{
		SslEnabled:  b.Enabled,
		SslCertPath: strings.TrimSpace(b.CertPath),
		SslKeyPath:  strings.TrimSpace(b.KeyPath),
		SslPort:     b.Port,
	}
	if dao.UpdateSystemConfig(in) {
		c.JSON(200, gin.H{"code": 0, "msg": "success"})
		return
	}
	c.JSON(200, gin.H{"code": 1, "msg": "error"})
}
