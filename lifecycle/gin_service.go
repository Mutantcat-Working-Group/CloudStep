package lifecycle

import (
	"com.mutantcat.cloud_step/router"
	"com.mutantcat.cloud_step/util"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// 初始化Gin服务
func InitGin() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	ginServer := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "OPTIONS", "PUT"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Health-Info"}
	ginServer.Use(cors.New(config))
	return ginServer
}

// 启动Gin服务。
// 行为: 读取 util.GetSysConfigMirror(); 若 SSL 总开关启用且 cert/key 路径均非空, 主栈走 HTTPS-only
// (端口 SslPort), 旧 HTTP 客户端需切到 HTTPS 端口; 否则行为与改造前一致(HTTP on port)。
// 注意: 新 SSL 配置仅冷启动生效, 管理员保存后需重启容器/进程。
func StartGin(ginServer *gin.Engine, port string) error {
	cfg := util.GetSysConfigMirror()
	if cfg.SslEnabled && cfg.SslCertPath != "" && cfg.SslKeyPath != "" {
		if err := ginServer.RunTLS(fmt.Sprintf(":%d", cfg.SslPort), cfg.SslCertPath, cfg.SslKeyPath); err != nil {
			return err
		}
		return nil
	}
	if err := ginServer.Run(":" + port); err != nil {
		return err
	}
	return nil
}

// 注册路由
func RegisterRouter(ginServer *gin.Engine, router ...router.RouterTemplate) error {
	for _, r := range router {
		err := r.PrepareRouter()
		if err != nil {
			return err
		}
		err = r.InitRouter(ginServer)
		if err != nil {
			return err
		}
	}
	return nil
}
